package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchUsesEmbeddingThenVectorFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/embeddings":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": map[string]any{"data": []map[string]any{{"embedding": []float64{0.2, 0.8}}}}})
		case "/v1/vector/query":
			if body["collection"] != "creator-content" || body["include_metadata"] != true {
				t.Fatalf("unexpected query: %#v", body)
			}
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": map[string]any{"matches": []map[string]any{{"id": "post-7", "score": 0.91, "metadata": map[string]any{"title": "Studio notes"}}}}})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := &InfraClient{BaseURL: server.URL, Key: "test-key", HTTP: server.Client()}
	got, err := client.Search(SearchRequest{Creator: "ada", Query: "lighting setup"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title != "Studio notes" || got[0].Score < 0.9 {
		t.Fatalf("unexpected result: %#v", got)
	}
}
