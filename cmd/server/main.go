package main

import (
	"fmt"
	"github.com/maynagashev/go-metrics/internal/handlers"
	"github.com/maynagashev/go-metrics/internal/storage"
	"net/http"
)

func main() {
	memStorage := storage.NewMemStorage()

	mux := http.NewServeMux()
	mux.HandleFunc("/update/", func(w http.ResponseWriter, r *http.Request) {
		handlers.Update(w, r, memStorage)
	})

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Printf("error starting server: %s\n", err)
	}

}
