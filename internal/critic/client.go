package critic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"
)

const reasonRunes = 200

// Result is one critic answer. OK is false when message.content is not the schema.
// A false OK is not an HTTP failure.
type Result struct {
	Type    string
	Lasting bool
	Reason  string
	OK      bool
}

// Client calls Ollama /api/chat. Schema and prompt are the embedded files.
type Client struct {
	BaseURL  string
	Model    string
	HTTP     *http.Client
	Prompt   string
	Schema   json.RawMessage
	Progress func()
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model     string          `json:"model"`
	Stream    bool            `json:"stream"`
	Think     bool            `json:"think"`
	KeepAlive int             `json:"keep_alive"`
	Format    json.RawMessage `json:"format"`
	Options   chatOptions     `json:"options"`
	Messages  []chatMessage   `json:"messages"`
}

type chatOptions struct {
	Temperature float64 `json:"temperature"`
}

type chatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

type answer struct {
	Type    *string `json:"type"`
	Lasting *bool   `json:"lasting"`
	Reason  *string `json:"reason"`
}

// HasModel reports whether /api/tags lists model.
func (c *Client) HasModel(ctx context.Context, model string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/api/tags", nil)
	if err != nil {
		return false, err
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("ollama tags: %s", bytes.TrimSpace(body))
	}
	var out struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, err
	}
	for _, m := range out.Models {
		if m.Name == model || m.Model == model {
			return true, nil
		}
	}
	return false, nil
}

// Judge sends one state. An HTTP or transport error is returned. Content that
// is not the schema returns OK false and a nil error.
func (c *Client) Judge(ctx context.Context, title, body string) (Result, error) {
	if c.Progress != nil {
		c.Progress()
	}
	prompt := strings.TrimRight(c.Prompt, "\n")
	content := prompt + "\n\n" + State(title, body)
	raw := requestBody(c.Model, content, c.Schema)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/api/chat", bytes.NewReader(raw))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http().Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}
	if resp.StatusCode >= 400 {
		return Result{}, fmt.Errorf("ollama chat: %s", bytes.TrimSpace(respBody))
	}
	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Result{}, err
	}
	return parseAnswer(parsed.Message.Content), nil
}

func requestBody(model, content string, schema []byte) []byte {
	var b bytes.Buffer
	b.WriteString(`{"model":`)
	modelJSON, _ := json.Marshal(model)
	b.Write(modelJSON)
	b.WriteString(`,"stream":false,"think":false,"keep_alive":0,"format":`)
	b.Write(schema)
	b.WriteString(`,"options":{"temperature":0},"messages":[{"role":"user","content":`)
	contentJSON, _ := json.Marshal(content)
	b.Write(contentJSON)
	b.WriteString(`}]}`)
	return b.Bytes()
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func parseAnswer(content string) Result {
	dec := json.NewDecoder(strings.NewReader(content))
	dec.DisallowUnknownFields()
	var a answer
	if err := dec.Decode(&a); err != nil {
		return Result{}
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return Result{}
	}
	if a.Type == nil || a.Lasting == nil || a.Reason == nil {
		return Result{}
	}
	switch *a.Type {
	case "decision", "rule", "feature", "guide", "map", "pitfall":
	default:
		return Result{}
	}
	return Result{Type: *a.Type, Lasting: *a.Lasting, Reason: cutReason(*a.Reason), OK: true}
}

func cutReason(s string) string {
	if utf8.RuneCountInString(s) <= reasonRunes {
		return s
	}
	n := 0
	for i := range s {
		if n == reasonRunes {
			return s[:i]
		}
		n++
	}
	return s
}
