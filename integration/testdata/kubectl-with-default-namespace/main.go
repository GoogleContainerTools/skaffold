package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	for {
		fmt.Printf("Hello world! Running on %s/%s\n", runtime.GOOS, runtime.GOARCH)

		time.Sleep(time.Second * 1)
	}
}
