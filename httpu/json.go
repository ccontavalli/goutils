package httpu

import (
	"encoding/json"
	"net/http"
)

// Sends a json reply via an http writer.
func SendJsonReply(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}
