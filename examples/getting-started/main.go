package main

import (
	"fmt"
        "os"
	"time"
)

func main() {
	for {
		fmt.Println("Hello world!!!!!")
                os.Exit(1)
		time.Sleep(time.Second * 1)
	}
}
