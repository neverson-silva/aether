package core

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"aether/internal/domain"
)

func (c *Core) SaveZipUpload(name string, data []byte) (*domain.ZipUpload, error) {
	if err := os.MkdirAll(filepath.Join(c.Cfg.BuildsDir, "uploads"), 0o750); err != nil {
		return nil, err
	}
	id := domain.NewID()
	zipPath := filepath.Join(c.Cfg.BuildsDir, "uploads", id+".zip")
	if err := os.WriteFile(zipPath, data, 0o640); err != nil {
		return nil, err
	}
	dest := filepath.Join(c.Cfg.BuildsDir, "uploads", id)
	if err := os.RemoveAll(dest); err != nil {
		return nil, err
	}
	if err := extractZip(data, dest); err != nil {
		os.Remove(zipPath)
		return nil, fmt.Errorf("zip inválido: %w", err)
	}
	flattenSingleRoot(dest)
	return &domain.ZipUpload{ID: id, Name: name, Size: int64(len(data)), Status: "ready"}, nil
}

func extractZip(data []byte, dest string) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		target := filepath.Join(dest, f.Name)
		if !isWithin(dest, target) {
			continue
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o750); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// flattenSingleRoot detecta um único diretório raiz no extraído (comum em
// zips exportados pelo GitHub/VSCode: "projeto/package.json") e move o
// conteúdo para a raiz, para que package.json/Dockerfile fiquem no topo.
func flattenSingleRoot(dest string) {
	entries, err := os.ReadDir(dest)
	if err != nil {
		return
	}
	var root string
	for _, e := range entries {
		if e.Name() == ".DS_Store" || e.Name() == "__MACOSX" {
			continue
		}
		if root != "" || !e.IsDir() {
			return
		}
		root = e.Name()
	}
	if root == "" {
		return
	}
	rootDir := filepath.Join(dest, root)
	if err := copyDir(rootDir, dest); err != nil {
		return
	}
	os.RemoveAll(rootDir)
}

func isWithin(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !filepath.IsAbs(rel)
}

func hasFile(dir, name string) bool {
	fi, err := os.Stat(filepath.Join(dir, name))
	return err == nil && !fi.IsDir()
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}
