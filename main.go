package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	client := NewInfraClient()
	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var in SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		results, err := client.Search(in)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"results": results})
	})
	log.Println("creator search listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
