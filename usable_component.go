package componentizer

import (
	"os"
	"path/filepath"
)

type (
	//UsableComponent Represent a component which can be used physically
	UsableComponent interface {
		//Name returns the name of the component
		Id() string
		//Templated returns true is the component content has been templated
		Templated() bool
		//Release deletes the templated content.
		Release()
		//RootPath returns the absolute path of the, eventually templated, component
		RootPath() string
		//ContainsFile returns the matching path of the searched file
		ContainsFile(name string) (bool, MatchingPath)
		//ContainsDirectory returns the matching path of the searched directory
		ContainsDirectory(name string) (bool, MatchingPath)
		//Source returns the component that was used to produce this UsableComponent
		Source() ComponentRef
	}

	usable struct {
		id        string
		release   func()
		path      string
		templated bool
		source    ComponentRef
	}
)

func (u usable) Id() string {
	return u.id
}

func (u usable) Release() {
	if u.release != nil {
		u.release()
	}
}

func (u usable) RootPath() string {
	return u.path
}

func (u usable) Templated() bool {
	return u.templated
}

func (u usable) ContainsFile(path string) (bool, MatchingPath) {
	return u.contains(false, path)
}

func (u usable) ContainsDirectory(path string) (bool, MatchingPath) {
	return u.contains(true, path)
}

func (u usable) Source() ComponentRef {
	return u.source
}

func (u usable) contains(isFolder bool, path string) (bool, MatchingPath) {
	res := mPath{
		comp: u,
	}
	filePath := filepath.Join(u.path, path)
	if info, err := os.Stat(filePath); err == nil && (isFolder == info.IsDir()) {
		res.relativePath = path
		return true, res
	}
	return false, res
}
