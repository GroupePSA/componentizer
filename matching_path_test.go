package componentizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildInventoriesAsParam(t *testing.T) {
	ps := MatchingPaths{}
	ps.Paths = make([]MatchingPath, 0, 0)

	ps.Paths = append(ps.Paths, mPath{
		comp:         usable{id: "c1"},
		relativePath: "path1",
	})

	ps.Paths = append(ps.Paths, mPath{
		comp:         usable{id: "c2"},
		relativePath: "path2",
	})

	ps.Paths = append(ps.Paths, mPath{
		comp:         usable{id: "c3"},
		relativePath: "path3",
	})

	prefix := "-i"
	strs := ps.PrefixPaths(prefix)
	assert.Len(t, strs, 6)
	assert.Equal(t, prefix, strs[0])
	assert.Equal(t, "path1", strs[1])
	assert.Equal(t, prefix, strs[2])
	assert.Equal(t, "path2", strs[3])
	assert.Equal(t, prefix, strs[4])
	assert.Equal(t, "path3", strs[5])
}
