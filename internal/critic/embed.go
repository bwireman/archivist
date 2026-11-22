package critic

import "embed"

//go:embed prompt.txt schema.json report.md
var embedded embed.FS

func PromptText() string {
	b, err := embedded.ReadFile("prompt.txt")
	if err != nil {
		panic(err)
	}
	return string(b)
}

func SchemaJSON() []byte {
	b, err := embedded.ReadFile("schema.json")
	if err != nil {
		panic(err)
	}
	return b
}

// PhaseReport is the phase 1 measurement. Enabled reads it.
func PhaseReport() string {
	b, err := embedded.ReadFile("report.md")
	if err != nil {
		panic(err)
	}
	return string(b)
}
