package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("Hello world!")

	_, err := os.Create("a/b/c/sss.txt")
	if err != nil {
		fmt.Println(err)
	}
	time.Sleep(time.Second * 1)
}
