package componentizer

import (
	"fmt"
	"net/url"
	"strings"
)

type (
	//Repository represents a component location
	Repository struct {
		// Loc holds the absolute location of the repository
		Loc *url.URL
		// The reference to the branch or tag to fetch. If not specified the default branch will be fetched
		Ref string
		// The authentication parameters to use if repository is not publicly accessible
		Authentication map[string]string
	}
)

//CreateRepository creates a repository
//	Parameters
//		repo: the repository Url where to fetch the component
//		ref: the ref to fetch, if the ref is not specified then the default branch will be fetched
func CreateRepository(loc string, ref string, auth map[string]string) (Repository, error) {
	u, err := url.Parse(loc)
	if err != nil {
		return Repository{}, err
	}

	r := Repository{
		Loc:            u,
		Ref:            ref,
		Authentication: make(map[string]string),
	}

	for k, v := range auth {
		r.Authentication[k] = v
	}

	return r, nil
}

func (r Repository) CreateChildRepository(loc *url.URL, ref string, auth map[string]string) (Repository, error) {
	if r.Loc != nil && !loc.IsAbs() {
		// Copy the parent location
		uCopy := *r.Loc

		if strings.HasPrefix(loc.Path, "/") {
			uCopy.Path = loc.Path
		} else {
			// Remove the last path part
			if uCopy.Path != "" {
				idx := strings.LastIndex(uCopy.Path, "/")
				if idx != -1 {
					uCopy.Path = uCopy.Path[:idx]
				}
			}
			// Then append the relative path to it
			if !strings.HasSuffix(uCopy.Path, "/") {
				uCopy.Path = uCopy.Path + "/"
			}
		}
		uCopy.Path = uCopy.Path + loc.Path
		loc = &uCopy
	}

	return CreateRepository(loc.String(), ref, auth)
}

func (r *Repository) Merge(with Repository) {
	if with.Loc.Path != "" {
		r.Loc = with.Loc
	}
	if with.Ref != "" {
		r.Ref = with.Ref
	}
	if r.Authentication == nil {
		r.Authentication = make(map[string]string)
	}
	for k := range with.Authentication {
		r.Authentication[k] = with.Authentication[k]
	}
}

func (r Repository) String() string {
	if r.Loc != nil {
		return fmt.Sprintf("%s@%s", r.Loc.String(), r.Ref)
	} else {
		return ""
	}
}
