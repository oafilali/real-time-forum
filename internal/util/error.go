package util

import (
	"log"
	"net/http"
)

func ErrorCheckHandlers(w http.ResponseWriter, msg string, err error, code int) bool {
	if err != nil {
		http.Error(w, msg, code)
		log.Println(msg, err)
		return true
	}
	return false
}
