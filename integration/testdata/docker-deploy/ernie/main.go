package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("Hey there Ernie!")

		time.Sleep(time.Second * 2)
	}
}
