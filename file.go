package componentizer

import (
	"log"
	"net/url"
)

//FileScmHandler Represents the scm connector allowing to fetch local repositories.
//
//It implements "github.com/ekara-platform/engine/component/scm.scmHandler
type FileScmHandler struct {
	Logger *log.Logger
}

//Matches implements "github.com/ekara-platform/engine/component/scm.scmHandler
func (fileScm FileScmHandler) Matches(u *url.URL, path string) bool {
	// We always return false to force the repository to be fetched again
	return false
}

//Fetch implements "github.com/ekara-platform/engine/component/scm.scmHandler
func (fileScm FileScmHandler) Fetch(u *url.URL, path string, auth map[string]string) error {
	return copyDir(u.Path, path)
}

//Update implements "github.com/ekara-platform/engine/component/scm.scmHandler
func (fileScm FileScmHandler) Update(path string, auth map[string]string) error {
	// Doing nothing here and it's okay because Matches returns false
	// then the repo will be fetched/copied from scratch and never updated
	return nil
}

//Switch implements "github.com/ekara-platform/engine/component/scm.scmHandler
func (fileScm FileScmHandler) Switch(path string, ref string) error {
	// Doing nothing here and it's okay because we are dealing with
	// physical files then there  is nothing to switch...
	return nil
}
