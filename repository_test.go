package componentizer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRepositoryMerge(t *testing.T) {
	auth := make(map[string]string)
	auth["key1"] = "abc"
	auth["key2"] = "def"
	r1, e := CreateRepository("https://github.com/ekara-platform/repo1", "master", auth)
	assert.Nil(t, e)
	r2, e := CreateRepository("repo2", "dev", make(map[string]string))
	assert.Nil(t, e)
	r1.Merge(r2)
	assert.Equal(t, "repo2", r1.Loc.Path)
	assert.Equal(t, "dev", r1.Ref)
	assert.Equal(t, "abc", r1.Authentication["key1"])
	assert.Equal(t, "def", r1.Authentication["key2"])
}

func TestRepositoryZeroVal(t *testing.T) {
	r1 := Repository{}
	auth := make(map[string]string)
	auth["key1"] = "abc"
	auth["key2"] = "def"
	r2, e := CreateRepository("repo2", "dev", auth)
	assert.Nil(t, e)
	r1.Merge(r2)
	assert.Equal(t, "repo2", r1.Loc.Path)
	assert.Equal(t, "dev", r1.Ref)
	assert.Equal(t, "abc", r1.Authentication["key1"])
	assert.Equal(t, "def", r1.Authentication["key2"])
}
