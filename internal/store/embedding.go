package store

import (
	"encoding/binary"
	"fmt"
	"math"
)

func encodeEmbedding(emb []float32) []byte {
	if len(emb) == 0 {
		return nil
	}
	buf := make([]byte, 4+len(emb)*4)
	binary.LittleEndian.PutUint32(buf[:4], uint32(len(emb)))
	for i, v := range emb {
		binary.LittleEndian.PutUint32(buf[4+i*4:], math.Float32bits(v))
	}
	return buf
}

func decodeEmbedding(buf []byte) ([]float32, error) {
	if len(buf) == 0 {
		return nil, nil
	}
	if len(buf) < 4 {
		return nil, fmt.Errorf("invalid embedding blob")
	}
	n := binary.LittleEndian.Uint32(buf[:4])
	if int(n)*4+4 != len(buf) {
		return nil, fmt.Errorf("embedding size mismatch")
	}
	emb := make([]float32, n)
	for i := range emb {
		emb[i] = math.Float32frombits(binary.LittleEndian.Uint32(buf[4+i*4:]))
	}
	return emb, nil
}

// vectorNorm is the squared L2 norm; cosineSimilarityEncoded takes its square
// root once per query rather than once per candidate.
func vectorNorm(v []float32) float64 {
	var n float64
	for _, x := range v {
		n += float64(x) * float64(x)
	}
	return n
}

func cosineSimilarityEncoded(query []float32, queryNorm float64, buf []byte) float64 {
	if len(query) == 0 || queryNorm == 0 || len(buf) < 4 {
		return 0
	}
	n := int(binary.LittleEndian.Uint32(buf[:4]))
	if n != len(query) || 4+n*4 != len(buf) {
		return 0
	}
	var dot, normB float64
	for i, q := range query {
		qv := float64(q)
		bv := float64(math.Float32frombits(binary.LittleEndian.Uint32(buf[4+i*4:])))
		dot += qv * bv
		normB += bv * bv
	}
	if normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(queryNorm) * math.Sqrt(normB))
}
