package main

import (
	"fmt"
	"time"
)

const (
	startingEmoji = '🐰'
	maxEmoji      = 200

	colorReset = "\u001b[0m"
)

var colors = []string{
	"\u001b[31;1m", // red bold
	"\u001b[33;1m", // yellow bold
	"\u001b[32;1m", // green bold
	"\u001b[34;1m", // blue bold
	"\u001b[36;1m", // cyan bold
	"\u001b[35;1m", // magenta bold
}

func main() {
	for {
		s := ""
		for i := 0; i < maxEmoji; i++ {
			s += fmt.Sprintf("%c%sHELLO WORLD%s", startingEmoji+i, colors[i%len(colors)], colorReset)
		}

		fmt.Println(s)

		time.Sleep(time.Second * 1)
	}
}
