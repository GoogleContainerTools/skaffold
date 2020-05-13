package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("Hello world 2!")

		time.Sleep(time.Second * 1)
	}
}
