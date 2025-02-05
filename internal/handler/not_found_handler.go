package handler

import (
	"forum/internal/util"
	"net/http"
)

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	util.ErrorHandler(w, r, http.StatusNotFound, "Page Not Found")
}
