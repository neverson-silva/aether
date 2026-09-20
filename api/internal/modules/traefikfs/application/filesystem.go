package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"aether/internal/platform/worker"
	"gopkg.in/yaml.v3"
)

const maxEditableFileSize = 2 << 20

var (
	ErrInvalidPath       = errors.New("invalid Traefik path")
	ErrProtectedFile     = errors.New("protected Traefik file")
	ErrFileNotEditable   = errors.New("Traefik file is not editable")
	ErrFileTooLarge      = errors.New("Traefik file is too large")
	ErrInvalidConfig     = errors.New("invalid Traefik configuration")
	ErrTraefikNotRunning = errors.New("Traefik container is not running")
)

type Filesystem struct {
	Root    string
	Runtime worker.Runtime
}

type Entry struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Size      int64     `json:"size"`
	Mode      string    `json:"mode"`
	Modified  time.Time `json:"modified_at"`
	Protected bool      `json:"protected"`
}

type File struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Editable  bool   `json:"editable"`
	Redacted  bool   `json:"redacted"`
	Protected bool   `json:"protected"`
}

type Status struct {
	Container string `json:"container"`
	State     string `json:"state"`
	Config    string `json:"config_path"`
	Dynamic   string `json:"dynamic_path"`
	ACME      string `json:"acme_path"`
}

func (f *Filesystem) List() ([]Entry, error) {
	if err := os.MkdirAll(f.Root, 0o755); err != nil {
		return nil, err
	}
	entries := make([]Entry, 0)
	err := filepath.WalkDir(f.Root, func(path string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == f.Root || item.Type()&os.ModeSymlink != 0 {
			if item.Type()&os.ModeSymlink != 0 {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(f.Root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		typ := "file"
		if item.IsDir() {
			typ = "directory"
		}
		entries = append(entries, Entry{
			Path: rel, Name: item.Name(), Type: typ, Size: info.Size(),
			Mode: info.Mode().String(), Modified: info.ModTime(), Protected: isProtectedPath(rel),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

func (f *Filesystem) Read(path string) (File, error) {
	clean, full, err := f.resolve(path)
	if err != nil {
		return File{}, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return File{}, err
	}
	if info.IsDir() {
		return File{}, ErrFileNotEditable
	}
	if info.Size() > maxEditableFileSize {
		return File{}, ErrFileTooLarge
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return File{}, err
	}
	protected := isProtectedPath(clean)
	if protected {
		redacted, redactErr := redactJSON(raw)
		if redactErr != nil {
			return File{Path: clean, Content: "{\n  \"status\": \"redacted\",\n  \"message\": \"ACME storage is protected\"\n}\n", Protected: true, Redacted: true}, nil
		}
		raw = redacted
	}
	return File{Path: clean, Content: string(raw), Editable: isEditablePath(clean), Protected: protected, Redacted: protected}, nil
}

func (f *Filesystem) Write(path, content string) error {
	clean, full, err := f.resolve(path)
	if err != nil {
		return err
	}
	if !isEditablePath(clean) {
		if isProtectedPath(clean) {
			return ErrProtectedFile
		}
		return ErrFileNotEditable
	}
	if len(content) > maxEditableFileSize {
		return ErrFileTooLarge
	}
	if err := validateConfig(clean, []byte(content)); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(full), ".aether-traefik-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.WriteString(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, full)
}

func (f *Filesystem) Delete(path string) error {
	clean, full, err := f.resolve(path)
	if err != nil {
		return err
	}
	if isProtectedPath(clean) {
		return ErrProtectedFile
	}
	if !isEditablePath(clean) || filepath.Base(clean) == "traefik.yml" {
		return ErrFileNotEditable
	}
	return os.Remove(full)
}

func (f *Filesystem) Restart(ctx context.Context) error {
	if f.Runtime == nil {
		return worker.ErrRuntimeUnavailable
	}
	containers, err := f.Runtime.ListContainers(ctx)
	if err != nil {
		return err
	}
	for _, item := range containers {
		if item.Name == "aether-traefik" {
			return f.Runtime.Restart(ctx, item.ID)
		}
	}
	return ErrTraefikNotRunning
}

func (f *Filesystem) Status(ctx context.Context) (Status, error) {
	state := "not found"
	if f.Runtime != nil {
		containers, err := f.Runtime.ListContainers(ctx)
		if err != nil {
			return Status{}, err
		}
		for _, item := range containers {
			if item.Name == "aether-traefik" {
				state = item.State
				break
			}
		}
	}
	return Status{
		Container: "aether-traefik",
		State:     state,
		Config:    filepath.Join(f.Root, "traefik.yml"),
		Dynamic:   filepath.Join(f.Root, "dynamic"),
		ACME:      filepath.Join(f.Root, "acme", "acme.json"),
	}, nil
}

func (f *Filesystem) resolve(path string) (string, string, error) {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	if clean == "." || clean == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", "", ErrInvalidPath
	}
	full := filepath.Join(f.Root, filepath.FromSlash(clean))
	rel, err := filepath.Rel(f.Root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", "", ErrInvalidPath
	}
	probe := full
	for {
		if _, err := os.Lstat(probe); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return "", "", err
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", "", os.ErrNotExist
		}
		probe = parent
	}
	resolved, err := filepath.EvalSymlinks(probe)
	if err != nil {
		return "", "", err
	}
	root, err := filepath.EvalSymlinks(f.Root)
	if err != nil {
		return "", "", err
	}
	resolvedRel, err := filepath.Rel(root, resolved)
	if err != nil || resolvedRel == ".." || strings.HasPrefix(resolvedRel, ".."+string(os.PathSeparator)) {
		return "", "", ErrInvalidPath
	}
	return clean, full, nil
}

func isProtectedPath(path string) bool {
	return filepath.Base(filepath.FromSlash(path)) == "acme.json"
}

func isEditablePath(path string) bool {
	if isProtectedPath(path) {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yml", ".yaml", ".toml", ".json":
		return true
	default:
		return false
	}
}

func validateConfig(path string, content []byte) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yml", ".yaml":
		var value yaml.Node
		if err := yaml.Unmarshal(content, &value); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
		}
	case ".json":
		var value any
		if err := json.Unmarshal(content, &value); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
		}
	}
	return nil
}

func redactJSON(raw []byte) ([]byte, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	redactValue(value)
	return json.MarshalIndent(value, "", "  ")
}

func redactValue(value any) {
	switch current := value.(type) {
	case map[string]any:
		for key, item := range current {
			lower := strings.ToLower(strings.ReplaceAll(key, "_", ""))
			if lower == "privatekey" || lower == "key" || lower == "certificate" {
				current[key] = "[redacted]"
				continue
			}
			redactValue(item)
		}
	case []any:
		for _, item := range current {
			redactValue(item)
		}
	}
}
