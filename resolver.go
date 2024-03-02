package main

import (
	"encoding/json"
	"flag"
	"log/slog"
	"os"
)

var (
	docsDir   string
	docsIndex string
)

func main() {
	flag.StringVar(&docsDir, "dir", "./docs", "docs root dir")
	flag.StringVar(&docsIndex, "index", "./docs/index.json", "docs index output path")
	flag.Parse()

	idx, err := WalkWithResolver(docsDir)
	if err != nil {
		slog.Error("failed to walk docs dir", slog.String("error", err.Error()))
		return
	}

	os.Remove(docsIndex)

	f, err := os.OpenFile(docsIndex, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return
	}
	defer f.Close()
	e := json.NewEncoder(f)
	e.SetIndent("", "\t")
	e.Encode(idx)
}
