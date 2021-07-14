package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("Oh hey Ernie.")

		time.Sleep(time.Second * 2)
	}
}
