package main

import (
	"fmt"
	"github.com/maynagashev/go-metrics/internal/server/router"
	"github.com/maynagashev/go-metrics/internal/storage"
	"net/http"
)

func main() {
	parseFlags()
	fmt.Printf("Starting server on %s\n", flagRunAddr)
	err := http.ListenAndServe(flagRunAddr, router.New(storage.New()))
	if err != nil {
		fmt.Printf("error starting server: %s\n", err)
	}
}
