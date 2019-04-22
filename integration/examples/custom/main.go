package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("Hello bazel!S!")
		time.Sleep(time.Second * 1)
	}
}
