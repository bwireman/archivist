package critic

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// Enabled reads only the lines "type beat: yes|no" and "lasting beat: yes|no".
// A question is included only when its line is yes. No other line enables it.
func Enabled(report string) (keepType, keepLasting bool) {
	for _, line := range strings.Split(report, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "type beat:"):
			keepType = strings.TrimSpace(strings.TrimPrefix(line, "type beat:")) == "yes"
		case strings.HasPrefix(line, "lasting beat:"):
			keepLasting = strings.TrimSpace(strings.TrimPrefix(line, "lasting beat:")) == "yes"
		}
	}
	return keepType, keepLasting
}

// Advisory is the remember payload. A nil pointer means omit it.
type Advisory struct {
	Type    string `json:"type,omitempty"`
	Lasting *bool  `json:"lasting,omitempty"`
	Reason  string `json:"reason"`
}

// RememberAdvisory scores title and body after the record is stored.
// An empty critic model, the embed model, an Ollama error, or unusable JSON
// returns nil. The stored record is left as the caller wrote it.
func RememberAdvisory(ctx context.Context, baseURL, criticModel, embedModel string, httpClient *http.Client, title, body string) *Advisory {
	criticModel = strings.TrimSpace(criticModel)
	embedModel = strings.TrimSpace(embedModel)
	if criticModel == "" || criticModel == embedModel {
		return nil
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 3 * time.Minute}
	}
	client := &Client{
		BaseURL: baseURL,
		Model:   criticModel,
		HTTP:    httpClient,
		Prompt:  PromptText(),
		Schema:  SchemaJSON(),
	}
	res, err := client.Judge(ctx, title, body)
	if err != nil || !res.OK {
		return nil
	}
	keepType, keepLasting := Enabled(PhaseReport())
	if !keepType && !keepLasting {
		return nil
	}
	adv := &Advisory{Reason: res.Reason}
	if keepType {
		adv.Type = res.Type
	}
	if keepLasting {
		adv.Lasting = &res.Lasting
	}
	return adv
}
