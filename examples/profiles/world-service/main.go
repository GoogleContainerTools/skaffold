package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("WORLD")

		time.Sleep(time.Second * 1)
	}
}
