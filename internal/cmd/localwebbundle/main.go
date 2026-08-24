package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func main() {
	dist := flag.String("dist", "clients/mobile/dist", "built web asset directory")
	output := flag.String("output", "localweb/web-dist.tar.gz", "output bundle")
	check := flag.Bool("check", false, "fail when the committed bundle is stale")
	flag.Parse()

	bundle, err := build(*dist)
	if err != nil {
		fatal(err)
	}
	if *check {
		current, readErr := os.ReadFile(*output)
		if readErr != nil || !bytes.Equal(current, bundle) {
			fatal(fmt.Errorf("%s is stale; run go run ./internal/cmd/localwebbundle", *output))
		}
		return
	}
	if err := os.WriteFile(*output, bundle, 0o644); err != nil {
		fatal(err)
	}
}

func build(root string) ([]byte, error) {
	var names []string
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		names = append(names, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("web asset directory %s is empty", root)
	}

	var output bytes.Buffer
	gzipWriter, err := gzip.NewWriterLevel(&output, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	gzipWriter.Header.ModTime = time.Unix(0, 0)
	gzipWriter.Header.OS = 255
	tarWriter := tar.NewWriter(gzipWriter)
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			return nil, err
		}
		data, err = normalizeAsset(root, name, data)
		if err != nil {
			return nil, err
		}
		header := &tar.Header{
			Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg,
			ModTime: time.Unix(0, 0), AccessTime: time.Unix(0, 0), ChangeTime: time.Unix(0, 0),
			Format: tar.FormatPAX,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, err
		}
		if _, err := tarWriter.Write(data); err != nil {
			return nil, err
		}
	}
	if err := tarWriter.Close(); err != nil {
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

type viewerAssetManifest struct {
	SchemaVersion int                 `json:"schemaVersion"`
	GeneratedBy   string              `json:"generatedBy"`
	RendererIDs   []string            `json:"rendererIds"`
	CopiedAt      string              `json:"copiedAt"`
	Assets        []viewerAssetRecord `json:"assets"`
}

type viewerAssetRecord struct {
	RendererID    string `json:"rendererId"`
	ID            string `json:"id"`
	To            string `json:"to"`
	Copied        bool   `json:"copied"`
	SourcePackage string `json:"sourcePackage,omitempty"`
	SourceVersion string `json:"sourceVersion,omitempty"`
}

func normalizeAsset(root, name string, data []byte) ([]byte, error) {
	if name != "flyfish-viewer-assets.json" {
		return data, nil
	}
	var manifest viewerAssetManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve web asset root: %w", err)
	}
	manifest.CopiedAt = "1970-01-01T00:00:00Z"
	for index := range manifest.Assets {
		relative, relErr := filepath.Rel(absoluteRoot, manifest.Assets[index].To)
		if relErr != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("%s contains asset outside the web root: %s", name, manifest.Assets[index].To)
		}
		manifest.Assets[index].To = "/" + filepath.ToSlash(relative)
	}
	normalized, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", name, err)
	}
	return append(normalized, '\n'), nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "local web bundle:", strings.TrimSpace(err.Error()))
	os.Exit(1)
}
