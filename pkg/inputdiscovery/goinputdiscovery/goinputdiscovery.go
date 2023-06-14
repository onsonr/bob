package goinputdiscovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/benchkram/bob/pkg/inputdiscovery"
	"github.com/benchkram/errz"
	"golang.org/x/tools/go/packages"
)

var Keyword = "gopackage"

type goInputDiscovery struct {
	projectDir string
}

type Option func(discovery *goInputDiscovery)

func NewGoInputDiscovery(options ...Option) inputdiscovery.InputDiscovery {
	id := &goInputDiscovery{}
	for _, opt := range options {
		opt(id)
	}
	return id
}

// GetInputs lists all directories which are used as input for the main go file
// The path of the given mainFile has to be absolute.
// Returned paths are absolute.
// The function expects that there is a 'go.mod' file next to the main file.
func (id *goInputDiscovery) GetInputs(packagePathAbs string) (_ []string, err error) {
	defer errz.Recover(&err)

	cfg := &packages.Config{
		Dir:  id.projectDir,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedDeps | packages.NeedImports | packages.NeedModule | packages.NeedEmbedFiles,
	}

	pkgs, err := packages.Load(cfg, packagePathAbs)
	errz.Fatal(err)

	if len(pkgs) < 1 {
		return nil, fmt.Errorf("did not find a go package at %s", packagePathAbs)
	} else if len(pkgs) > 1 {
		return nil, fmt.Errorf("found more than one go package at %s", packagePathAbs)
	}

	pkg := pkgs[0]

	if pkg.Module == nil {
		return nil, fmt.Errorf("package has no go.mod file: expected it to be in the bob root dir")
	}
	modFilePath := pkg.Module.GoMod
	packageName := pkg.Module.Path

	prefix := packageName + "/"

	paths := unique(localDependencies(pkg.Imports, prefix))

	var result []string
	for _, p := range paths {
		result = append(result, filepath.Join(packagePathAbs, p))
	}

	// add go files in package
	result = append(result, pkg.GoFiles...)

	// add embedded files
	result = append(result, pkg.EmbedFiles...)

	// add all other files
	result = append(result, pkg.OtherFiles...)

	// add the go mod and go sum file if they exist
	_, err = os.Stat(modFilePath)
	if err != nil {
		return nil, fmt.Errorf("can not find 'go.mod' file at %s", modFilePath)
	}
	result = append(result, modFilePath)
	sumFilePath := filepath.Join(id.projectDir, "go.sum")
	_, err = os.Stat(sumFilePath)
	if err != nil {
		return nil, fmt.Errorf("can not find 'go.sum' file at %s", sumFilePath)
	}
	result = append(result, sumFilePath)

	return result, nil
}

func localDependencies(imports map[string]*packages.Package, prefix string) []string {
	var deps []string
	for _, pkg := range imports {
		// if the package is a local package add its whole dir
		if strings.HasPrefix(pkg.ID, prefix) {
			slug := strings.TrimPrefix(pkg.ID, prefix)
			slugParts := strings.Split(slug, "/")
			if len(slugParts) > 0 {
				deps = append(deps, slugParts[0])
			}
		}

		if len(pkg.Imports) > 0 {
			newDeps := localDependencies(pkg.Imports, prefix)
			deps = append(deps, newDeps...)
		}
	}
	return deps
}

func unique(ss []string) []string {
	unique := make([]string, 0, len(ss))

	um := make(map[string]struct{})
	for _, s := range ss {
		if _, ok := um[s]; !ok {
			um[s] = struct{}{}
			unique = append(unique, s)
		}
	}

	return unique
}
