package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/thkx/deepagent/agent"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, func(key string) string {
		return os.Getenv(key)
	}))
}

func run(args []string, stdout io.Writer, stderr io.Writer, getenv func(key string) string) int {
	var file string
	var strict bool
	fs := flag.NewFlagSet("deepagent-audit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&file, "file", "", "path to HITL audit JSONL file")
	fs.BoolVar(&strict, "strict", false, "enable strict validation (required fields and monotonic timestamps)")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	file = strings.TrimSpace(file)
	if file == "" {
		file = strings.TrimSpace(getenv("DEEPAGENT_HITL_AUDIT_FILE"))
	}
	if file == "" {
		fmt.Fprintln(stderr, "error: audit file path is required (use -file or DEEPAGENT_HITL_AUDIT_FILE)")
		return 2
	}
	if !strict {
		strict = strings.EqualFold(strings.TrimSpace(getenv("DEEPAGENT_HITL_AUDIT_STRICT")), "true")
	}

	verifyFn := agent.VerifyHITLAuditFileChain
	if strict {
		verifyFn = agent.VerifyHITLAuditFileChainStrict
	}
	if err := verifyFn(file); err != nil {
		fmt.Fprintf(stderr, "audit chain verification failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "audit chain verification passed: %s\n", file)
	return 0
}
