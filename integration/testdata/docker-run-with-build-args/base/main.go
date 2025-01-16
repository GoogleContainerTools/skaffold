package main

import (
	"fmt"
)

var ImageRepo = "unknown"
var ImageTag = "unknown"
var ImageName = "unknown"

func main() {
	output := fmt.Sprintf("IMAGE_REPO: %s, IMAGE_NAME: %s, IMAGE_TAG:%s\n", ImageRepo, ImageName, ImageTag)
	fmt.Println(output)
}
