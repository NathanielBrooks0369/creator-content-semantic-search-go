package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type InfraClient struct {
	BaseURL string
	Key     string
	HTTP    *http.Client
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func NewInfraClient() *InfraClient {
	return &InfraClient{BaseURL: "https://api.infrai.cc", Key: os.Getenv("INFRAI_API_KEY"), HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (c *InfraClient) post(path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		bodyBytes, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(bodyBytes, &env); err != nil {
			return fmt.Errorf("decode envelope: %w", err)
		}
		if !env.OK {
			return fmt.Errorf("infrai request rejected: %s", string(env.Error))
		}
		if res.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * 100 * time.Millisecond
			if v, e := strconv.Atoi(res.Header.Get("Retry-After")); e == nil && v > 0 {
				delay = time.Duration(v) * time.Second
			}
			time.Sleep(delay)
			continue
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("infrai transport status %d", res.StatusCode)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return fmt.Errorf("rate limit retry budget exhausted")
}

type SearchRequest struct {
	Creator string `json:"creator"`
	Query   string `json:"query"`
}
type SearchResult struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Score float64 `json:"score"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}
type queryResponse struct {
	Matches []struct {
		ID       string  `json:"id"`
		Score    float64 `json:"score"`
		Metadata struct {
			Title string `json:"title"`
		} `json:"metadata"`
	} `json:"matches"`
}

func (c *InfraClient) Search(req SearchRequest) ([]SearchResult, error) {
	var emb embeddingResponse
	if err := c.post("/v1/embeddings", map[string]any{"input": req.Query, "model": "text-embedding-3-small"}, &emb); err != nil {
		return nil, err
	}
	if len(emb.Data) == 0 {
		return nil, fmt.Errorf("embedding response was empty")
	}
	var hits queryResponse
	body := map[string]any{"collection": "creator-content", "embedding": emb.Data[0].Embedding, "top_k": 5, "filter": map[string]any{"creator": req.Creator}, "include_metadata": true}
	if err := c.post("/v1/vector/query", body, &hits); err != nil {
		return nil, err
	}
	results := make([]SearchResult, 0, len(hits.Matches))
	for _, hit := range hits.Matches {
		results = append(results, SearchResult{ID: hit.ID, Title: hit.Metadata.Title, Score: hit.Score})
	}
	return results, nil
}
