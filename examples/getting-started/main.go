package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("Hello skaffold!")

		time.Sleep(time.Second * 1)
	}
}
