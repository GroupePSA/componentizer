package componentizer

import (
	"errors"
	"net/url"
)

type (
	// SCMType represents a type of Source Control Management system
	SCMType string
)

const (
	//GitScm type of GIT source control management system
	GitScm SCMType = SCMType(SchemeGits)
	//SvnScm type of SVN source control management system
	SvnScm SCMType = SCMType(SchemeSvn)
	//UnknownScm represents an unknown source control management system
	UnknownScm SCMType = ""

	//SchemeFile  scheme for a file
	SchemeFile string = "file"
	//SchemeGits  scheme for Git
	SchemeGits string = "git"
	//SchemeSvn  scheme for svn
	SchemeSvn string = "svn"
	//SchemeHttp  scheme for http
	SchemeHttp string = "http"
	//SchemeHttps  scheme for https
	SchemeHttps string = "https"
)

func resolveSCMType(loc url.URL) (SCMType, error) {
	switch loc.Scheme {
	case SchemeFile, SchemeGits, SchemeHttp, SchemeHttps:
		// TODO: for now assume git on local directories, later try to detect
		return GitScm, nil
	case SchemeSvn:
		return SvnScm, nil
	}
	return UnknownScm, errors.New("unknown fetch protocol: " + loc.Scheme)
}
