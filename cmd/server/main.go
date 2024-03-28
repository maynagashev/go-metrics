package main

import (
	"fmt"
	"github.com/maynagashev/go-metrics/internal/handlers"
	"net/http"
)

func main() {

	mux := http.NewServeMux()
	mux.HandleFunc("/update/", handlers.Update)

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Println(fmt.Errorf("error starting server: %w", err))
	}

}
