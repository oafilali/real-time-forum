package util

import (
	"forum/internal/handler"
	"log"
	"net/http"
)

func ErrorCheckHandlers(w http.ResponseWriter, r *http.Request, msg string, err error, code int) bool {
	if err != nil {
		log.Println(msg, err)
		handler.ErrorHandler(w, r, code, msg)
		return true
	}
	return false
}
