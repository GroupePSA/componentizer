package componentizer

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	gl "github.com/gobwas/glob"
	"github.com/oklog/ulid"
)

// runTemplate runs the templates defined into a path
func executeTemplate(path string, patterns []string, ctx TemplateContext) (string, error) {
	if len(patterns) > 0 {
		globs := make([]gl.Glob, 0, 0)
		files := make([]string, 0, 0)
		for _, p := range patterns {
			pa := filepath.Join(path, p)
			// workaround of issue : https://github.com/gobwas/glob/issues/35
			if runtime.GOOS == "windows" {
				pa = strings.Replace(pa, "\\", "/", -1)
			}
			globs = append(globs, gl.MustCompile(pa, '/'))
		}
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			for _, p := range patterns {
				pa := filepath.Join(path, p)
				if path == pa {
					files = append(files, path)
					return nil
				}
			}

			for _, g := range globs {
				pa := path
				// workaround of issue : https://github.com/gobwas/glob/issues/35
				if runtime.GOOS == "windows" {
					pa = strings.Replace(pa, "\\", "/", -1)
				}
				if g.Match(pa) {
					files = append(files, path)
					return nil
				}
			}
			return nil
		})

		if err != nil {
			return "", err
		}

		var uuid string
		var tmpPath string

		// No matching files encountered then it won't be templated
		if len(files) == 0 {
			return "", nil
		}

		uuid = genUlid()
		tmpPath = path + "_" + uuid
		err = copyDir(path, tmpPath)
		if err != nil {
			return "", err
		}

		for _, input := range files {
			input = strings.Replace(input, path, tmpPath, -1)

			content, err := ioutil.ReadFile(input)
			if err != nil {
				return "", err
			}

			templatedContent, err := ctx.Execute(string(content))
			if err != nil {
				return "", err
			}

			err = ioutil.WriteFile(input, []byte(templatedContent), 0644)
		}
		return tmpPath, nil
	}
	return "", nil
}

func genUlid() string {
	t := time.Now().UTC()
	entropy := rand.New(rand.NewSource(t.UnixNano()))
	id := ulid.MustNew(ulid.Timestamp(t), entropy)
	return id.String()
}
