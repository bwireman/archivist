package chunk

import (
	"path/filepath"
	"strings"
)

// SplitFile chooses tree-sitter or generic chunking based on file extension.
func SplitFile(path, content string, repoADR, globalADR []string) []Chunk {
	chunkType, scope := Classify(path, repoADR, globalADR)
	var chunks []Chunk
	if chunkType == TypeDoc || chunkType == TypeADR {
		chunks = SplitGeneric(path, content, chunkType, defaultMaxChunkSize, defaultOverlap)
	} else {
		ext := strings.ToLower(filepath.Ext(path))
		if _, ok := languageSpecs[ext]; ok {
			chunks = SplitWithTreeSitter(path, content, chunkType)
		} else {
			chunks = SplitGeneric(path, content, chunkType, defaultMaxChunkSize, defaultOverlap)
		}
	}
	StampADRScope(chunks, scope)
	return chunks
}
