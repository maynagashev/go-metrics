package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maynagashev/go-metrics/internal/server/handlers/update"
	"github.com/maynagashev/go-metrics/internal/storage/memory"
	"net/http"
)

func main() {
	// Инициализация роутера
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Инициализация хранилища
	memStorage := memory.New()

	r.Post("/update/*", update.New(memStorage))

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		fmt.Printf("error starting server: %s\n", err)
	}

}
