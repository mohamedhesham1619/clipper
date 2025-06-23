package handlers

import (
	"clipper/internal/web"
	"net/http"
)

// homeHandler serves the main UI page
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(web.UI)
}
