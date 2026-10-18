package retrieve

import "strings"

const qwenQueryPrefix = "Instruct: Given a question about a software project's archive, retrieve the relevant record\nQuery:"

// QueryText prefixes a search query for Qwen3 embedding models. Other models,
// and an empty model name, keep the raw query. Document embeds do not use this.
func QueryText(model, query string) string {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "qwen3-embedding") {
		return query
	}
	return qwenQueryPrefix + query
}
