package http

import (
	"fmt"
	"net/http"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	// TODO(ai) import more sophisticated health mechanism from internal libraries
	fmt.Fprintf(w, "OK\n")
}
