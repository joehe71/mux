package gateway

import (
	"encoding/json"
	"testing"
)

func TestPrepareCodexBody(t *testing.T) {
	body := prepareCodexBody([]byte(`{"model":"GPT-5.4 mini","max_output_tokens":1,"input":"hi"}`))
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode prepared body: %v", err)
	}
	if got := payload["model"]; got != "gpt-5.4-mini" {
		t.Fatalf("model = %v, want gpt-5.4-mini", got)
	}
	if _, exists := payload["max_output_tokens"]; exists {
		t.Fatal("max_output_tokens should be removed")
	}
	if payload["stream"] != true || payload["store"] != false {
		t.Fatalf("stream/store defaults = %v/%v", payload["stream"], payload["store"])
	}
	input, ok := payload["input"].([]any)
	if !ok || len(input) != 1 {
		t.Fatalf("input = %#v, want one message", payload["input"])
	}
}
