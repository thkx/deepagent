package builtin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/thkx/deepagent/tools/builtin/fs"
)

// ExecuteConfig holds configuration for sandbox command execution
type ExecuteConfig struct {
	AllowedCommands        map[string]bool
	Timeout                time.Duration
	MaxOutputSize          int64
	WorkingDir             string
	MaxArgs                int
	MaxArgLength           int
	IsolationMode          string // process | docker | gvisor
	ContainerBinary        string
	ContainerImage         string
	ContainerRuntime       string
	ContainerNetwork       string
	ContainerCPUs          string
	ContainerMemory        string
	ContainerPidsLimit     int
	ReadOnlyRootFS         bool
	ContainerUser          string
	EnforceNonRootUser     bool
	RequireRootless        bool
	SeccompProfile         string
	RequireSeccomp         bool
	RequireImageAllowlist  bool
	AllowedImages          []string
	RequireSignedImages    bool
	CosignBinary           string
	CosignKeyRef           string
	SignatureVerifyTimeout time.Duration
}

// DefaultExecuteConfig returns the default safe configuration
func DefaultExecuteConfig() ExecuteConfig {
	image := getenvOrDefault("DEEPAGENT_SANDBOX_IMAGE", "alpine:3.20")
	allowlist := parseCSVEnv("DEEPAGENT_SANDBOX_ALLOWED_IMAGES")
	if len(allowlist) == 0 {
		allowlist = []string{image}
	}
	return ExecuteConfig{
		AllowedCommands: map[string]bool{
			"ls":       true,
			"cat":      true,
			"echo":     true,
			"pwd":      true,
			"whoami":   true,
			"date":     true,
			"grep":     true,
			"head":     true,
			"tail":     true,
			"wc":       true,
			"sort":     true,
			"uniq":     true,
			"cut":      true,
			"tr":       true,
			"sed":      true,
			"awk":      true,
			"find":     true,
			"test":     true,
			"stat":     true,
			"file":     true,
			"which":    true,
			"dirname":  true,
			"basename": true,
		},
		Timeout:                30 * time.Second,
		MaxOutputSize:          1024 * 1024, // 1MB
		WorkingDir:             ".",
		MaxArgs:                16,
		MaxArgLength:           256,
		IsolationMode:          getenvOrDefault("DEEPAGENT_SANDBOX_MODE", "process"),
		ContainerBinary:        getenvOrDefault("DEEPAGENT_SANDBOX_BIN", "docker"),
		ContainerImage:         image,
		ContainerRuntime:       getenvOrDefault("DEEPAGENT_SANDBOX_RUNTIME", "runsc"),
		ContainerNetwork:       getenvOrDefault("DEEPAGENT_SANDBOX_NETWORK", "none"),
		ContainerCPUs:          getenvOrDefault("DEEPAGENT_SANDBOX_CPUS", "0.5"),
		ContainerMemory:        getenvOrDefault("DEEPAGENT_SANDBOX_MEMORY", "256m"),
		ContainerPidsLimit:     getenvIntOrDefault("DEEPAGENT_SANDBOX_PIDS_LIMIT", 64),
		ReadOnlyRootFS:         true,
		ContainerUser:          getenvOrDefault("DEEPAGENT_SANDBOX_CONTAINER_USER", "65532:65532"),
		EnforceNonRootUser:     getenvBoolOrDefault("DEEPAGENT_SANDBOX_NONROOT", true),
		RequireRootless:        getenvBoolOrDefault("DEEPAGENT_SANDBOX_REQUIRE_ROOTLESS", true),
		SeccompProfile:         strings.TrimSpace(os.Getenv("DEEPAGENT_SANDBOX_SECCOMP_PROFILE")),
		RequireSeccomp:         getenvBoolOrDefault("DEEPAGENT_SANDBOX_REQUIRE_SECCOMP", false),
		RequireImageAllowlist:  getenvBoolOrDefault("DEEPAGENT_SANDBOX_REQUIRE_IMAGE_ALLOWLIST", true),
		AllowedImages:          allowlist,
		RequireSignedImages:    getenvBoolOrDefault("DEEPAGENT_SANDBOX_REQUIRE_SIGNED_IMAGES", false),
		CosignBinary:           getenvOrDefault("DEEPAGENT_SANDBOX_COSIGN_BIN", "cosign"),
		CosignKeyRef:           strings.TrimSpace(os.Getenv("DEEPAGENT_SANDBOX_COSIGN_KEY")),
		SignatureVerifyTimeout: getenvDurationOrDefault("DEEPAGENT_SANDBOX_SIGNATURE_TIMEOUT", 20*time.Second),
	}
}

