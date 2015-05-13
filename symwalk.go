package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
)

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entries.
func readDirNames(dirname string) ([]string, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

var LoopErr = errors.New("Loop in directory structure")

func hasLoop(path string,
	visited map[string]struct{},
	parents map[string]struct{},
) error {
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}

	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}

	if _, ok := parents[path]; ok {
		return LoopErr
	}

	if _, ok := visited[path]; ok {
		return nil
	}

	visited[path] = struct{}{}

	parents[path] = struct{}{}
	defer delete(parents, path)

	names, err := readDirNames(path)
	if err != nil {
		return err
	}

	for _, name := range names {
		filename := filepath.Join(path, name)
		if err := hasLoop(filename, visited, parents); err != nil {
			return err
		}
	}
	return nil
}

func HasLoop(path string) error {
	visited, parents := make(map[string]struct{}), make(map[string]struct{})
	if err := hasLoop(path, visited, parents); err != nil {
		return err
	}
	return nil
}

func walk(path string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	var err error

	evalPath := path
	if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		evalPath, err = filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		info, err = os.Lstat(evalPath)
	}

	if err = walkFn(path, info, err); err != nil {
		if info.IsDir() && err == filepath.SkipDir {
			return nil
		}
		return err
	}

	if !info.IsDir() {
		return nil
	}

	names, err := readDirNames(evalPath)
	if err != nil {
		return walkFn(path, info, err)
	}

	for _, name := range names {
		filename := filepath.Join(path, name)
		fileInfo, err := os.Lstat(filename)
		if err != nil {
			if err := walkFn(filename, fileInfo, err); err != nil && err != filepath.SkipDir {
				return err
			}
		} else {
			err = walk(filename, fileInfo, walkFn)
			if err != nil {
				if !fileInfo.IsDir() || err != filepath.SkipDir {
					return err
				}
			}
		}
	}
	return nil
}

func Walk(root string, walkFn filepath.WalkFunc) error {
	if err := HasLoop(root); err != nil {
		return err
	}

	info, err := os.Lstat(root)
	if err != nil {
		return walkFn(root, nil, err)
	}
	return walk(root, info, walkFn)
}

func main() {
	err := Walk("/home/me/test", func(path string, info os.FileInfo, err error) error {
		fmt.Println(path)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
