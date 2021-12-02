package filestore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Benchkram/bob/pkg/store"
	"github.com/Benchkram/errz"
)

type s struct {
	dir string
}

// New creates a filestore. The caller is responsible to pass a
// existing directory.
func New(dir string, opts ...Option) store.Store {
	s := &s{
		dir: dir,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(s)
	}

	return s
}

// NewArtifact creates a new file. The caller is responsible to call Close().
// Existing artifacts are overwritten.
func (s *s) NewArtifact(_ context.Context, id string) (store.Artifact, error) {
	return os.Create(filepath.Join(s.dir, id))
}

// GetArtifact opens a file
func (s *s) GetArtifact(_ context.Context, id string) (empty store.Artifact, _ error) {
	return os.Open(filepath.Join(s.dir, id))
}

func (s *s) Clean(_ context.Context) (err error) {
	defer errz.Recover(&err)

	homeDir, err := os.UserHomeDir()
	errz.Fatal(err)
	if s.dir == "/" || s.dir == homeDir {
		return fmt.Errorf("Cleanup of %s is not allowed", s.dir)
	}

	entrys, err := os.ReadDir(s.dir)
	errz.Fatal(err)

	for _, entry := range entrys {
		if entry.IsDir() {
			continue
		}
		_ = os.Remove(filepath.Join(s.dir, entry.Name()))
	}

	return nil
}