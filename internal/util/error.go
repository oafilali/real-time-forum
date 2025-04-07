package util

import (
	"errors"
	"net/http"
)

// Custom errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// ErrorHandler manages generic error responses
func ErrorHandler(w http.ResponseWriter, r *http.Request, errorCode int, errorMessage string) {
	w.WriteHeader(errorCode)
	
	// In a real application, you might want to log the error
	data := struct {
		ErrorCode    int
		ErrorMessage string
	}{
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	}

	// Assuming Templates is defined in template.go
	err := Templates.ExecuteTemplate(w, "error.html", data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}