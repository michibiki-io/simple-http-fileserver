package service

import (
	"io/fs"
	"net/http"
	"strings"
)

// containsDotFile reports whether name contains a path element starting with a period.
// The name is assumed to be a delimited by forward slashes, as guaranteed
// by the http.FileSystem interface.
func containsDotFile(name string) bool {
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

// dotFileHidingFile is the http.File use in dotFileHidingFileSystem.
// It is used to wrap the Readdir method of http.File so that we can
// remove files and directories that start with a period from its output.
type dotFileHidingFile struct {
	file http.File
}

// Readdir is a wrapper around the Readdir method of the embedded File
// that filters out all files that start with a period in their name.
func (f dotFileHidingFile) Readdir(n int) (fis []fs.FileInfo, err error) {
	files, err := f.file.Readdir(n)
	for _, file := range files { // Filters out the dot files
		if !strings.HasPrefix(file.Name(), ".") {
			fis = append(fis, file)
		}
	}
	return
}
func (f dotFileHidingFile) Close() error               { return f.file.Close() }
func (f dotFileHidingFile) Read(b []byte) (int, error) { return f.file.Read(b) }
func (f dotFileHidingFile) Stat() (fs.FileInfo, error) { return f.file.Stat() }
func (f dotFileHidingFile) Seek(offset int64, whence int) (int64, error) {
	return f.file.Seek(offset, whence)
}

type dotFileHidingFileSystem struct {
	fileSystem http.FileSystem
}

// Open is a wrapper around the Open method of the embedded FileSystem
// that serves a 403 permission error when name has a file or directory
// with whose name starts with a period in its path.
func (fsys *dotFileHidingFileSystem) Open(name string) (http.File, error) {
	if containsDotFile(name) { // If dot file, return 403 response
		return nil, fs.ErrPermission
	}

	file, err := fsys.fileSystem.Open(name)
	if err != nil {
		return nil, err
	}
	return dotFileHidingFile{file: file}, err
}

func (fsys *dotFileHidingFileSystem) Exists(prefix string, filepath string) bool {
	url := strings.TrimPrefix(filepath, prefix)
	if len(url) < len(filepath) {
		_, err := fsys.fileSystem.Open(url)
		if err != nil {
			return false
		}
		return true
	}
	return false
}

func DotFileHidingFileSystem(fileSystems http.FileSystem) *dotFileHidingFileSystem {
	return &dotFileHidingFileSystem{
		fileSystems,
	}
}
