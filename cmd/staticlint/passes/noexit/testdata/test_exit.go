// Package main тестовый пакет для анализатора noexit
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
