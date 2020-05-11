package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("HELLO")

		time.Sleep(time.Second * 1)
	}
}
