package chunk

import (
	"path/filepath"
	"strings"
)

// SplitFile chooses tree-sitter or generic chunking based on file extension.
func SplitFile(path, content string, adrPatterns []string) []Chunk {
	chunkType := ClassifyFile(path, adrPatterns)
	if chunkType == TypeDoc || chunkType == TypeADR {
		return SplitGeneric(path, content, chunkType, defaultMaxChunkSize, defaultOverlap)
	}

	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := languageSpecs[ext]; ok {
		return SplitWithTreeSitter(path, content, chunkType)
	}
	return SplitGeneric(path, content, chunkType, defaultMaxChunkSize, defaultOverlap)
}
