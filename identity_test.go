package hercules

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

func fixtureIdentityDetector() *IdentityDetector {
	peopleDict := map[string]int{}
	peopleDict["vadim@sourced.tech"] = 0
	peopleDict["gmarkhor@gmail.com"] = 0
	reversePeopleDict := make([]string, 1)
	reversePeopleDict[0] = "Vadim"
	id := IdentityDetector{
		PeopleDict:         peopleDict,
		ReversedPeopleDict: reversePeopleDict,
	}
	id.Initialize(testRepository)
	return &id
}

func TestIdentityDetectorMeta(t *testing.T) {
	id := fixtureIdentityDetector()
	assert.Equal(t, id.Name(), "IdentityDetector")
	assert.Equal(t, len(id.Requires()), 0)
	assert.Equal(t, len(id.Provides()), 1)
	assert.Equal(t, id.Provides()[0], "author")
	opts := id.ListConfigurationOptions()
	assert.Len(t, opts, 1)
	assert.Equal(t, opts[0].Name, ConfigIdentityDetectorPeopleDictPath)
}

func TestIdentityDetectorConfigure(t *testing.T) {
	id := fixtureIdentityDetector()
	facts := map[string]interface{}{}
	m1 := map[string]int{}
	m2 := []string{}
	facts[FactIdentityDetectorPeopleDict] = m1
	facts[FactIdentityDetectorReversedPeopleDict] = m2
	id.Configure(facts)
	assert.Equal(t, m1, facts[FactIdentityDetectorPeopleDict])
	assert.Equal(t, m2, facts[FactIdentityDetectorReversedPeopleDict])
	assert.Equal(t, id.PeopleDict, facts[FactIdentityDetectorPeopleDict])
	assert.Equal(t, id.ReversedPeopleDict, facts[FactIdentityDetectorReversedPeopleDict])
	id = fixtureIdentityDetector()
	tmpf, err := ioutil.TempFile("", "hercules-test-")
	assert.Nil(t, err)
	defer os.Remove(tmpf.Name())
	_, err = tmpf.WriteString(`Egor|egor@sourced.tech
Vadim|vadim@sourced.tech`)
	assert.Nil(t, err)
	assert.Nil(t, tmpf.Close())
	delete(facts, FactIdentityDetectorPeopleDict)
	delete(facts, FactIdentityDetectorReversedPeopleDict)
	facts[ConfigIdentityDetectorPeopleDictPath] = tmpf.Name()
	id.Configure(facts)
	assert.Len(t, id.PeopleDict, 2)
	assert.Len(t, id.ReversedPeopleDict, 1)
	assert.Equal(t, id.ReversedPeopleDict[0], "Vadim")
	delete(facts, FactIdentityDetectorPeopleDict)
	delete(facts, FactIdentityDetectorReversedPeopleDict)
	id = fixtureIdentityDetector()
	id.PeopleDict = nil
	id.Configure(facts)
	assert.Equal(t, id.PeopleDict, facts[FactIdentityDetectorPeopleDict])
	assert.Equal(t, id.ReversedPeopleDict, facts[FactIdentityDetectorReversedPeopleDict])
	assert.Len(t, id.PeopleDict, 4)
	assert.Len(t, id.ReversedPeopleDict, 3)
	assert.Equal(t, id.ReversedPeopleDict[0], "Egor")
	assert.Equal(t, facts[FactIdentityDetectorPeopleCount], 2)
	delete(facts, FactIdentityDetectorPeopleDict)
	delete(facts, FactIdentityDetectorReversedPeopleDict)
	id = fixtureIdentityDetector()
	id.ReversedPeopleDict = nil
	id.Configure(facts)
	assert.Equal(t, id.PeopleDict, facts[FactIdentityDetectorPeopleDict])
	assert.Equal(t, id.ReversedPeopleDict, facts[FactIdentityDetectorReversedPeopleDict])
	assert.Len(t, id.PeopleDict, 4)
	assert.Len(t, id.ReversedPeopleDict, 3)
	assert.Equal(t, id.ReversedPeopleDict[0], "Egor")
	assert.Equal(t, facts[FactIdentityDetectorPeopleCount], 2)
	delete(facts, FactIdentityDetectorPeopleDict)
	delete(facts, FactIdentityDetectorReversedPeopleDict)
	delete(facts, ConfigIdentityDetectorPeopleDictPath)
	commits := make([]*object.Commit, 0)
	iter, err := testRepository.CommitObjects()
	commit, err := iter.Next()
	for ; err != io.EOF; commit, err = iter.Next() {
		if err != nil {
			panic(err)
		}
		commits = append(commits, commit)
	}
	facts["commits"] = commits
	id = fixtureIdentityDetector()
	id.PeopleDict = nil
	id.ReversedPeopleDict = nil
	id.Configure(facts)
	assert.Equal(t, id.PeopleDict, facts[FactIdentityDetectorPeopleDict])
	assert.Equal(t, id.ReversedPeopleDict, facts[FactIdentityDetectorReversedPeopleDict])
	assert.True(t, len(id.PeopleDict) >= 3)
	assert.True(t, len(id.ReversedPeopleDict) >= 4)
}

