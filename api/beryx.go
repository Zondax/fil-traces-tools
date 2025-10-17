package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

const BeryxURL = "https://api.zondax.ch/fil/data/v4/mainnet"

// Transaction represents a single transaction in the response
type Transaction struct {
	Height    int64 `json:"height"`
	Canonical bool  `json:"canonical"`
}

// TransactionsResponse represents the API response structure
type TransactionsResponse struct {
	Transactions []Transaction `json:"transactions"`
	NextCursor   string        `json:"next_cursor"`
	TotalItems   int           `json:"total_items"`
	TotalTxs     int           `json:"total_txs"`
}

type Beryx struct {
	token string
}

func NewBeryx(token string) *Beryx {
	return &Beryx{
		token: token,
	}
}

// GetAddressEvents fetches transaction events for a given address
func (b *Beryx) GetAddressEventHeights(ctx context.Context, address string) ([]int64, error) {
	// Construct the URL
	url := fmt.Sprintf("%s/transactions/address/%s", BeryxURL, address)

	// Create the HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.token))
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if resp != nil {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("beryx failed to close response body: %v\n", err)
			}
		}
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response
	var result TransactionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	var heights []int64
	for _, transaction := range result.Transactions {
		if !transaction.Canonical {
			continue
		}
		heights = append(heights, transaction.Height)
	}
	// sort heights in ascending order
	sort.Slice(heights, func(i, j int) bool {
		return heights[i] < heights[j]
	})

	return heights, nil
}
