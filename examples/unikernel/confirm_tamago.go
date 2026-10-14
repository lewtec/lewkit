//go:build tamago

package main

import (
	"fmt"
	_ "unsafe"
)

//go:linkname consoleGetc runtime/goos.Getc
func consoleGetc() byte

func waitEnter() {
	fmt.Println("press enter to confirm")
	for {
		b := consoleGetc()
		if b == '\n' || b == '\r' {
			return
		}
	}
}
