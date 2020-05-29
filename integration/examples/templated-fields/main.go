package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	for {
		// FOO_IMAGE_REPO and FOO_IMAGE_TAG are defined in the Helm chart using Skaffold's templated IMAGE_REPO and IMAGE_TAG
		fmt.Printf("Running image %v:%v\n", os.Getenv("FOO_IMAGE_REPO"), os.Getenv("FOO_IMAGE_TAG"))
		time.Sleep(time.Second * 1)
	}
}
