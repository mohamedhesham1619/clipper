package handlers

import (
	"clipper/internal/web"
	"net/http"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	
	w.Write(web.UI)
}
