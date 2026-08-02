package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const upstreamPath = "/backend-api/codex/responses"

type Credential struct {
	AccountID   string
	AccessToken string
}

type Selector func(ctx context.Context, model string) (Credential, error)

type Logger interface {
	Error(message string, attrs ...slog.Attr)
}

type Gateway struct {
	server   *http.Server
	selector Selector
	logger   Logger
}

func New(port int, selector Selector, logger Logger) *Gateway {
	gateway := &Gateway{selector: selector, logger: logger}
	gateway.server = &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", port),
		Handler:           http.HandlerFunc(gateway.handle),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return gateway
}

func (g *Gateway) Start() error { return g.server.ListenAndServe() }

func (g *Gateway) Close(ctx context.Context) error { return g.server.Shutdown(ctx) }

func (g *Gateway) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		writeCORS(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 16<<20))
	if err != nil {
		http.Error(w, "read request failed", http.StatusBadRequest)
		return
	}
	model, err := requestModel(r, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	credential, err := g.selector(r.Context(), model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	body = prepareCodexBody(body)
	upstream, _ := url.Parse("https://chatgpt.com" + upstreamPath)
	request, err := http.NewRequestWithContext(r.Context(), r.Method, upstream.String(), bytes.NewReader(body))
	if err != nil {
		http.Error(w, "create upstream request failed", http.StatusBadGateway)
		return
	}
	request.Header = r.Header.Clone()
	request.Header.Del("Authorization")
	request.Header.Set("Authorization", "Bearer "+credential.AccessToken)
	request.Header.Set("chatgpt-account-id", credential.AccountID)
	request.Header.Set("originator", "pi")
	request.Header.Set("OpenAI-Beta", "responses=experimental")
	request.Header.Set("Accept", "text/event-stream")
	//nolint:gosec // the upstream URL is a fixed constant for the Codex gateway.
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode >= http.StatusBadRequest {
		upstreamBody, readErr := io.ReadAll(io.LimitReader(response.Body, 64<<10))
		if g.logger != nil {
			g.logger.Error("codex upstream request failed", slog.Int("status", response.StatusCode), slog.String("body", string(upstreamBody)), slog.Any("read_error", readErr))
		}
		writeCORS(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(response.StatusCode)
		_, _ = w.Write(upstreamBody)
		return
	}
	writeCORS(w)
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, response.Body)
}

func prepareCodexBody(body []byte) []byte {
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return body
	}
	if _, ok := payload["stream"]; !ok {
		payload["stream"] = true
	}
	if _, ok := payload["store"]; !ok {
		payload["store"] = false
	}
	if _, ok := payload["instructions"]; !ok {
		payload["instructions"] = "You are a helpful assistant."
	}
	if text, ok := payload["input"].(string); ok {
		payload["input"] = []any{map[string]any{
			"role": "user",
			"content": []any{map[string]any{
				"type": "input_text",
				"text": text,
			}},
		}}
	}
	prepared, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return prepared
}

func requestModel(r *http.Request, body []byte) (string, error) {
	model := strings.TrimSpace(r.Header.Get("x-model"))
	if model == "" {
		var payload struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return "", fmt.Errorf("decode request model: %w", err)
		}
		model = strings.TrimSpace(payload.Model)
	}
	if model == "" {
		return "", fmt.Errorf("model is required")
	}
	return model, nil
}

func writeCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
}
