package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Printf("hello multi platform, I am %s-%s\n", runtime.GOOS, runtime.GOARCH) 

		time.Sleep(time.Second * 1)
	}
}
