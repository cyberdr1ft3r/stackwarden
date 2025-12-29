package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Health struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Health{Status: "ok", Service: "agent"})
	})

	addr := "127.0.0.1:9091"
	log.Printf("agent listening on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
