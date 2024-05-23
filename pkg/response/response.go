package response

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

const (
	StatusOK    = "OK"
	StatusError = "Error"
)

func OK(w http.ResponseWriter, msg string) {
	resp := Response{
		Status:  StatusOK,
		Message: msg,
	}
	writeResponse(w, resp, http.StatusOK)
}

func Error(w http.ResponseWriter, err error, statusCode int) {
	resp := Response{
		Status: StatusError,
		Error:  err.Error(),
	}
	writeResponse(w, resp, statusCode)
}

// Стандартные ответы в json формате.
func writeResponse(w http.ResponseWriter, resp Response, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Кодируем структуру в json.
	encoded, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем ответ.
	_, err = fmt.Fprint(w, string(encoded))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
