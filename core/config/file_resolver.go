package config

import (
	"io"
	"os"
	"path"
)

type FileResolver interface {
	Resolve(file string) io.Reader
}

type RelativeResolver struct {
	paths []string
}

func NewRelativeResolver(paths ...string) FileResolver {

	pathList := make([]string, 0, len(paths))

	for _, v := range paths {
		pathList = append(pathList, v)
	}

	// add the current cwd
	//
	if cwd, err := os.Getwd(); err == nil {
		pathList = append(pathList, cwd)
	}

	// add the exe dir
	pathList = append(pathList, path.Dir(os.Args[0]))

	return &RelativeResolver{
		paths: pathList,
	}
}

func (rr RelativeResolver) Resolve(file string) io.Reader {

	if path.IsAbs(file) {
		if reader, err := os.Open(file); err == nil {
			return reader
		}
	}

	// loop the paths
	//
	for _, v := range rr.paths {
		fp := path.Join(v, file)
		if reader, err := os.Open(fp); err == nil {
			return reader
		}
	}

	return nil
}
