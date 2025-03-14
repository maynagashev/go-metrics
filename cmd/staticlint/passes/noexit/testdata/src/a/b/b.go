// Package b is a test package for the noexit analyzer.
package b

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello from package b!")
	os.Exit(0) // This is fine, not in package main
}

func otherFunc() {
	os.Exit(1) // This is also fine
}