type SandboxBackend struct {
	inner  fs.Backend
	config ExecuteConfig
}

func NewSandboxBackend(inner fs.Backend) fs.Backend {
	if inner == nil {
		inner = fs.NewInMemoryBackend()
	}
	return &SandboxBackend{
		inner:  inner,
		config: DefaultExecuteConfig(),
	}
}

// NewSandboxBackendWithConfig creates a SandboxBackend with custom configuration
func NewSandboxBackendWithConfig(inner fs.Backend, config ExecuteConfig) fs.Backend {
	if inner == nil {
		inner = fs.NewInMemoryBackend()
	}
	return &SandboxBackend{
		inner:  inner,
		config: config,
	}
}

// === 必须实现 fs.Backend 接口的所有方法 ===

func (s *SandboxBackend) List(ctx context.Context, path string) ([]string, error) {
	return s.inner.List(ctx, path)
}

func (s *SandboxBackend) Read(ctx context.Context, path string) (string, error) {
	return s.inner.Read(ctx, path)
}

func (s *SandboxBackend) Write(ctx context.Context, path, content string) error {
	return s.inner.Write(ctx, path, content)
}

func (s *SandboxBackend) Edit(ctx context.Context, path, instructions string) error {
	return s.inner.Edit(ctx, path, instructions)
}

// === Sandbox 特有方法（供 execute 工具使用）===

func (s *SandboxBackend) Execute(ctx context.Context, command string) (string, error) {
	// 生产环境建议使用 Docker 隔离，这里提供基础安全版本
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("empty command")
	}
	if strings.ContainsAny(command, "|&;><`$\\\n\r") {
		return "", fmt.Errorf("unsafe command syntax detected")
	}

	parts := strings.Fields(command)
	if !s.config.AllowedCommands[parts[0]] {
		return "", fmt.Errorf("command not allowed in sandbox: %s", parts[0])
	}
	if s.config.MaxArgs > 0 && len(parts)-1 > s.config.MaxArgs {
		return "", fmt.Errorf("too many arguments: max %d", s.config.MaxArgs)
	}
	for _, arg := range parts[1:] {
		if s.config.MaxArgLength > 0 && len(arg) > s.config.MaxArgLength {
			return "", fmt.Errorf("argument too long: %q", arg)
		}
		if strings.Contains(arg, "..") {
			return "", fmt.Errorf("unsafe argument detected: %q", arg)
		}
	}

	mode := strings.ToLower(strings.TrimSpace(s.config.IsolationMode))
	if mode == "" {
		mode = "process"
	}

	// Apply timeout if configured
	execCtx := ctx
	if s.config.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
	}

	if mode == "docker" || mode == "gvisor" {
		if err := s.validateContainerPolicy(execCtx); err != nil {
			return "", err
		}
		if err := s.verifyImageSignature(execCtx); err != nil {
			return "", err
		}
	}

	name, args, dir, env, err := s.buildCommandForMode(mode, parts)
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(execCtx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = env
	}
	output, err := cmd.CombinedOutput()

	// Truncate output if it exceeds MaxOutputSize
	if s.config.MaxOutputSize > 0 && int64(len(output)) > s.config.MaxOutputSize {
		output = append(output[:s.config.MaxOutputSize], []byte("\n...output truncated...\n")...)
	}

	if err != nil {
		return string(output), err
	}
	return string(output), nil
}

