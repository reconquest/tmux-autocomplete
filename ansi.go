package main

import "fmt"

func cursorSequence(x, y int) string {
	return fmt.Sprintf("\x1b[%d;%dH", y+1, x+1)
}
