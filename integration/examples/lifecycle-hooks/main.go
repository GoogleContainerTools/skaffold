package main

import (
	"fmt"
	"io/ioutil"
	"time"
)

func main() {
	for {
		data, err := ioutil.ReadFile("hello.txt")
		if err != nil {
			fmt.Printf("failed to read file hello.txt: %v", err)
		}
		fmt.Println(string(data))
		time.Sleep(time.Second * 1)
	}
}