func (s *SandboxBackend) buildCommandForMode(mode string, parts []string) (name string, args []string, dir string, env []string, err error) {
	switch mode {
	case "process":
		return s.buildProcessCommand(parts)
	case "docker", "gvisor":
		return s.buildContainerCommand(mode, parts)
	default:
		return "", nil, "", nil, fmt.Errorf("unsupported sandbox isolation mode: %s", mode)
	}
}

func (s *SandboxBackend) buildProcessCommand(parts []string) (name string, args []string, dir string, env []string, err error) {
	name = parts[0]
	args = parts[1:]
	if s.config.WorkingDir != "" {
		workDir := s.config.WorkingDir
		if !filepathIsSafe(workDir) {
			return "", nil, "", nil, fmt.Errorf("unsafe working dir configured: %s", workDir)
		}
		dir = workDir
	}
	env = []string{"PATH=/usr/bin:/bin:/usr/sbin:/sbin"}
	return name, args, dir, env, nil
}

func (s *SandboxBackend) buildContainerCommand(mode string, parts []string) (name string, args []string, dir string, env []string, err error) {
	bin := strings.TrimSpace(s.config.ContainerBinary)
	if bin == "" {
		bin = "docker"
	}
	if _, lookErr := exec.LookPath(bin); lookErr != nil {
		return "", nil, "", nil, fmt.Errorf("container runtime binary not found: %s", bin)
	}

	image := strings.TrimSpace(s.config.ContainerImage)
	if image == "" {
		return "", nil, "", nil, fmt.Errorf("container image is required for %s mode", mode)
	}
	if s.config.RequireImageAllowlist && !imageAllowed(image, s.config.AllowedImages) {
		return "", nil, "", nil, fmt.Errorf("container image is not in allowlist: %s", image)
	}

	containerArgs := []string{"run", "--rm"}
	network := strings.TrimSpace(s.config.ContainerNetwork)
	if network == "" {
		network = "none"
	}
	containerArgs = append(containerArgs, "--network", network)

	if s.config.ContainerPidsLimit > 0 {
		containerArgs = append(containerArgs, "--pids-limit", strconv.Itoa(s.config.ContainerPidsLimit))
	}
	if strings.TrimSpace(s.config.ContainerCPUs) != "" {
		containerArgs = append(containerArgs, "--cpus", strings.TrimSpace(s.config.ContainerCPUs))
	}
	if strings.TrimSpace(s.config.ContainerMemory) != "" {
		containerArgs = append(containerArgs, "--memory", strings.TrimSpace(s.config.ContainerMemory))
	}
	if s.config.ReadOnlyRootFS {
		containerArgs = append(containerArgs, "--read-only")
		containerArgs = append(containerArgs, "--tmpfs", "/tmp:rw,noexec,nosuid,size=64m")
	}
	containerArgs = append(containerArgs, "--cap-drop", "ALL")
	containerArgs = append(containerArgs, "--security-opt", "no-new-privileges")
	if s.config.EnforceNonRootUser {
		user := strings.TrimSpace(s.config.ContainerUser)
		if user == "" {
			return "", nil, "", nil, fmt.Errorf("container user is required when EnforceNonRootUser=true")
		}
		containerArgs = append(containerArgs, "--user", user)
	}
	if strings.TrimSpace(s.config.SeccompProfile) != "" {
		profile := strings.TrimSpace(s.config.SeccompProfile)
		if !filepathIsSafe(profile) {
			return "", nil, "", nil, fmt.Errorf("unsafe seccomp profile path: %s", profile)
		}
		absProfile, absErr := filepath.Abs(profile)
		if absErr != nil {
			return "", nil, "", nil, absErr
		}
		containerArgs = append(containerArgs, "--security-opt", "seccomp="+absProfile)
	}

	if mode == "gvisor" {
		runtimeName := strings.TrimSpace(s.config.ContainerRuntime)
		if runtimeName == "" {
			runtimeName = "runsc"
		}
		containerArgs = append(containerArgs, "--runtime", runtimeName)
	}

	if s.config.WorkingDir != "" {
		workDir := s.config.WorkingDir
		if !filepathIsSafe(workDir) {
			return "", nil, "", nil, fmt.Errorf("unsafe working dir configured: %s", workDir)
		}
		absWorkDir, absErr := filepath.Abs(workDir)
		if absErr != nil {
			return "", nil, "", nil, absErr
		}
		containerArgs = append(containerArgs, "-v", fmt.Sprintf("%s:/workspace:ro", absWorkDir))
		containerArgs = append(containerArgs, "-w", "/workspace")
	}

	containerArgs = append(containerArgs, image)
	containerArgs = append(containerArgs, parts...)
	return bin, containerArgs, "", []string{"PATH=/usr/bin:/bin:/usr/sbin:/sbin"}, nil
}

