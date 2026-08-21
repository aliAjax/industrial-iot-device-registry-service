package jsonutil

import (
	"encoding/json"
	"io"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, value any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(value)
}

func ReadJSON(r io.Reader, value any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	return dec.Decode(value)
}

func Error(w http.ResponseWriter, status int, code, message string) error {
	return WriteJSON(w, status, map[string]string{"error": code, "message": message})
}
