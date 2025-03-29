package testhelpers

import (
	"os"
)

func (a *AssertionManager) FileExists(filePath string) {
	a.testObject.Helper()
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			a.testObject.Fatalf("Expected file %s to exist.", filePath)
		} else {
			a.testObject.Fatal("Unexpected error:", err.Error())
		}
	}
}

func (a *AssertionManager) FileIsNotEmpty(filePath string) {
	a.testObject.Helper()
	info, err := os.Stat(filePath)
	if err != nil {
		a.testObject.Fatal("Unexpected error:", err.Error())
	}

	if info.IsDir() {
		a.testObject.Fatalf("File %s is a directory.", filePath)
	}

	if info.Size() == 0 {
		a.testObject.Fatalf("Expected file %s to not be empty.", filePath)
	}
}
