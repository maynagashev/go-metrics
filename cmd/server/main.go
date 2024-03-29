package main

import (
	"fmt"
	"github.com/maynagashev/go-metrics/internal/handlers/update"
	"github.com/maynagashev/go-metrics/internal/storage/memory"
	"net/http"
)

func main() {
	memStorage := memory.New()

	mux := http.NewServeMux()
	mux.HandleFunc("/update/", update.New(memStorage))

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Printf("error starting server: %s\n", err)
	}

}
