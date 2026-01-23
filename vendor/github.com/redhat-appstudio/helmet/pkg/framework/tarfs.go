package framework

import (
	"bytes"
	"io/fs"

	"github.com/quay/claircore/pkg/tarfs"
)

// NewTarFS creates an fs.FS from a tarball.
func NewTarFS(tarball []byte) (fs.FS, error) {
	return tarfs.New(bytes.NewReader(tarball))
}
