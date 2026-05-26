package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/coder/guts"
)

func main() {
	parser, err := guts.NewGolangParser()
	if err != nil {
		log.Fatal(err)
	}

	if err := parser.IncludeGenerate("./internal/api"); err != nil {
		log.Fatal(err)
	}

	ts, err := parser.ToTypescript()
	if err != nil {
		log.Fatal(err)
	}

	out, err := ts.Serialize()
	if err != nil {
		log.Fatal(err)
	}

	path := filepath.Join("web", "src", "lib", "api", "types.generated.ts")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		log.Fatal(err)
	}
}
