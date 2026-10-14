//go:build !linux && !tamago

package main

import "fmt"

func waitEnter() {
	fmt.Println("press enter to confirm")
}
