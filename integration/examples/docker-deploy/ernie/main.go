package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("Hey there Bert!")

		time.Sleep(time.Second * 2)
	}
}
