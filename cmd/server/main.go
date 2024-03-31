package main

import (
	"fmt"
	"github.com/maynagashev/go-metrics/internal/server/router"
	"github.com/maynagashev/go-metrics/internal/storage"
	"net/http"
)

func main() {
	err := http.ListenAndServe(":8080", router.New(storage.New()))
	if err != nil {
		fmt.Printf("error starting server: %s\n", err)
	}
}
