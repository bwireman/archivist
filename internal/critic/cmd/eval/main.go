package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/critic"
	"github.com/bwireman/archivist/internal/store"
)

func main() {
	model := flag.String("model", "gemma4:e2b", "ollama instruct tag")
	report := flag.String("report", "evals/critic/report.md", "report path; does not update the embedded gate in internal/critic/report.md")
	negatives := flag.String("negatives", "internal/critic/negatives.jsonl", "lasting negatives jsonl")
	flag.Parse()
	if err := run(context.Background(), *model, *report, *negatives); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, model, reportPath, negativesPath string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	ollama, err := readOllama(config.GlobalConfigPath())
	if err != nil {
		return err
	}
	base := ollama.ResolvedBaseURL()
	if strings.TrimSpace(model) == strings.TrimSpace(ollama.EmbedModel) {
		return fmt.Errorf("critic model %q is the embed model", model)
	}
	client := &critic.Client{
		BaseURL: base,
		Model:   model,
		HTTP:    &http.Client{Timeout: 3 * time.Minute},
		Prompt:  critic.PromptText(),
		Schema:  critic.SchemaJSON(),
	}
	ok, err := client.HasModel(ctx, model)
	if err != nil {
		return err
	}
	if !ok {
		if err := pull(ctx, base, model); err != nil {
			return err
		}
	}
	db, err := store.OpenReadOnly(config.StorePath(root))
	if err != nil {
		return err
	}
	defer db.Close()
	recs, err := db.AllRecords()
	if err != nil {
		return err
	}
	items := critic.RecordsToItems(recs)
	f, err := os.Open(negativesPath)
	if err != nil {
		return err
	}
	defer f.Close()
	negs, err := critic.LoadNegatives(f)
	if err != nil {
		return err
	}
	items = append(items, negs...)
	n := len(items)
	var i int
	client.Progress = func() {
		i++
		fmt.Fprintf(os.Stderr, "critic %d/%d\n", i, n)
	}
	return critic.Run(ctx, client, ollama.EmbedModel, items, reportPath)
}

func readOllama(path string) (config.OllamaConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config.OllamaConfig{}, nil
		}
		return config.OllamaConfig{}, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return config.OllamaConfig{}, err
	}
	if _, ok := raw["ollama"]; ok {
		var cfg config.Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return config.OllamaConfig{}, err
		}
		return cfg.Ollama, nil
	}
	var ollama config.OllamaConfig
	if err := json.Unmarshal(data, &ollama); err != nil {
		return config.OllamaConfig{}, err
	}
	return ollama, nil
}

func pull(ctx context.Context, base, model string) error {
	cmd := exec.CommandContext(ctx, "ollama", "pull", model)
	cmd.Env = append(os.Environ(), "OLLAMA_HOST="+base)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ollama pull %s: %w: %s", model, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