func (s *SandboxBackend) validateContainerPolicy(ctx context.Context) error {
	if s.config.RequireSeccomp && strings.TrimSpace(s.config.SeccompProfile) == "" {
		return fmt.Errorf("seccomp profile is required but not configured")
	}
	if s.config.RequireRootless {
		ok, err := isRootlessDocker(ctx, s.config.ContainerBinary)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("rootless docker is required but not detected")
		}
	}
	return nil
}

func (s *SandboxBackend) verifyImageSignature(ctx context.Context) error {
	if !s.config.RequireSignedImages {
		return nil
	}
	cosignBin := strings.TrimSpace(s.config.CosignBinary)
	if cosignBin == "" {
		cosignBin = "cosign"
	}
	if _, err := exec.LookPath(cosignBin); err != nil {
		return fmt.Errorf("cosign not found: %s", cosignBin)
	}

	verifyCtx := ctx
	if s.config.SignatureVerifyTimeout > 0 {
		var cancel context.CancelFunc
		verifyCtx, cancel = context.WithTimeout(ctx, s.config.SignatureVerifyTimeout)
		defer cancel()
	}

	args := []string{"verify"}
	if strings.TrimSpace(s.config.CosignKeyRef) != "" {
		args = append(args, "--key", strings.TrimSpace(s.config.CosignKeyRef))
	}
	args = append(args, strings.TrimSpace(s.config.ContainerImage))

	cmd := exec.CommandContext(verifyCtx, cosignBin, args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin:/usr/sbin:/sbin"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("image signature verification failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func filepathIsSafe(dir string) bool {
	if dir == "" {
		return false
	}
	if strings.Contains(dir, "..") {
		return false
	}
	if strings.ContainsAny(dir, "\n\r") {
		return false
	}
	if strings.HasPrefix(dir, "~") {
		return false
	}
	_, err := os.Stat(dir)
	return err == nil
}

func getenvOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func getenvIntOrDefault(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getenvBoolOrDefault(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func getenvDurationOrDefault(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func parseCSVEnv(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func imageAllowed(image string, allowlist []string) bool {
	if len(allowlist) == 0 {
		return false
	}
	for _, allowed := range allowlist {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		if strings.HasSuffix(allowed, "*") {
			prefix := strings.TrimSuffix(allowed, "*")
			if strings.HasPrefix(image, prefix) {
				return true
			}
			continue
		}
		if image == allowed {
			return true
		}
	}
	return false
}

func isRootlessDocker(ctx context.Context, dockerBinary string) (bool, error) {
	bin := strings.TrimSpace(dockerBinary)
	if bin == "" {
		bin = "docker"
	}
	if _, err := exec.LookPath(bin); err != nil {
		return false, fmt.Errorf("container runtime binary not found: %s", bin)
	}
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(checkCtx, bin, "info", "--format", "{{json .SecurityOptions}}")
	cmd.Env = []string{"PATH=/usr/bin:/bin:/usr/sbin:/sbin"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to inspect docker rootless mode: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	security := strings.ToLower(string(out))
	return strings.Contains(security, "rootless"), nil
}
