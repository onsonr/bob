package bob

import (
	"fmt"

	"github.com/benchkram/bob/pkg/envutil"
	"github.com/benchkram/errz"

	"github.com/benchkram/bob/bob/bobfile"
	"github.com/benchkram/bob/pkg/nix"
	"github.com/benchkram/bob/pkg/usererror"
)

// NixBuilder acts as a wrapper for github.com/benchkram/bob/pkg/nix package
// and is used for building tasks dependencies
type NixBuilder struct {
	// cache allows caching the dependency to store path
	cache *nix.Cache
	// shellCache allows caching of the nix-shell --command='env' output
	shellCache *nix.ShellCache
}

type NixOption func(n *NixBuilder)

func WithCache(cache *nix.Cache) NixOption {
	return func(n *NixBuilder) {
		n.cache = cache
	}
}

func WithShellCache(cache *nix.ShellCache) NixOption {
	return func(n *NixBuilder) {
		n.shellCache = cache
	}
}

// NewNixBuilder instantiates a new Nix builder instance
func NewNixBuilder(opts ...NixOption) *NixBuilder {
	n := &NixBuilder{}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(n)
	}

	return n
}

// BuildNixDependenciesInPipeline collects and builds nix-dependencies for a pipeline starting at taskName.
func (n *NixBuilder) BuildNixDependenciesInPipeline(ag *bobfile.Bobfile, taskName string) (err error) {
	defer errz.Recover(&err)

	if !nix.IsInstalled() {
		return usererror.Wrap(fmt.Errorf("nix is not installed on your system. Get it from %s", nix.DownloadURl()))
	}

	tasksInPipeline, err := ag.BTasks.CollectTasksInPipeline(taskName)
	errz.Fatal(err)

	return n.BuildNixDependencies(ag, tasksInPipeline, []string{})
}

// BuildNixDependencies builds nix dependencies and prepares the affected tasks
// by setting the store paths on each task in the given aggregate.
func (n *NixBuilder) BuildNixDependencies(ag *bobfile.Bobfile, buildTasksInPipeline, runTasksInPipeline []string) (err error) {
	defer errz.Recover(&err)

	if !nix.IsInstalled() {
		return usererror.Wrap(fmt.Errorf("nix is not installed on your system. Get it from %s", nix.DownloadURl()))
	}

	// Resolve nix storePaths from dependencies
	// and rewrite the affected tasks.
	for _, name := range buildTasksInPipeline {
		t := ag.BTasks[name]

		// construct used dependencies for this task
		var deps []nix.Dependency
		deps = append(deps, t.Dependencies()...)
		deps = nix.UniqueDeps(deps)

		t.SetNixpkgs(ag.Nixpkgs)

		nixShellEnv, err := n.BuildEnvironment(deps, ag.Nixpkgs)
		errz.Fatal(err)
		t.SetEnv(envutil.Merge(nixShellEnv, t.Env()))

		ag.BTasks[name] = t
	}

	for _, name := range runTasksInPipeline {
		t := ag.RTasks[name]

		// construct used dependencies for this task
		var deps []nix.Dependency
		deps = append(deps, t.Dependencies()...)
		deps = nix.UniqueDeps(deps)

		t.SetNixpkgs(ag.Nixpkgs)

		nixShellEnv, err := n.BuildEnvironment(deps, ag.Nixpkgs)
		errz.Fatal(err)
		t.SetEnv(envutil.Merge(nixShellEnv, t.Env()))

		ag.RTasks[name] = t
	}

	return nil
}

// BuildDependencies builds the list of all nix deps
func (n *NixBuilder) BuildDependencies(deps []nix.Dependency) error {
	return nix.BuildDependencies(deps, n.cache)
}

// BuildEnvironment builds the environment with all nix deps
func (n *NixBuilder) BuildEnvironment(deps []nix.Dependency, nixpkgs string) (_ []string, err error) {
	return nix.BuildEnvironment(deps, nixpkgs, n.cache, n.shellCache)
}
