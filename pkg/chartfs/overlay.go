package chartfs

import (
	"errors"
	"io/fs"
)

// OverlayFS implements fs.FS and looks for files in the embedded filesystem
// first, then the local filesystem.
type OverlayFS struct {
	Embedded fs.FS // embedded data (first)
	Local    fs.FS // local data (second)
}

// Open opens the named file, when it doesn't exist in the embedded FS it falls
// back to the local.
func (o *OverlayFS) Open(name string) (fs.File, error) {
	f, err := o.Embedded.Open(name)
	if err == nil {
		return f, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return o.Local.Open(name)
}
