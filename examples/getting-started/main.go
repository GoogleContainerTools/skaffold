package main

import (
	"fmt"
	"time"
)

type HI struct {
}

func main() {
	for {
		fmt.Println("Hello world!!!")

		time.Sleep(time.Second * 1)
	}
}
