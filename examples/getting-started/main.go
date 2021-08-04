package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("Hello world!")

		time.Sleep(time.Hour * 1)
	}
}
