package main

// neuteredFileSystem is used to prevent directory listing of static assets
import (
	"net/http"
	"os"
)

type neuteredFileSystem struct {
	fs http.FileSystem
}

func (nfs neuteredFileSystem) Open(path string) (http.File, error) {

	// Check if path exists
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	// If path exists, check if is a file or a directory.
	// If is a directory, stop here with an error saying that file
	// does not exist. So user will get a 404 error code for a file/directory
	// that does not exist, and for directories that exist.
	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if s.IsDir() {
		return nil, os.ErrNotExist
	}

	// If file exists and the path is not a directory, let's return the file
	return f, nil
}
