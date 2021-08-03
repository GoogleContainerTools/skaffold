package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)
	data, err := ioutil.ReadFile("hello.txt")
	if err != nil {
		fmt.Printf("failed to read file hello.txt: %v", err)
	}
	go func() {
		for range sigs {
			fmt.Printf("received SIGHUP signal. Reloading file hello.txt\n")
			data, err = ioutil.ReadFile("hello.txt")
			if err != nil {
				fmt.Printf("failed to read file hello.txt: %v", err)
			}
		}
	}()
	for {
		fmt.Println(string(data))
		time.Sleep(time.Second * 1)
	}
}
