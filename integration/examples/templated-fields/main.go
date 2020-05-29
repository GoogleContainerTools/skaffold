package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	for {
		fmt.Printf("Running image %v:%v\n", os.Getenv("FOO_IMAGE_REPO"), os.Getenv("FOO_IMAGE_TAG"))
		time.Sleep(time.Second * 1)
	}
}
