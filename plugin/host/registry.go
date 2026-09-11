package host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anytty/anytty/shared/filelock"

	"gopkg.in/yaml.v3"
)

type Installation struct {
	ID        string `yaml:"id"`
	Directory string `yaml:"directory"`
	Enabled   bool   `yaml:"enabled"`
}
type Registry struct {
	Version int            `yaml:"version"`
	Plugins []Installation `yaml:"plugins"`
}

// UpdateRegistry serializes the whole read/modify/publish transaction across
// CLI processes. A failed mutation leaves the last usable registry intact.
func UpdateRegistry(ctx context.Context, path string, update func(*Registry) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	lock, err := filelock.AcquireContext(ctx, path+".lock", false)
	if err != nil {
		return err
	}
	defer lock.Close()
	registry, err := LoadRegistry(path)
	if err != nil {
		return err
	}
	if err = update(&registry); err != nil {
		return err
	}
	return SaveRegistry(path, registry)
}

func LoadRegistry(path string) (Registry, error) {
	result := Registry{Version: 1}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	if err = yaml.Unmarshal(data, &result); err != nil {
		return result, err
	}
	if result.Version != 1 {
		return result, errors.New("unsupported plugin registry")
	}
	seen := map[string]bool{}
	for _, item := range result.Plugins {
		if !validID.MatchString(item.ID) || seen[item.ID] || !filepath.IsAbs(item.Directory) {
			return result, errors.New("invalid plugin registry entry")
		}
		seen[item.ID] = true
	}
	return result, nil
}
func SaveRegistry(path string, registry Registry) error {
	data, err := yaml.Marshal(registry)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".plugins-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err = file.Chmod(0600); err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), path)
}

// Link registers an already reviewed local package. It never executes its code.
func (registry *Registry) Link(directory string) (Manifest, error) {
	directory, err := filepath.Abs(directory)
	if err != nil {
		return Manifest{}, err
	}
	directory, err = filepath.EvalSymlinks(directory)
	if err != nil {
		return Manifest{}, err
	}
	manifest, err := ReadManifest(directory)
	if err != nil {
		return Manifest{}, err
	}
	for i, item := range registry.Plugins {
		if item.ID == manifest.ID {
			registry.Plugins[i] = Installation{manifest.ID, directory, true}
			return manifest, nil
		}
	}
	registry.Plugins = append(registry.Plugins, Installation{manifest.ID, directory, true})
	sort.Slice(registry.Plugins, func(i, j int) bool { return registry.Plugins[i].ID < registry.Plugins[j].ID })
	return manifest, nil
}

// Install takes a private snapshot of a local package. Updates get a new
// directory, so an already running child keeps its original executable/files.
func (registry *Registry) Install(source, packages string) (Manifest, error) {
	source, err := filepath.Abs(source)
	if err != nil {
		return Manifest{}, err
	}
	source, err = filepath.EvalSymlinks(source)
	if err != nil {
		return Manifest{}, err
	}
	manifest, err := ReadManifest(source)
	if err != nil {
		return Manifest{}, err
	}
	packages, err = filepath.Abs(packages)
	if err != nil {
		return Manifest{}, err
	}
	if rel, err := filepath.Rel(source, packages); err != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
		return Manifest{}, errors.New("installation directory must be outside the source package")
	}
	if err = os.MkdirAll(packages, 0700); err != nil {
		return Manifest{}, err
	}
	packages, err = filepath.EvalSymlinks(packages)
	if err != nil {
		return Manifest{}, err
	}
	if rel, err := filepath.Rel(source, packages); err != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
		return Manifest{}, errors.New("installation directory resolves inside the source package")
	}
	destination, err := os.MkdirTemp(packages, manifest.ID+"-")
	if err != nil {
		return Manifest{}, err
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(destination)
		}
	}()
	err = filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("package entry %s must be a regular file or directory", rel)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600|info.Mode().Perm()&0111)
		if err != nil {
			return err
		}
		_, err = io.Copy(output, input)
		closeErr := output.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
	if err != nil {
		return Manifest{}, err
	}
	manifest, err = registry.Link(destination)
	if err != nil {
		return Manifest{}, err
	}
	ok = true
	return manifest, nil
}
func (registry *Registry) Enable(id string, enabled bool) error {
	for i := range registry.Plugins {
		if registry.Plugins[i].ID == id {
			registry.Plugins[i].Enabled = enabled
			return nil
		}
	}
	return fmt.Errorf("plugin %s not installed", id)
}
func (registry *Registry) Remove(id string) error {
	for i := range registry.Plugins {
		if registry.Plugins[i].ID == id {
			registry.Plugins = append(registry.Plugins[:i], registry.Plugins[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("plugin %s not installed", id)
}
