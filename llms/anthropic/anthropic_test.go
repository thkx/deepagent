package anthropic

import "testing"

func TestParseToolResultBlock(t *testing.T) {
	block, ok := parseToolResultBlock(`{"tool":"write_file","tool_call_id":"toolu_123","ok":true,"data":{"status":"written"}}`)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if block["type"] != "tool_result" {
		t.Fatalf("expected tool_result block, got: %v", block["type"])
	}
	if block["tool_use_id"] != "toolu_123" {
		t.Fatalf("unexpected tool_use_id: %v", block["tool_use_id"])
	}
}

func TestParseToolResultBlockRequiresToolCallID(t *testing.T) {
	_, ok := parseToolResultBlock(`{"tool":"write_file","ok":true}`)
	if ok {
		t.Fatalf("expected parse failure when tool_call_id missing")
	}
}
