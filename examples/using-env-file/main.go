package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	message := os.Getenv("MESSAGE")
	for {
		fmt.Println(message)
		time.Sleep(time.Second * 1)
	}
}
