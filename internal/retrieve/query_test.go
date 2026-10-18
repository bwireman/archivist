package retrieve

import "testing"

func TestQueryText(t *testing.T) {
	const query = "hybrid"
	cases := []struct {
		model string
		want  string
	}{
		{"qwen3-embedding:8b", qwenQueryPrefix + query},
		{"qwen3-embedding:0.6b", qwenQueryPrefix + query},
		{"mxbai-embed-large", query},
		{"", query},
	}
	for _, tc := range cases {
		if got := QueryText(tc.model, query); got != tc.want {
			t.Errorf("QueryText(%q) = %q, want %q", tc.model, got, tc.want)
		}
	}
}
