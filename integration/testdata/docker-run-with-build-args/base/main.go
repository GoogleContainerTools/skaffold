package main

import (
	"fmt"
)

var ImageRepo = "unknown"
var ImageTag = "unknown"
var ImageName = "unknown"

func main() {
	fmt.Printf("IMAGE_REPO: %s, IMAGE_NAME: %s, IMAGE_TAG:%s", ImageRepo, ImageName, ImageTag)
}
