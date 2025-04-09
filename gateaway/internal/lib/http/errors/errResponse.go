package errors

import "net/http"

func Error(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
