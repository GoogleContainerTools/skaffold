package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"
	"tools/linters"
)


func main() {
	// TODO:(marlon) to add allowList of files which can import logrus.
	singlechecker.Main(linters.LogrusAnalyzer)
}
