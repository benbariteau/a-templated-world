package main

import (
	"fmt"
	_ "image"
	_ "image/draw"
	"os"
)

func main() {
	args := os.Args
	if len(args) != 2 {
		fmt.Printf("usage:\t%v [path/to/sources]\n", args[0])
		return
	}
}
