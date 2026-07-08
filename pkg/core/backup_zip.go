package core

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// zipDirectories zips files matching configuration rules
func (r *Runtime) zipDirectories(includes, excludes []string, destZip string) error {
	zipFile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	for _, inc := range includes {
		walkDir := inc
		if walkDir == "." {
			walkDir, _ = os.Getwd()
		}

		filepath.Walk(walkDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			rel, _ := filepath.Rel(walkDir, path)
			if rel == "." || rel == ".." {
				return nil
			}

			for _, ex := range excludes {
				if strings.Contains(rel, ex) || strings.HasPrefix(rel, ex) {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}

			if strings.HasSuffix(path, ".zip") || strings.HasSuffix(path, ".enc") {
				return nil
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return nil
			}

			header.Name = filepath.ToSlash(rel)
			if info.IsDir() {
				header.Name += "/"
			} else {
				header.Method = zip.Deflate
			}

			writer, err := archive.CreateHeader(header)
			if err != nil {
				return nil
			}

			if !info.IsDir() {
				file, err := os.Open(path)
				if err == nil {
					io.Copy(writer, file)
					file.Close()
				}
			}
			return nil
		})
	}
	return nil
}

// zipDirectory zips a single directory
func (r *Runtime) zipDirectory(srcDir, destZip string) error {
	zipFile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		rel, _ := filepath.Rel(srcDir, path)
		if rel == "." || rel == ".." {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return nil
		}

		header.Name = filepath.ToSlash(rel)
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return nil
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err == nil {
				io.Copy(writer, file)
				file.Close()
			}
		}
		return nil
	})
	return nil
}

// unzipFile extracts a zip file
func (r *Runtime) unzipFile(srcZip, destDir string) bool {
	archive, err := zip.OpenReader(srcZip)
	if err != nil {
		return false
	}
	defer archive.Close()

	destClean := filepath.Clean(destDir)

	for _, f := range archive.File {
		filePath := filepath.Join(destClean, f.Name)
		if !strings.HasPrefix(filepath.Clean(filePath), destClean) {
			return false // Zip slip protection
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return false
		}

		fileInArchive, err := f.Open()
		if err != nil {
			dstFile.Close()
			return false
		}

		io.Copy(dstFile, fileInArchive)
		dstFile.Close()
		fileInArchive.Close()
	}
	return true
}

// readManifestFromZip helper
func (r *Runtime) readManifestFromZip(zipPath string) *BackupManifest {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil
	}
	defer archive.Close()

	for _, f := range archive.File {
		if f.Name == "manifest.json" {
			rc, err := f.Open()
			if err != nil {
				return nil
			}
			defer rc.Close()
			var m BackupManifest
			if json.NewDecoder(rc).Decode(&m) == nil {
				return &m
			}
		}
	}
	return nil
}