func TestIdentityDetectorRegistration(t *testing.T) {
	tp, exists := Registry.registered[(&IdentityDetector{}).Name()]
	assert.True(t, exists)
	assert.Equal(t, tp.Elem().Name(), "IdentityDetector")
	tps, exists := Registry.provided[(&IdentityDetector{}).Provides()[0]]
	assert.True(t, exists)
	assert.Len(t, tps, 1)
	assert.Equal(t, tps[0].Elem().Name(), "IdentityDetector")
}

func TestIdentityDetectorConfigureEmpty(t *testing.T) {
	id := IdentityDetector{}
	assert.Panics(t, func() { id.Configure(map[string]interface{}{}) })
}

func TestIdentityDetectorConsume(t *testing.T) {
	commit, _ := testRepository.CommitObject(plumbing.NewHash(
		"5c0e755dd85ac74584d9988cc361eccf02ce1a48"))
	deps := map[string]interface{}{}
	deps["commit"] = commit
	res, err := fixtureIdentityDetector().Consume(deps)
	assert.Nil(t, err)
	assert.Equal(t, res["author"].(int), 0)
	commit, _ = testRepository.CommitObject(plumbing.NewHash(
		"8a03b5620b1caa72ec9cb847ea88332621e2950a"))
	deps["commit"] = commit
	res, err = fixtureIdentityDetector().Consume(deps)
	assert.Nil(t, err)
	assert.Equal(t, res["author"].(int), MISSING_AUTHOR)
}

func TestLoadPeopleDict(t *testing.T) {
	id := fixtureIdentityDetector()
	err := id.LoadPeopleDict(path.Join("test_data", "identities"))
	assert.Nil(t, err)
	assert.Equal(t, len(id.PeopleDict), 7)
	assert.Contains(t, id.PeopleDict, "linus torvalds")
	assert.Contains(t, id.PeopleDict, "torvalds@linux-foundation.org")
	assert.Contains(t, id.PeopleDict, "vadim markovtsev")
	assert.Contains(t, id.PeopleDict, "vadim@sourced.tech")
	assert.Contains(t, id.PeopleDict, "another@one.com")
	assert.Contains(t, id.PeopleDict, "máximo cuadros")
	assert.Contains(t, id.PeopleDict, "maximo@sourced.tech")
	assert.Equal(t, len(id.ReversedPeopleDict), 4)
	assert.Equal(t, id.ReversedPeopleDict[0], "Linus Torvalds")
	assert.Equal(t, id.ReversedPeopleDict[1], "Vadim Markovtsev")
	assert.Equal(t, id.ReversedPeopleDict[2], "Máximo Cuadros")
	assert.Equal(t, id.ReversedPeopleDict[3], UNMATCHED_AUTHOR)
}

