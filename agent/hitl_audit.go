package agent

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type HITLAuditEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	Event      string    `json:"event"`
	Tool       string    `json:"tool,omitempty"`
	ToolCallID string    `json:"tool_call_id,omitempty"`
	ThreadID   string    `json:"thread_id,omitempty"`
	RequestID  string    `json:"request_id,omitempty"`
	Iteration  int       `json:"iteration,omitempty"`
	Arguments  any       `json:"arguments,omitempty"`
	ArgsHash   string    `json:"args_hash,omitempty"`
	Decision   string    `json:"decision,omitempty"`
	Approved   bool      `json:"approved,omitempty"`
	Error      string    `json:"error,omitempty"`
	PrevHash   string    `json:"prev_hash,omitempty"`
	Hash       string    `json:"hash,omitempty"`
}

type HITLAuditLogger interface {
	LogHITLEvent(entry HITLAuditEntry) error
}

type FileHITLAuditLogger struct {
	path     string
	mu       sync.Mutex
	loaded   bool
	lastHash string
}

func NewFileHITLAuditLogger(path string) (*FileHITLAuditLogger, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	return &FileHITLAuditLogger{path: path}, nil
}

func (l *FileHITLAuditLogger) LogHITLEvent(entry HITLAuditEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.loadLastHashLocked(); err != nil {
		return err
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	entry.PrevHash = l.lastHash
	entry.Hash = computeHITLAuditHash(entry)

	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	l.lastHash = entry.Hash
	return nil
}

func (l *FileHITLAuditLogger) loadLastHashLocked() error {
	if l.loaded {
		return nil
	}
	l.loaded = true
	f, err := os.Open(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	var lastLine string
	for sc.Scan() {
		lastLine = sc.Text()
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if lastLine == "" {
		return nil
	}
	var e HITLAuditEntry
	if err := json.Unmarshal([]byte(lastLine), &e); err != nil {
		return err
	}
	l.lastHash = e.Hash
	return nil
}

func computeHITLAuditHash(entry HITLAuditEntry) string {
	payload := struct {
		Timestamp  string `json:"timestamp"`
		Event      string `json:"event"`
		Tool       string `json:"tool,omitempty"`
		ToolCallID string `json:"tool_call_id,omitempty"`
		ThreadID   string `json:"thread_id,omitempty"`
		RequestID  string `json:"request_id,omitempty"`
		Iteration  int    `json:"iteration,omitempty"`
		Arguments  any    `json:"arguments,omitempty"`
		ArgsHash   string `json:"args_hash,omitempty"`
		Decision   string `json:"decision,omitempty"`
		Approved   bool   `json:"approved,omitempty"`
		Error      string `json:"error,omitempty"`
		PrevHash   string `json:"prev_hash,omitempty"`
	}{
		Timestamp:  entry.Timestamp.UTC().Format(time.RFC3339Nano),
		Event:      entry.Event,
		Tool:       entry.Tool,
		ToolCallID: entry.ToolCallID,
		ThreadID:   entry.ThreadID,
		RequestID:  entry.RequestID,
		Iteration:  entry.Iteration,
		Arguments:  entry.Arguments,
		ArgsHash:   entry.ArgsHash,
		Decision:   entry.Decision,
		Approved:   entry.Approved,
		Error:      entry.Error,
		PrevHash:   entry.PrevHash,
	}
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum)
}

func VerifyHITLAuditFileChain(path string) error {
	return verifyHITLAuditFileChain(path, false)
}

func VerifyHITLAuditFileChainStrict(path string) error {
	return verifyHITLAuditFileChain(path, true)
}

func verifyHITLAuditFileChain(path string, strict bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	prevHash := ""
	var prevTS time.Time
	line := 0
	for sc.Scan() {
		line++
		text := sc.Text()
		if text == "" {
			continue
		}
		var e HITLAuditEntry
		if err := json.Unmarshal([]byte(text), &e); err != nil {
			return fmt.Errorf("invalid json on line %d: %w", line, err)
		}
		if e.PrevHash != prevHash {
			return fmt.Errorf("hash chain mismatch on line %d: prev_hash=%q expected=%q", line, e.PrevHash, prevHash)
		}
		expected := computeHITLAuditHash(HITLAuditEntry{
			Timestamp:  e.Timestamp,
			Event:      e.Event,
			Tool:       e.Tool,
			ToolCallID: e.ToolCallID,
			ThreadID:   e.ThreadID,
			RequestID:  e.RequestID,
			Iteration:  e.Iteration,
			Arguments:  e.Arguments,
			ArgsHash:   e.ArgsHash,
			Decision:   e.Decision,
			Approved:   e.Approved,
			Error:      e.Error,
			PrevHash:   e.PrevHash,
		})
		if e.Hash != expected {
			return fmt.Errorf("entry hash mismatch on line %d", line)
		}
		if strict {
			if err := validateHITLAuditEntryStrict(e, prevTS); err != nil {
				return fmt.Errorf("strict validation failed on line %d: %w", line, err)
			}
		}
		prevHash = e.Hash
		prevTS = e.Timestamp
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return nil
}

func validateHITLAuditEntryStrict(e HITLAuditEntry, prevTS time.Time) error {
	if e.Event == "" {
		return fmt.Errorf("event is required")
	}
	if e.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}
	if !prevTS.IsZero() && e.Timestamp.Before(prevTS) {
		return fmt.Errorf("timestamp is not monotonic")
	}

	switch e.Event {
	case "hitl_request", "hitl_decision":
		if e.Tool == "" {
			return fmt.Errorf("tool is required for %s", e.Event)
		}
		if e.ToolCallID == "" {
			return fmt.Errorf("tool_call_id is required for %s", e.Event)
		}
		if e.ThreadID == "" {
			return fmt.Errorf("thread_id is required for %s", e.Event)
		}
		if e.RequestID == "" {
			return fmt.Errorf("request_id is required for %s", e.Event)
		}
		if e.Iteration <= 0 {
			return fmt.Errorf("iteration must be > 0 for %s", e.Event)
		}
	}
	if e.Event == "hitl_decision" && e.Decision == "" {
		return fmt.Errorf("decision is required for hitl_decision")
	}
	return nil
}
