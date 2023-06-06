package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	seconds := 0
	env := os.Getenv("FOO")
	for seconds < 5 {
		fmt.Printf("Hello world %v! %v\n", env, seconds)
		seconds++
		time.Sleep(time.Second * 1)
	}
}
