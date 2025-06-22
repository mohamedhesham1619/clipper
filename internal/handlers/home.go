package handlers

import (
	"clipper/web"
	"net/http"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	
	w.Write(web.UI)
}
