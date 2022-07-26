package bobsync

import (
	"fmt"
	"github.com/benchkram/bob/pkg/versionedsync/collection"
	"github.com/benchkram/bob/pkg/versionedsync/file"
	"sort"
)

type FileList []*file.F

// Len is the number of elements in the collection.
func (f FileList) Len() int {
	return len(f)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (f FileList) Less(i, j int) bool {
	return (f)[i].LocalPath < (f)[j].LocalPath
}

// Swap swaps the elements with indexes i and j.
func (f FileList) Swap(i, j int) {
	tmpF := (f)[i]
	(f)[i] = (f)[j]
	(f)[j] = tmpF
}

type Delta struct {
	// Unchanged are files which have the same hash on local and remote
	// Files in this slice should always have an ID set
	Unchanged FileList
	// ToBeUpdated are files which exist on local and remote but have different hashes
	// Files in this slice should always have an ID set
	ToBeUpdated FileList
	// LocalFilesMissingOnRemote can be read different for push and pull
	// push: what has to be created on the remote and is only on local
	// pull: what has to be removed on local since it is not on remote
	// Files in this slice never have an ID set
	LocalFilesMissingOnRemote FileList
	// RemoteFilesMissingOnLocal can be read different for push and pull
	// push: what has to be removed on the remote and is only on remote
	// pull: what has to be created on local since it is only on remote
	// Files in this slice should always have an ID set
	RemoteFilesMissingOnLocal FileList
}

func (d *Delta) String() string {
	result := ""

	for _, f := range d.Unchanged {
		result += fmt.Sprintf("(unchanged)   %s\n", f.LocalPath)
	}
	for _, f := range d.ToBeUpdated {
		result += fmt.Sprintf("(changed)     %s\n", f.LocalPath)
	}
	for _, f := range d.LocalFilesMissingOnRemote {
		result += fmt.Sprintf("(local only)  %s\n", f.LocalPath)
	}
	for _, f := range d.RemoteFilesMissingOnLocal {
		result += fmt.Sprintf("(remote only) %s\n", f.LocalPath)
	}
	return result
}

func (d *Delta) PushOverview() string {
	result := ""

	for _, f := range d.Unchanged {
		result += fmt.Sprintf("(unchanged)       %s\n", f.LocalPath)
	}
	for _, f := range d.ToBeUpdated {
		result += fmt.Sprintf("(override server) %s\n", f.LocalPath)
	}
	for _, f := range d.LocalFilesMissingOnRemote {
		result += fmt.Sprintf("(upload)          %s\n", f.LocalPath)
	}
	for _, f := range d.RemoteFilesMissingOnLocal {
		result += fmt.Sprintf("(delete remote)   %s\n", f.LocalPath)
	}
	return result
}

func (d *Delta) PullOverview() string {
	result := ""

	for _, f := range d.Unchanged {
		result += fmt.Sprintf("(unchanged)      %s\n", f.LocalPath)
	}
	for _, f := range d.ToBeUpdated {
		result += fmt.Sprintf("(override local) %s\n", f.LocalPath)
	}
	for _, f := range d.LocalFilesMissingOnRemote {
		result += fmt.Sprintf("(delete local)   %s\n", f.LocalPath)
	}
	for _, f := range d.RemoteFilesMissingOnLocal {
		result += fmt.Sprintf("(download)       %s\n", f.LocalPath)
	}
	return result

}

// NewDelta creates a delta that describes differences between local and remote
func NewDelta(local HashCache, remote collection.C) *Delta {
	delta := &Delta{}

	for _, remoteF := range remote.Files {
		fingerprint, ok := local[remoteF.LocalPath]
		if ok && fingerprint.Hash == remoteF.Hash {
			delta.Unchanged = append(delta.Unchanged, remoteF)
		} else if ok {
			// local and remote differ
			delta.ToBeUpdated = append(delta.ToBeUpdated, remoteF)
		} else {
			// remote file non-existent on local
			delta.RemoteFilesMissingOnLocal = append(delta.RemoteFilesMissingOnLocal, remoteF)
		}
	}
	for localPath, fingerprint := range local {
		_, ok := remote.FileByPath(localPath)
		if !ok {
			// localPath non-existent on remote
			delta.LocalFilesMissingOnRemote = append(delta.LocalFilesMissingOnRemote,
				&file.F{
					LocalPath:   localPath,
					Hash:        fingerprint.Hash,
					IsDirectory: fingerprint.IsDir,
				})
		}
	}
	delta.Sort()
	return delta
}

func (d *Delta) Sort() {
	sort.Sort(d.Unchanged)
	sort.Sort(d.ToBeUpdated)
	sort.Sort(d.RemoteFilesMissingOnLocal)
	sort.Sort(d.LocalFilesMissingOnRemote)
}
