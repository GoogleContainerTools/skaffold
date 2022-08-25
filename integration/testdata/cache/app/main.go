package main

import (
	"fmt"
	"io/ioutil"
	"time"
)

func main() {
	dat, err := ioutil.ReadFile("hello.txt")
	if err != nil {
		panic(err)
	}

	for {
		fmt.Println(string(dat))
		time.Sleep(time.Second * 1)
	}
}