/*
// internal compiler error in 1.8
func TestGeneratePeopleDict(t *testing.T) {
	id := fixtureIdentityDetector()
	commits := make([]*object.Commit, 0)
	iter, err := testRepository.CommitObjects()
	for ; err != io.EOF; commit, err := iter.Next() {
		if err != nil {
			panic(err)
		}
		commits = append(commits, commit)
	}
	id.GeneratePeopleDict(commits)
}
*/

func TestGeneratePeopleDict(t *testing.T) {
	id := fixtureIdentityDetector()
	commits := make([]*object.Commit, 0)
	iter, err := testRepository.CommitObjects()
	commit, err := iter.Next()
	for ; err != io.EOF; commit, err = iter.Next() {
		if err != nil {
			panic(err)
		}
		commits = append(commits, commit)
	}
	{
		i := 0
		for ; commits[i].Author.Name != "Vadim Markovtsev"; i++ {
		}
		if i > 0 {
			commit := commits[0]
			commits[0] = commits[i]
			commits[i] = commit
		}
		i = 1
		for ; commits[i].Author.Name != "Alexander Bezzubov"; i++ {
		}
		if i > 0 {
			commit := commits[1]
			commits[1] = commits[i]
			commits[i] = commit
		}
		i = 2
		for ; commits[i].Author.Name != "Máximo Cuadros"; i++ {
		}
		if i > 0 {
			commit := commits[2]
			commits[2] = commits[i]
			commits[i] = commit
		}
	}
	id.GeneratePeopleDict(commits)
	assert.True(t, len(id.PeopleDict) >= 7)
	assert.True(t, len(id.ReversedPeopleDict) >= 3)
	assert.Equal(t, id.PeopleDict["vadim markovtsev"], 0)
	assert.Equal(t, id.PeopleDict["vadim@sourced.tech"], 0)
	assert.Equal(t, id.PeopleDict["gmarkhor@gmail.com"], 0)
	assert.Equal(t, id.PeopleDict["alexander bezzubov"], 1)
	assert.Equal(t, id.PeopleDict["bzz@apache.org"], 1)
	assert.Equal(t, id.PeopleDict["máximo cuadros"], 2)
	assert.Equal(t, id.PeopleDict["mcuadros@gmail.com"], 2)
	assert.Equal(t, id.ReversedPeopleDict[0], "vadim markovtsev|gmarkhor@gmail.com|vadim@sourced.tech")
	assert.Equal(t, id.ReversedPeopleDict[1], "alexander bezzubov|bzz@apache.org")
	assert.Equal(t, id.ReversedPeopleDict[2], "máximo cuadros|mcuadros@gmail.com")
	assert.NotEqual(t, id.ReversedPeopleDict[len(id.ReversedPeopleDict)-1], UNMATCHED_AUTHOR)
}

func TestLoadPeopleDictInvalidPath(t *testing.T) {
	id := fixtureIdentityDetector()
	ipath := "/xxxyyyzzzInvalidPath!hehe"
	err := id.LoadPeopleDict(ipath)
	assert.NotNil(t, err)
	assert.Equal(t, err.(*os.PathError).Path, ipath)
}

type fakeBlobEncodedObject struct {
	Contents string
}

func (obj fakeBlobEncodedObject) Hash() plumbing.Hash {
	return plumbing.NewHash("ffffffffffffffffffffffffffffffffffffffff")
}

func (obj fakeBlobEncodedObject) Type() plumbing.ObjectType {
	return plumbing.BlobObject
}

func (obj fakeBlobEncodedObject) SetType(plumbing.ObjectType) {}

func (obj fakeBlobEncodedObject) Size() int64 {
	return int64(len(obj.Contents))
}

func (obj fakeBlobEncodedObject) SetSize(int64) {}

func (obj fakeBlobEncodedObject) Reader() (io.ReadCloser, error) {
	return ioutil.NopCloser(strings.NewReader(obj.Contents)), nil
}

