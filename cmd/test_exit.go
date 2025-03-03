// Package main is a test package for the noexit analyzer.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello, world!")
	// os.Exit(0) - removed
	otherFunc()
}

func otherFunc() {
	// This is fine, not in main
	os.Exit(1)
}
