package helpers

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func RemoveFile(path string) error {
	if FileExists(path) {
		return os.Remove(path)
	}
	return nil
}

func CopyFile(src string, dest string) (err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return
	}
	defer srcFile.Close()

	srcStat, err := srcFile.Stat()
	if err != nil {
		return
	}

	if !srcStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file.", src)
	}

	destFile, err := os.Create(dest)
	if err != nil {
		return
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return
}

func CopyDir(src string, dest string) (err error) {
	srcStat, err := os.Stat(src)
	if err != nil {
		return
	}

	if !srcStat.Mode().IsDir() {
		return fmt.Errorf("%s is not a directory.", src)
	}

	_, err = os.Stat(dest)
	if !os.IsNotExist(err) {
		return fmt.Errorf("Destination %s already exists.", dest)
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}

	err = os.MkdirAll(dest, srcStat.Mode())
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.Mode().IsDir() {
			err = CopyDir(srcPath, destPath)
		} else {
			err = CopyFile(srcPath, destPath)
		}
		if err != nil {
			return
		}
	}

	return
}

//RemoveFilesWithPattern ...
func RemoveFilesWithPattern(targetDir, pattern string) error {
	r, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	files, err := ioutil.ReadDir(targetDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if r.MatchString(f.Name()) {
			err := os.RemoveAll(filepath.Join(targetDir, f.Name()))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
