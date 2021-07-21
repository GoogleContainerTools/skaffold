package main

import (
	"os"
	"path/filepath"
)

func validateBinaries(binpaths []string) error {
	for _, binpath := range binpaths {
		_, err := os.Stat(binpath)
		if err != nil {
			return err
		}
	}
	return nil

}

func validateExampleDir(exampleName string) error {
	_, err := os.Stat(filepath.Join("../../examples/", exampleName))
	if err != nil {
		return err
	}
	return nil
}

func validateArgs(args []string) error {
	if err := validateBinaries(args[1 : len(args)-2]); err != nil {
		return err
	}
	if err := validateExampleDir(args[len(args)-2]); err != nil {
		return err
	}

	return nil
}
