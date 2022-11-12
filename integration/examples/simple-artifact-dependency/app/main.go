package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	dat, err := os.ReadFile("hello.txt")
	if err != nil {
		panic(err)
	}

	for {
		fmt.Println(string(dat))
		time.Sleep(time.Second * 1)
	}
}
