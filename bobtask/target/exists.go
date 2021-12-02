package target

import (
	"path/filepath"

	"github.com/Benchkram/bob/pkg/file"
)

// Exists determines if the target exists without
// validating it's integrety.
func (t *T) Exists() bool {
	switch t.Type {
	case Path:
		return t.existsFile()
	case Docker:
		return t.existsDocker()
	default:
		return t.existsFile()
	}
}

func (t *T) existsFile() bool {
	if len(t.Paths) == 0 {
		return true
	}
	// check plain existence
	for _, f := range t.Paths {
		target := filepath.Join(t.dir, f)
		if !file.Exists(target) {
			return false
		}
	}

	return true
}

func (t *T) existsDocker() bool {
	return true
}