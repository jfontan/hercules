package hercules

import (
	"bufio"
	"bytes"
	"errors"
	"unicode/utf8"

	"github.com/sergi/go-diff/diffmatchpatch"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/utils/merkletrie"
)

// FileDiff calculates the difference of files which were modified.
type FileDiff struct {
}

type FileDiffData struct {
	OldLinesOfCode int
	NewLinesOfCode int
	Diffs          []diffmatchpatch.Diff
}

func (diff *FileDiff) Name() string {
	return "FileDiff"
}

func (diff *FileDiff) Provides() []string {
	arr := [...]string{"file_diff"}
	return arr[:]
}

func (diff *FileDiff) Requires() []string {
	arr := [...]string{"changes", "blob_cache"}
	return arr[:]
}

func (diff *FileDiff) ListConfigurationOptions() []ConfigurationOption {
	return []ConfigurationOption{}
}

func (diff *FileDiff) Configure(facts map[string]interface{}) {}

func (diff *FileDiff) Initialize(repository *git.Repository) {}

func (diff *FileDiff) Consume(deps map[string]interface{}) (map[string]interface{}, error) {
	result := map[string]FileDiffData{}
	cache := deps["blob_cache"].(map[plumbing.Hash]*object.Blob)
	tree_diff := deps["changes"].(object.Changes)
	for _, change := range tree_diff {
		action, err := change.Action()
		if err != nil {
			return nil, err
		}
		switch action {
		case merkletrie.Modify:
			blob_from := cache[change.From.TreeEntry.Hash]
			blob_to := cache[change.To.TreeEntry.Hash]
			// we are not validating UTF-8 here because for example
			// git/git 4f7770c87ce3c302e1639a7737a6d2531fe4b160 fetch-pack.c is invalid UTF-8
			str_from, err := BlobToString(blob_from)
			if err != nil {
				return nil, err
			}
			str_to, err := BlobToString(blob_to)
			if err != nil {
				return nil, err
			}
			dmp := diffmatchpatch.New()
			src, dst, _ := dmp.DiffLinesToRunes(str_from, str_to)
			diffs := dmp.DiffMainRunes(src, dst, false)
			result[change.To.Name] = FileDiffData{
				OldLinesOfCode: len(src),
				NewLinesOfCode: len(dst),
				Diffs:          diffs,
			}
		default:
			continue
		}
	}
	return map[string]interface{}{"file_diff": result}, nil
}

func CountLines(file *object.Blob) (int, error) {
	if file == nil {
		return -1, errors.New("Blob is nil: probably not cached.")
	}
	reader, err := file.Reader()
	if err != nil {
		return -1, err
	}
	defer checkClose(reader)
	var scanner *bufio.Scanner
	buffer := make([]byte, bufio.MaxScanTokenSize)
	counter := 0
	for scanner == nil || scanner.Err() == bufio.ErrTooLong {
		if scanner != nil && !utf8.Valid(scanner.Bytes()) {
			return -1, errors.New("binary")
		}
		scanner = bufio.NewScanner(reader)
		scanner.Buffer(buffer, 0)
		for scanner.Scan() {
			if !utf8.Valid(scanner.Bytes()) {
				return -1, errors.New("binary")
			}
			counter++
		}
	}
	return counter, nil
}

func BlobToString(file *object.Blob) (string, error) {
	if file == nil {
		return "", errors.New("Blob is nil: probably not cached.")
	}
	reader, err := file.Reader()
	if err != nil {
		return "", err
	}
	defer checkClose(reader)
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	return buf.String(), nil
}

func init() {
	Registry.Register(&FileDiff{})
}
