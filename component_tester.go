package componentizer

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type (
	//ComponentTester is an helper user to run unit tests based on local GIT repositories
	ComponentTester struct {
		t        *testing.T
		logger   *log.Logger
		rootDir  string
		fixDir   string
		compDir  string
		location string

		model Model
		tplC  TemplateContext

		cM ComponentManager
	}

	TestContext struct {
		T              *testing.T
		Logger         *log.Logger
		Directory      string
		DescriptorName string
	}

	testRepo struct {
		path     string
		rep      *git.Repository
		lastHash plumbing.Hash
	}

	testComponentRef string
)

//CreateComponentTester creates a new ComponentTester
func CreateComponentTester(ctx TestContext, tplC TemplateContext) *ComponentTester {
	rootDir := filepath.Join(ctx.Directory, uuid.New().String())
	tester := &ComponentTester{
		t:       ctx.T,
		logger:  ctx.Logger,
		rootDir: rootDir,
		fixDir:  filepath.Join(rootDir, "fixtures"),
		compDir: filepath.Join(rootDir, "components"),
		tplC:    tplC,
	}
	tester.Clean()
	tester.cM = CreateComponentManager(tester.logger, tester.compDir)
	return tester
}

//Clean deletes all the content created locally during a test
func (t *ComponentTester) Clean() {
	t.logger.Printf("Cleaning up test directory %s\n", t.rootDir)
	os.RemoveAll(t.rootDir)
}

//Init initializes the ComponentTester and build the environment
// bases on the launch context used during the tester's creation.
func (t *ComponentTester) Init(c Component) error {
	m, err := t.cM.Init(c, t.tplC)
	if err != nil {
		return err
	}
	t.model = m
	return nil
}

//CreateRepDefaultDescriptor creates a new component folder corresponding
// to the given path and write into it an empty descriptor
func (t *ComponentTester) CreateDirEmptyDesc(path string) *testRepo {
	path = filepath.Join(t.fixDir, path)
	rep, err := git.PlainInit(path, false)
	assert.NotNil(t.t, rep)
	assert.Nil(t.t, err)
	res := &testRepo{
		path: path,
		rep:  rep,
	}
	res.WriteCommit("ekara.yaml", ``)
	return res
}

//CreateRep creates a new component folder corresponding
// to the given path
func (t *ComponentTester) CreateDir(path string) *testRepo {
	path = filepath.Join(t.fixDir, path)
	rep, err := git.PlainInit(path, false)
	assert.NotNil(t.t, rep)
	assert.Nil(t.t, err)
	return &testRepo{
		path: path,
		rep:  rep,
	}
}

//WriteCommit commit into the repo the given content into
// a file named as the provided name
func (r *testRepo) WriteCommit(name, content string) {
	co := &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Create Branchh",
			Email: "dummy@stupid.com",
			When:  time.Now(),
		},
	}

	err := ioutil.WriteFile(filepath.Join(r.path, name), []byte(content), 0644)
	if err != nil {
		panic(err)
	}
	wt, err := r.rep.Worktree()
	if err != nil {
		panic(err)
	}
	_, err = wt.Add(".")
	if err != nil {
		panic(err)
	}
	r.lastHash, err = wt.Commit("Written", co)
	if err != nil {
		panic(err)
	}
}

//WriteCommit create the desired folder and then commit into it the given content into
// a file named as the provided name
func (r *testRepo) WriteFolderCommit(folder, name, content string) {
	co := &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Create Branch",
			Email: "dummy@stupid.com",
			When:  time.Now(),
		},
	}

	path := filepath.Join(r.path, folder)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0777)
	}

	namePath := filepath.Join(path, name)
	err := ioutil.WriteFile(namePath, []byte(content), 0644)
	if err != nil {
		panic(err)
	}
	wt, err := r.rep.Worktree()
	if err != nil {
		panic(err)
	}
	_, err = wt.Add(".")
	if err != nil {
		panic(err)
	}
	r.lastHash, err = wt.Commit("Written", co)
	if err != nil {
		panic(err)
	}
}

//CreateBranch creates a branch into the repo
func (r *testRepo) CreateBranch(branch string) {
	branch = fmt.Sprintf("refs/heads/%s", branch)
	bo := &git.CheckoutOptions{
		Create: true,
		Force:  true,
		Branch: plumbing.ReferenceName(branch),
	}
	wt, err := r.rep.Worktree()
	if err != nil {
		panic(err)
	}

	err = wt.Checkout(bo)
	if err != nil {
		panic(err)
	}
}

//Checkout Switch to the desired branch
func (r *testRepo) Checkout(branch string) {
	branch = fmt.Sprintf("refs/heads/%s", branch)
	bo := &git.CheckoutOptions{
		Create: false,
		Force:  false,
		Branch: plumbing.ReferenceName(branch),
	}
	wt, err := r.rep.Worktree()
	if err != nil {
		panic(err)
	}

	err = wt.Checkout(bo)
	if err != nil {
		panic(err)
	}
}

//Tag creates a tag into the repo
func (r *testRepo) Tag(tag string) {
	to := &git.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  "Test Create Tag",
			Email: "dummy@stupid.com",
			When:  time.Now(),
		},
		Message: "Tag created",
	}
	_, err := r.rep.CreateTag(tag, r.lastHash, to)
	if err != nil {
		panic(err)
	}
}

func (r *testRepo) AsRepository(ref string) Repository {
	abs, err := filepath.Abs(r.path)
	if err != nil {
		panic(err) // will not be able to execute test at this point
	}
	u, err := url.Parse("file://localhost" + abs)
	if err != nil {
		panic(err) // will not be able to execute test at this point
	}
	return Repository{
		Loc: u,
		Ref: ref,
	}
}

//ComponentCount returns the number of components available
func (t *ComponentTester) ComponentCount() int {
	files, _ := ioutil.ReadDir(t.compDir)
	res := 0
	for _, f := range files {
		if f.IsDir() {
			res++
		}
	}
	return res
}

func (t *ComponentTester) AssertComponentAvailable(refs ...string) {
	missing := []string{}
	for _, ref := range refs {
		if !t.cM.IsAvailable(testComponentRef(ref)) {
			missing = append(missing, ref)
		}
	}
	assert.Empty(t.t, missing, "Missing components: "+strings.Join(missing, ", "))
}

//AssertFile asserts that a usable component contains a specific file
func (t ComponentTester) AssertFile(u UsableComponent, file string) {
	_, err := ioutil.ReadFile(filepath.Join(u.RootPath(), file))
	assert.Nil(t.t, err)
}

//AssertFileContent asserts that a usable component contains a specific file with the desired content
func (t ComponentTester) AssertFileContent(u UsableComponent, file, desiredContent string) {
	b, err := ioutil.ReadFile(filepath.Join(u.RootPath(), file))
	assert.Nil(t.t, err)
	assert.Equal(t.t, desiredContent, string(b))
}

func (t *ComponentTester) Model() Model {
	return t.model
}

func (t *ComponentTester) TemplateContext() TemplateContext {
	assert.NotNil(t.t, t.tplC)
	return t.tplC
}

func (t *ComponentTester) T() *testing.T {
	return t.t
}

func (t *ComponentTester) ComponentManager() ComponentManager {
	assert.NotNil(t.t, t.cM)
	return t.cM
}

func (t testComponentRef) ComponentId() string {
	return string(t)
}

func (t testComponentRef) HasComponent() bool {
	return t != ""
}

func (t testComponentRef) Component(model interface{}) (Component, error) {
	return nil, errors.New("test references are not resolvable")
}
