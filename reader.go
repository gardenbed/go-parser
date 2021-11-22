package parser

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// getModuleName returns the name of go module from a given path.
func getModuleName(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	filename := filepath.Join(path, "go.mod")

	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			if parent := filepath.Dir(path); parent != "/" {
				return getModuleName(parent)
			}
		}
		return "", err
	}

	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := scanner.Text(); strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", errors.New("invalid go.mod file: no module name found")
}

type visitFunc func(baseDir, relDir string) error

// visitPackages traverses all packages from a given path.
func visitPackages(includeSubs bool, path string, visit visitFunc) error {
	// Verify the path
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", path)
	}

	return visitPackagesRecursively(includeSubs, path, ".", visit)
}

func visitPackagesRecursively(includeSubs bool, basePath, relPath string, visit visitFunc) error {
	// First, visit the current package
	if err := visit(basePath, relPath); err != nil {
		return err
	}

	// Then, visit all packages inside the current package
	if includeSubs {
		files, err := ioutil.ReadDir(filepath.Join(basePath, relPath))
		if err != nil {
			return err
		}

		for _, file := range files {
			if file.IsDir() && isPackageDir(file.Name()) {
				subRelPath := filepath.Join(relPath, file.Name())
				if err := visitPackagesRecursively(includeSubs, basePath, subRelPath, visit); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// This helper function determines if a directory is a package directory and should be further traversed.
func isPackageDir(name string) bool {
	// Ignore directories starting with a dot (.git, .github, .build, etc)
	startsWithDot := strings.HasPrefix(name, ".")

	// Ignore build directories
	isBuildDir := name == "bin" || name == "build"

	return !startsWithDot && !isBuildDir
}
