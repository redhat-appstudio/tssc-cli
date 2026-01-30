package chartfs

import (
	"io/fs"
	"path/filepath"

	"helm.sh/helm/v3/pkg/chart/loader"
)

// BufferedFiles represents the group of files needed to load a Helm chart in
// memory, the files data is stored as `loader.BufferedFile` instances.
type BufferedFiles struct {
	fs      fs.FS                  // file system instance
	baseDir string                 // base directory path
	files   []*loader.BufferedFile // buffered files
}

// Files returns the buffered files collected by "Walk" function.
func (b *BufferedFiles) Files() []*loader.BufferedFile {
	return b.files
}

// Walk is the callback function used by "fs.WalkDir" to collect the files data.
func (b *BufferedFiles) Walk(filePath string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	// Skip directories. The "Walk" function is invoked recursively for each file,
	// or directory, in the filesystem so directories can be safely ignored,
	// without the need for recursive calls.
	if d.IsDir() {
		return nil
	}

	data, err := fs.ReadFile(b.fs, filePath)
	if err != nil {
		return err
	}

	// The file path is relative to the base directory, therefore paths like
	// "templates/file.tpl" are kept as is.
	relPath, err := filepath.Rel(b.baseDir, filePath)
	if err != nil {
		return err
	}
	// Appending the file data using only the base name as reference. The
	// collected files represent a single Helm chart payload.
	b.files = append(b.files, &loader.BufferedFile{Name: relPath, Data: data})
	return nil
}

// NewBufferedFiles creates a new BufferedFiles instance.
func NewBufferedFiles(cfs fs.FS, baseDir string) *BufferedFiles {
	return &BufferedFiles{
		fs:      cfs,
		baseDir: baseDir,
		files:   []*loader.BufferedFile{},
	}
}
