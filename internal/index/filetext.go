package index

import (
	"strings"

	"github.com/bwireman/archivist/internal/store"
)

const fileEmbedLimit = 8192

// FileEmbedText is the plain document embedded for one file.
// Exported symbols come first. The text stops at 8KiB. No symbols is empty.
func FileEmbedText(path string, syms []store.Symbol) string {
	if len(syms) == 0 {
		return ""
	}
	var exported, rest []store.Symbol
	for _, sym := range syms {
		if sym.Exported {
			exported = append(exported, sym)
		} else {
			rest = append(rest, sym)
		}
	}
	var b strings.Builder
	b.WriteString("file: ")
	b.WriteString(path)
	b.WriteByte('\n')
	writeSyms(&b, exported)
	writeSyms(&b, rest)
	return b.String()
}

func writeSyms(b *strings.Builder, syms []store.Symbol) {
	for _, sym := range syms {
		if b.Len() >= fileEmbedLimit {
			return
		}
		line := sym.Name + " (" + sym.Kind + ")"
		if sym.DocLine != "" {
			line += " — " + sym.DocLine
		}
		if b.Len()+1+len(line) > fileEmbedLimit {
			return
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
}
