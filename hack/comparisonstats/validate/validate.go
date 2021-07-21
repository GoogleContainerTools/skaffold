package validate

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const NumBinaries = 2

func ValidateArgs(args []string) error {
	if len(args) < NumBinaries+1 {
		logrus.Fatalf("comparisonstats expects input of the form: $ comparisonstats /usr/bin/bin1 /usr/bin/bin2 helm-deployment main.go")
	}

	if err := validateBinaries(args[1:NumBinaries]); err != nil {
		return err
	}

	if err := validateExampleAppNameAndSrcFile(args[NumBinaries], args[1+NumBinaries]); err != nil {
		return err
	}

	return nil
}

func validateBinaries(binpaths []string) error {
	for _, binpath := range binpaths {
		_, err := os.Stat(binpath)
		if err != nil {
			return err
		}
	}
	return nil

}

func validateExampleAppNameAndSrcFile(exampleAppName, exampleSrcFile string) error {
	fp := filepath.Join("examples/", exampleAppName)
	if filepath.IsAbs(exampleAppName) {
		fp = exampleAppName
	}
	_, err := os.Stat(fp)
	if err != nil {
		return err
	}

	fp = filepath.Join("examples/", exampleAppName, exampleSrcFile)
	if filepath.IsAbs(exampleAppName) {
		fp = exampleAppName
	}
	_, err = os.Stat(fp)
	if err != nil {
		return err
	}
	return nil
}
