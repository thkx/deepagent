package builtin

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/thkx/deepagent/tools/builtin/fs"
)

func TestBuildContainerCommandDocker(t *testing.T) {
	cfg := DefaultExecuteConfig()
	cfg.IsolationMode = "docker"
	cfg.ContainerBinary = "echo"
	cfg.ContainerImage = "alpine:3.20"
	cfg.AllowedImages = []string{"alpine:3.20"}
	cfg.WorkingDir = "."
	sb := NewSandboxBackendWithConfig(fs.NewInMemoryBackend(), cfg).(*SandboxBackend)

	name, args, dir, env, err := sb.buildContainerCommand("docker", []string{"ls", "-la"})
	if err != nil {
		t.Fatalf("buildContainerCommand error: %v", err)
	}
	if name != "echo" {
		t.Fatalf("unexpected binary: %s", name)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "run --rm") {
		t.Fatalf("missing docker run args: %s", joined)
	}
	if !strings.Contains(joined, "--network none") {
		t.Fatalf("missing network isolation: %s", joined)
	}
	if !strings.Contains(joined, "--user 65532:65532") {
		t.Fatalf("missing non-root user enforcement: %s", joined)
	}
	if !strings.Contains(joined, "alpine:3.20 ls -la") {
		t.Fatalf("missing image/command: %s", joined)
	}
	if dir != "" {
		t.Fatalf("container command should not set host dir, got: %s", dir)
	}
	if len(env) == 0 {
		t.Fatalf("expected restricted environment")
	}
}

func TestBuildContainerCommandGVisorAddsRuntime(t *testing.T) {
	cfg := DefaultExecuteConfig()
	cfg.IsolationMode = "gvisor"
	cfg.ContainerBinary = "echo"
	cfg.ContainerRuntime = "runsc"
	cfg.ContainerImage = "alpine:3.20"
	cfg.AllowedImages = []string{"alpine:3.20"}
	cfg.WorkingDir = "."
	sb := NewSandboxBackendWithConfig(fs.NewInMemoryBackend(), cfg).(*SandboxBackend)

	_, args, _, _, err := sb.buildContainerCommand("gvisor", []string{"pwd"})
	if err != nil {
		t.Fatalf("buildContainerCommand error: %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--runtime runsc") {
		t.Fatalf("missing gvisor runtime flag: %s", joined)
	}
}

func TestBuildContainerCommandRejectsImageOutsideAllowlist(t *testing.T) {
	cfg := DefaultExecuteConfig()
	cfg.IsolationMode = "docker"
	cfg.ContainerBinary = "echo"
	cfg.ContainerImage = "busybox:latest"
	cfg.AllowedImages = []string{"alpine:3.20"}
	sb := NewSandboxBackendWithConfig(fs.NewInMemoryBackend(), cfg).(*SandboxBackend)

	_, _, _, _, err := sb.buildContainerCommand("docker", []string{"pwd"})
	if err == nil {
		t.Fatalf("expected image allowlist rejection")
	}
	if !strings.Contains(err.Error(), "not in allowlist") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateContainerPolicyRequiresSeccompProfile(t *testing.T) {
	cfg := DefaultExecuteConfig()
	cfg.IsolationMode = "docker"
	cfg.RequireRootless = false
	cfg.RequireSeccomp = true
	cfg.SeccompProfile = ""
	sb := NewSandboxBackendWithConfig(fs.NewInMemoryBackend(), cfg).(*SandboxBackend)

	err := sb.validateContainerPolicy(context.Background())
	if err == nil {
		t.Fatalf("expected seccomp requirement error")
	}
	if !strings.Contains(err.Error(), "seccomp profile is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyImageSignatureMissingCosign(t *testing.T) {
	cfg := DefaultExecuteConfig()
	cfg.RequireSignedImages = true
	cfg.CosignBinary = "nonexistent-cosign-bin"
	cfg.ContainerImage = "alpine:3.20"
	sb := NewSandboxBackendWithConfig(fs.NewInMemoryBackend(), cfg).(*SandboxBackend)

	err := sb.verifyImageSignature(context.Background())
	if err == nil {
		t.Fatalf("expected cosign missing error")
	}
	if !strings.Contains(err.Error(), "cosign not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildContainerCommandAddsSeccompProfile(t *testing.T) {
	seccompFile, err := os.CreateTemp("", "seccomp-*.json")
	if err != nil {
		t.Fatalf("create temp seccomp file: %v", err)
	}
	defer os.Remove(seccompFile.Name())
	_ = seccompFile.Close()

	cfg := DefaultExecuteConfig()
	cfg.IsolationMode = "docker"
	cfg.ContainerBinary = "echo"
	cfg.ContainerImage = "alpine:3.20"
	cfg.AllowedImages = []string{"alpine:3.20"}
	cfg.SeccompProfile = seccompFile.Name()
	cfg.RequireRootless = false
	sb := NewSandboxBackendWithConfig(fs.NewInMemoryBackend(), cfg).(*SandboxBackend)

	_, args, _, _, err := sb.buildContainerCommand("docker", []string{"pwd"})
	if err != nil {
		t.Fatalf("buildContainerCommand error: %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "seccomp=") {
		t.Fatalf("expected seccomp profile in args: %s", joined)
	}
}

func TestExecuteRejectsUnknownIsolationMode(t *testing.T) {
	cfg := DefaultExecuteConfig()
	cfg.IsolationMode = "unknown-mode"
	sb := NewSandboxBackendWithConfig(fs.NewInMemoryBackend(), cfg).(*SandboxBackend)

	_, err := sb.Execute(context.Background(), "pwd")
	if err == nil {
		t.Fatalf("expected error for unknown isolation mode")
	}
	if !strings.Contains(err.Error(), "unsupported sandbox isolation mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}
