package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("Hello Docker!")

		time.Sleep(time.Second * 1)
	}
}
