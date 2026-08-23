package main

import (
	"fmt"
	"os"

	"github.com/bwireman/archivist/internal/cmd"
)

func main() {
	if err := cmd.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
