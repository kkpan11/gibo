package utils

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
)

func RepoDir() string {
	override := os.Getenv("GIBO_BOILERPLATES")
	if len(override) > 0 {
		return override
	}
	return os.Getenv("HOME") + "/.gitignore-boilerplates"
}

func cloneRepo(repo string) error {
	err := os.MkdirAll(repo, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	_, err = git.PlainClone(repo, false, &git.CloneOptions{
		URL:   "https://github.com/github/gitignore.git",
		Depth: 1,
	})
	if err != nil && err != git.ErrRepositoryAlreadyExists {
		return err
	}
	return nil
}

func cloneIfNeeded() error {
	fileInfo, err := os.Stat(RepoDir())
	if os.IsNotExist(err) {
		err = cloneRepo(RepoDir())
		if err != nil {
			return err
		}
	} else if !fileInfo.IsDir() {
		return fmt.Errorf("%v exists but is not a directory", RepoDir())
	}
	return nil
}

func pathForBoilerplate(name string) (string, error) {
	filename := name + ".gitignore"
	var result string = ""
	filepath.WalkDir(RepoDir(), func(path string, d fs.DirEntry, err error) error {
		if filepath.Base(path) == filename {
			result = path
			// Exit WalkDir early, we've found our match
			return filepath.SkipAll
		}
		return nil
	})
	if len(result) > 0 {
		return result, nil
	}
	return "", fmt.Errorf("%v: boilerplate not found", name)
}

func PrintBoilerplate(name string) error {
	if err := cloneIfNeeded(); err != nil {
		return err
	}
	path, err := pathForBoilerplate(name)
	if err != nil {
		return err
	}

	relativePath := strings.TrimPrefix(path, RepoDir())

	// Get the revision hash for the current head
	r, err := git.PlainOpen(RepoDir())
	if err != nil {
		return err
	}
	revision, err := r.Head()
	if err != nil {
		return err
	}
	remoteWebRoot := "https://raw.github.com/github/gitignore/" + revision.Hash().String()

	fmt.Println("### Generated by gibo (https://github.com/simonwhitaker/gibo)")
	fmt.Printf("### %v%v\n\n", remoteWebRoot, relativePath)
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	io.Copy(os.Stdout, f)
	return nil
}

func ListBoilerplates() ([]string, error) {
	var result []string
	if err := cloneIfNeeded(); err != nil {
		return nil, err
	}
	err := filepath.WalkDir(RepoDir(), func(path string, d fs.DirEntry, err error) error {
		base := filepath.Base(path)
		ext := filepath.Ext(base)
		if ext == ".gitignore" {
			name := strings.TrimSuffix(base, ext)
			result = append(result, name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(result)
	return result, nil
}

func ListBoilerplatesNoError() []string {
	result, _ := ListBoilerplates()
	return result
}

func Update() error {
	cloneIfNeeded()
	r, err := git.PlainOpen(RepoDir())
	if err != nil {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	err = w.Pull(&git.PullOptions{})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}
	return nil
}

func PrintInColumns(list []string, width int) {
	maxLength := 0
	for _, s := range list {
		if len(s) > maxLength {
			maxLength = len(s)
		}
	}

	// For n columns, we need enough room for n-1 spaces between the columns.
	//
	// For a list where the longest string is of length maxLength, the maximum
	// number of columns we can print is the largest number, n, where:
	//
	//    n * maxLength + n - 1 <= width
	//
	// Rearranging:
	//
	//    n * maxLength + n <= width + 1
	//    n * (maxLength - 1) <= width + 1
	//    n <= (width + 1) / (maxLength + 1)
	numColumns := (width + 1) / (maxLength + 1)
	if numColumns == 0 {
		numColumns = 1
	}

	fmtString := fmt.Sprintf("%%-%ds", maxLength)
	for i, s := range list {
		fmt.Printf(fmtString, s)
		if (i%numColumns == numColumns-1) || i == len(list)-1 {
			fmt.Print("\n")
		} else {
			fmt.Print(" ")
		}
	}
}
