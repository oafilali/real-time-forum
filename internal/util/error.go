package util

import (
	"log"
	"net/http"
)

func ErrorCheckHandlers(w http.ResponseWriter, r *http.Request, msg string, err error, code int) bool {
	if err != nil {
		log.Println(msg, err)
		ErrorHandler(w, r, code, msg)
		return true
	}
	return false
}

func ErrorHandler(w http.ResponseWriter, r *http.Request, errorCode int, errorMessage string) {
	data := struct {
		ErrorCode    int
		ErrorMessage string
	}{
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	}

	err := Templates.ExecuteTemplate(w, "error.html", data)
	if err != nil {
		log.Println("Error loading the page:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(errorCode)
}
