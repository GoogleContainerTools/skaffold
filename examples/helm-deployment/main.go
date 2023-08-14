package main

import (
	"fmt"
	"time"
)

func main() {
	for counter := 0; ; counter++ {
		fmt.Println("Hello222222333333333 world!", counter)

		time.Sleep(time.Second * 1)
	}
}
