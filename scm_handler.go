package componentizer

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
)

//scmHandler is the common definition of all SCM handlers used to acces
// to component repositories
type scmHandler interface {
	//Matches return true if a repository has already be fetched into the path and if its
	// remote configuration is the same than the desired  one
	Matches(u *url.URL, path string) bool
	//Fetch fetches the repository content into the given path.
	Fetch(u *url.URL, path string, auth map[string]string) error
	//Update updates the repository content into the given path.
	Update(path string, auth map[string]string) error
	//Switch executes a checkout to the desired reference
	Switch(path string, ref string) error
}

//Handler allows to fetch a component.
//
//If the component repository has already been fetched and if it matches
// then it will be updated, if not it will be fetched.
//
type Handler func() (fetchedComponent, error)

//GetHandler returns an handler able to fetch a component
func GetScmHandler(l *log.Logger, dir string, c Component) (Handler, error) {
	loc := c.GetRepository().Loc
	switch loc.Scheme {
	case SchemeFile:
		return fetchThroughSCM(c, FileScmHandler{Logger: l}, loc, dir), nil
	case SchemeGits, SchemeHttp, SchemeHttps:
		return fetchThroughSCM(c, GitScmHandler{Logger: l}, loc, dir), nil
	default:
		return nil, fmt.Errorf("unsupported SCM: %s", loc.String())
	}
}

func fetchThroughSCM(c Component, scm scmHandler, u *url.URL, dir string) func() (fetchedComponent, error) {
	return func() (fetchedComponent, error) {
		fc := fetchedComponent{
			id: c.ComponentId(),
		}
		cPath := filepath.Join(dir, c.ComponentId())
		fc.rootPath = cPath
		if _, err := os.Stat(cPath); err == nil {
			if scm.Matches(u, cPath) {
				err := scm.Update(cPath, c.GetRepository().Authentication)
				if err != nil {
					return fc, err
				}
			} else {
				err := os.RemoveAll(cPath)
				if err != nil {
					return fc, err
				}
				err = scm.Fetch(u, cPath, c.GetRepository().Authentication)
				if err != nil {
					return fc, err
				}
			}
		} else {
			err := scm.Fetch(u, cPath, c.GetRepository().Authentication)
			if err != nil {
				return fc, err
			}
		}
		err := scm.Switch(cPath, c.GetRepository().Ref)
		if err != nil {
			return fc, err
		}

		return fc, nil
	}
}
