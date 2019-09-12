package main

import (
	"github.com/GoogleContainerTools/skaffold/hack/versions/pkg/schema_check"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := schema_check.RunSchemaCheckOnChangedFiles(); err != nil {
		logrus.Fatal(err)
	}
}
