package localweb

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"path"
	"strings"
)

//go:embed web-dist.tar.gz
var bundledWebDist []byte

type webAsset struct {
	contentType string
	data        []byte
}

func loadBundledAssets() (map[string]webAsset, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(bundledWebDist))
	if err != nil {
		return nil, fmt.Errorf("open local web asset bundle: %w", err)
	}
	defer gzipReader.Close()

	assets := make(map[string]webAsset)
	reader := tar.NewReader(gzipReader)
	for {
		header, nextErr := reader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return nil, fmt.Errorf("read local web asset bundle: %w", nextErr)
		}
		name := strings.TrimPrefix(path.Clean("/"+header.Name), "/")
		if header.Typeflag != tar.TypeReg || name == "" || strings.HasPrefix(name, "../") {
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, header.Size+1))
		if readErr != nil {
			return nil, fmt.Errorf("read local web asset %q: %w", name, readErr)
		}
		if int64(len(data)) != header.Size {
			return nil, fmt.Errorf("read local web asset %q: size mismatch", name)
		}
		assets[name] = webAsset{contentType: contentType(name), data: data}
	}
	if len(assets["index.html"].data) == 0 {
		return nil, fmt.Errorf("local web asset bundle is missing index.html")
	}
	return assets, nil
}

func contentType(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".json", ".map":
		return "application/json; charset=utf-8"
	case ".txt", ".md":
		return "text/plain; charset=utf-8"
	case ".wasm":
		return "application/wasm"
	case ".woff2":
		return "font/woff2"
	case ".woff":
		return "font/woff"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}
