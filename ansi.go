package main

import (
	"fmt"
)

func moveCursor(x, y int) {
	fmt.Printf("\x1b[%d;%dH", y+1, x+1)
}
