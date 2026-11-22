package critic

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bwireman/archivist/internal/record"
)

// Item is one scored text. ScoreType is false for lasting-only negatives.
type Item struct {
	Title       string
	Body        string
	GoldType    string
	GoldLasting bool
	ScoreType   bool
}

// Summary is the phase 1 report.
type Summary struct {
	Model           string
	NType           int
	NLasting        int
	TypeAccuracy    float64
	TypeMajority    float64
	TypeBeat        bool
	LastingAccuracy float64
	LastingMajority float64
	LastingBeat     bool
	PromptSHA256    string
	SchemaSHA256    string
}

// Measure scores items. A judge error aborts. An unusable answer counts as wrong.
func Measure(ctx context.Context, items []Item, judge func(context.Context, string, string) (Result, error)) (Summary, error) {
	var typeCorrect, typeN, lastingCorrect, lastingN int
	typeGold := make([]string, 0, len(items))
	lastingGold := make([]string, 0, len(items))
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return Summary{}, err
		}
		got, err := judge(ctx, item.Title, item.Body)
		if err != nil {
			return Summary{}, err
		}
		if item.ScoreType {
			typeN++
			typeGold = append(typeGold, item.GoldType)
			if got.OK && got.Type == item.GoldType {
				typeCorrect++
			}
		}
		lastingN++
		if item.GoldLasting {
			lastingGold = append(lastingGold, "true")
		} else {
			lastingGold = append(lastingGold, "false")
		}
		if got.OK && got.Lasting == item.GoldLasting {
			lastingCorrect++
		}
	}
	s := Summary{
		NType:           typeN,
		NLasting:        lastingN,
		TypeAccuracy:    ratio(typeCorrect, typeN),
		TypeMajority:    majority(typeGold),
		LastingAccuracy: ratio(lastingCorrect, lastingN),
		LastingMajority: majority(lastingGold),
	}
	s.TypeBeat = s.TypeAccuracy > s.TypeMajority
	s.LastingBeat = s.LastingAccuracy > s.LastingMajority
	return s, nil
}

func ratio(correct, n int) float64 {
	if n == 0 {
		return 0
	}
	return float64(correct) / float64(n)
}

func majority(labels []string) float64 {
	if len(labels) == 0 {
		return 0
	}
	counts := map[string]int{}
	best := 0
	for _, label := range labels {
		counts[label]++
		if counts[label] > best {
			best = counts[label]
		}
	}
	return float64(best) / float64(len(labels))
}

// Negative is one line of evals/critic/negatives.jsonl.
type Negative struct {
	Class string `json:"class"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// LoadNegatives reads lasting-only items. Gold lasting is false. They are not scored on type.
func LoadNegatives(r io.Reader) ([]Item, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var out []Item
	line := 0
	for sc.Scan() {
		line++
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			continue
		}
		var n Negative
		if err := json.Unmarshal([]byte(text), &n); err != nil {
			return nil, fmt.Errorf("negatives line %d: %w", line, err)
		}
		switch n.Class {
		case "code", "transcript", "error", "catalog", "restatement":
		default:
			return nil, fmt.Errorf("negatives line %d: unknown class %q", line, n.Class)
		}
		if strings.TrimSpace(n.Title) == "" || strings.TrimSpace(n.Body) == "" {
			return nil, fmt.Errorf("negatives line %d: empty title or body", line)
		}
		out = append(out, Item{Title: n.Title, Body: n.Body, GoldLasting: false, ScoreType: false})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// RecordsToItems keeps current rows with a title and a body.
// Deprecated and superseded rows are left out. Gold lasting is true.
func RecordsToItems(recs []*record.Record) []Item {
	var out []Item
	for _, rec := range recs {
		if rec == nil || rec.Retired() {
			continue
		}
		if strings.TrimSpace(rec.Title) == "" || strings.TrimSpace(rec.Body) == "" {
			continue
		}
		out = append(out, Item{
			Title:       rec.Title,
			Body:        rec.Body,
			GoldType:    string(rec.Type),
			GoldLasting: true,
			ScoreType:   true,
		})
	}
	return out
}

// FormatReport writes the phase 1 headings in order.
func FormatReport(s Summary) string {
	return fmt.Sprintf(`model: %s
n_type: %d
n_lasting: %d
type_accuracy: %.6f
type_majority: %.6f
type beat: %s
lasting_accuracy: %.6f
lasting_majority: %.6f
lasting beat: %s
prompt_sha256: %s
schema_sha256: %s
`, s.Model, s.NType, s.NLasting, s.TypeAccuracy, s.TypeMajority, beat(s.TypeBeat), s.LastingAccuracy, s.LastingMajority, beat(s.LastingBeat), s.PromptSHA256, s.SchemaSHA256)
}

func beat(yes bool) string {
	if yes {
		return "yes"
	}
	return "no"
}

func fileSHA256(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Run measures items and writes the report only after every judge call succeeds
// and the model is listed. A missing model returns an error and does not create reportPath.
func Run(ctx context.Context, client *Client, embedModel string, items []Item, reportPath string) error {
	if client == nil {
		return fmt.Errorf("critic client is nil")
	}
	if strings.TrimSpace(client.Model) == "" {
		return fmt.Errorf("critic model is empty")
	}
	if client.Model == strings.TrimSpace(embedModel) {
		return fmt.Errorf("critic model %q is the embed model", client.Model)
	}
	ok, err := client.HasModel(ctx, client.Model)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("ollama has no model %q", client.Model)
	}
	summary, err := Measure(ctx, items, client.Judge)
	if err != nil {
		return err
	}
	summary.Model = client.Model
	summary.PromptSHA256 = fileSHA256([]byte(client.Prompt))
	summary.SchemaSHA256 = fileSHA256(client.Schema)
	return os.WriteFile(reportPath, []byte(FormatReport(summary)), 0o644)
}