func (obj fakeBlobEncodedObject) Writer() (io.WriteCloser, error) {
	return nil, nil
}

type fakeTreeEncodedObject struct {
	Name string
}

func (obj fakeTreeEncodedObject) Hash() plumbing.Hash {
	return plumbing.NewHash("ffffffffffffffffffffffffffffffffffffffff")
}

func (obj fakeTreeEncodedObject) Type() plumbing.ObjectType {
	return plumbing.TreeObject
}

func (obj fakeTreeEncodedObject) SetType(plumbing.ObjectType) {}

func (obj fakeTreeEncodedObject) Size() int64 {
	return 1
}

func (obj fakeTreeEncodedObject) SetSize(int64) {}

func (obj fakeTreeEncodedObject) Reader() (io.ReadCloser, error) {
	return ioutil.NopCloser(strings.NewReader(
		"100644 " + obj.Name + "\x00ffffffffffffffffffffffffffffffffffffffff")), nil
}

func (obj fakeTreeEncodedObject) Writer() (io.WriteCloser, error) {
	return nil, nil
}

type fakeEncodedObjectStorer struct {
	Name     string
	Contents string
}

func (strr fakeEncodedObjectStorer) NewEncodedObject() plumbing.EncodedObject {
	return nil
}

func (strr fakeEncodedObjectStorer) HasEncodedObject(plumbing.Hash) error {
	return nil
}

func (strr fakeEncodedObjectStorer) SetEncodedObject(plumbing.EncodedObject) (plumbing.Hash, error) {
	return plumbing.NewHash("0000000000000000000000000000000000000000"), nil
}

func (strr fakeEncodedObjectStorer) EncodedObject(objType plumbing.ObjectType, hash plumbing.Hash) (plumbing.EncodedObject, error) {
	if objType == plumbing.TreeObject {
		return fakeTreeEncodedObject{Name: strr.Name}, nil
	} else if objType == plumbing.BlobObject {
		return fakeBlobEncodedObject{Contents: strr.Contents}, nil
	}
	return nil, nil
}

func (strr fakeEncodedObjectStorer) IterEncodedObjects(plumbing.ObjectType) (storer.EncodedObjectIter, error) {
	return nil, nil
}

func getFakeCommitWithFile(name string, contents string) *object.Commit {
	c := object.Commit{
		Hash: plumbing.NewHash("ffffffffffffffffffffffffffffffffffffffff"),
		Author: object.Signature{
			Name:  "Vadim Markovtsev",
			Email: "vadim@sourced.tech",
		},
		Committer: object.Signature{
			Name:  "Vadim Markovtsev",
			Email: "vadim@sourced.tech",
		},
		Message:  "Virtual file " + name,
		TreeHash: plumbing.NewHash("ffffffffffffffffffffffffffffffffffffffff"),
	}
	voc := reflect.ValueOf(&c)
	voc = voc.Elem()
	f := voc.FieldByName("s")
	ptr := unsafe.Pointer(f.UnsafeAddr())
	strr := fakeEncodedObjectStorer{Name: name, Contents: contents}
	*(*storer.EncodedObjectStorer)(ptr) = strr
	return &c
}

func TestGeneratePeopleDictMailmap(t *testing.T) {
	id := fixtureIdentityDetector()
	commits := make([]*object.Commit, 0)
	iter, err := testRepository.CommitObjects()
	commit, err := iter.Next()
	for ; err != io.EOF; commit, err = iter.Next() {
		if err != nil {
			panic(err)
		}
		commits = append(commits, commit)
	}
	fake := getFakeCommitWithFile(
		".mailmap",
		"Strange Guy <vadim@sourced.tech>\nVadim Markovtsev <vadim@sourced.tech> Strange Guy <vadim@sourced.tech>")
	commits = append(commits, fake)
	id.GeneratePeopleDict(commits)
	assert.Contains(t, id.ReversedPeopleDict,
		"strange guy|vadim markovtsev|gmarkhor@gmail.com|vadim@sourced.tech")
}
