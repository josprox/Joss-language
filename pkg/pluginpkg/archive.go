package pluginpkg

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"
)

const (
	FormatVersion  = 2
	MetadataPath   = "META-INF/joss-plugin.json"
	SymbolsPath    = "META-INF/joss-symbols.json"
	MaxArchiveSize = 256 << 20
	MaxFileSize    = 128 << 20
)

type Metadata struct {
	Format       int               `json:"format"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Bytecode     string            `json:"bytecode"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
	Native       map[string]string `json:"native,omitempty"` // os-arch -> executable path
	Protocol     string            `json:"protocol,omitempty"`
	Symbols      string            `json:"symbols,omitempty"`
}

type Archive struct {
	Metadata Metadata
	Files    map[string][]byte
}

func Build(metadata Metadata, files map[string][]byte) ([]byte, error) {
	metadata.Format = FormatVersion
	if metadata.Name == "" || metadata.Version == "" || metadata.Bytecode == "" {
		return nil, fmt.Errorf("jp: metadata requiere name, version y bytecode")
	}
	if _, ok := files[metadata.Bytecode]; !ok {
		return nil, fmt.Errorf("jp: falta bytecode %q", metadata.Bytecode)
	}
	if metadata.Symbols != "" {
		if _, ok := files[metadata.Symbols]; !ok {
			return nil, fmt.Errorf("jp: falta indice de simbolos %q", metadata.Symbols)
		}
	}
	metaJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, err
	}
	all := make(map[string][]byte, len(files)+1)
	for name, content := range files {
		clean, err := cleanArchivePath(name)
		if err != nil {
			return nil, err
		}
		all[clean] = content
	}
	all[MetadataPath] = metaJSON

	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)
	var buffer bytes.Buffer
	zw := zip.NewWriter(&buffer)
	fixedTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, name := range names {
		content := all[name]
		if len(content) > MaxFileSize {
			_ = zw.Close()
			return nil, fmt.Errorf("jp: %s excede %d MiB", name, MaxFileSize>>20)
		}
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetModTime(fixedTime)
		header.SetMode(0644)
		if isNativeTarget(metadata.Native, name) {
			header.SetMode(0755)
		}
		writer, err := zw.CreateHeader(header)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		if _, err := writer.Write(content); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	if buffer.Len() > MaxArchiveSize {
		return nil, fmt.Errorf("jp: el paquete excede %d MiB", MaxArchiveSize>>20)
	}
	return buffer.Bytes(), nil
}

func Read(data []byte) (*Archive, error) {
	if len(data) > MaxArchiveSize {
		return nil, fmt.Errorf("jp: el paquete excede %d MiB", MaxArchiveSize>>20)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("jp: ZIP v2 invalido: %w", err)
	}
	files := make(map[string][]byte, len(zr.File))
	total := 0
	for _, file := range zr.File {
		clean, err := cleanArchivePath(file.Name)
		if err != nil {
			return nil, err
		}
		if _, duplicate := files[clean]; duplicate {
			return nil, fmt.Errorf("jp: entrada duplicada %q", clean)
		}
		if file.UncompressedSize64 > MaxFileSize {
			return nil, fmt.Errorf("jp: %s excede %d MiB", clean, MaxFileSize>>20)
		}
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		content, readErr := io.ReadAll(io.LimitReader(reader, MaxFileSize+1))
		closeErr := reader.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(content) > MaxFileSize {
			return nil, fmt.Errorf("jp: %s excede %d MiB", clean, MaxFileSize>>20)
		}
		total += len(content)
		if total > MaxArchiveSize {
			return nil, fmt.Errorf("jp: contenido expandido excede %d MiB", MaxArchiveSize>>20)
		}
		files[clean] = content
	}
	metaData, ok := files[MetadataPath]
	if !ok {
		return nil, fmt.Errorf("jp: falta %s", MetadataPath)
	}
	var metadata Metadata
	if err := json.Unmarshal(metaData, &metadata); err != nil {
		return nil, fmt.Errorf("jp: metadata invalida: %w", err)
	}
	if metadata.Format != FormatVersion {
		return nil, fmt.Errorf("jp: version de formato %d no soportada", metadata.Format)
	}
	if metadata.Name == "" || metadata.Version == "" || metadata.Bytecode == "" {
		return nil, fmt.Errorf("jp: metadata incompleta")
	}
	if _, ok := files[metadata.Bytecode]; !ok {
		return nil, fmt.Errorf("jp: falta bytecode %q", metadata.Bytecode)
	}
	if metadata.Symbols != "" {
		if _, ok := files[metadata.Symbols]; !ok {
			return nil, fmt.Errorf("jp: falta indice de simbolos %q", metadata.Symbols)
		}
	}
	for target, executable := range metadata.Native {
		if strings.TrimSpace(target) == "" {
			return nil, fmt.Errorf("jp: target nativo vacio")
		}
		if _, ok := files[executable]; !ok {
			return nil, fmt.Errorf("jp: target %s apunta a %q inexistente", target, executable)
		}
	}
	return &Archive{Metadata: metadata, Files: files}, nil
}

func IsV2(data []byte) bool {
	return len(data) >= 4 && data[0] == 'P' && data[1] == 'K'
}

func cleanArchivePath(name string) (string, error) {
	clean := path.Clean(strings.ReplaceAll(name, "\\", "/"))
	if clean == "." || clean == ".." || path.IsAbs(clean) || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("jp: ruta insegura %q", name)
	}
	return clean, nil
}

func isNativeTarget(targets map[string]string, name string) bool {
	for _, executable := range targets {
		if executable == name {
			return true
		}
	}
	return false
}
