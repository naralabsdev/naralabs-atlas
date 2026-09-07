// Package horizon provides historical transaction lookups for Atlas backfill.
package horizon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(horizonURL string) *Client {
	return &Client{
		baseURL: horizonURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type TransactionPage struct {
	Records  []Transaction `json:"records"`
	NextHref string        `json:"-"`
}

type Transaction struct {
	Hash           string `json:"hash"`
	Ledger         int64  `json:"ledger"`
	CreatedAt      string `json:"created_at"`
	Successful     bool   `json:"successful"`
	OperationCount int    `json:"operation_count"`
}

type txPageEnvelope struct {
	Embedded struct {
		Records []Transaction `json:"records"`
	} `json:"_embedded"`
	Links struct {
		Next struct {
			Href string `json:"href"`
		} `json:"next"`
	} `json:"_links"`
}

func (c *Client) TransactionsByLedgerRange(
	ctx context.Context,
	fromLedger, toLedger int64,
	limit int,
) ([]Transaction, error) {
	if limit <= 0 {
		limit = 200
	}
	query := url.Values{}
	query.Set("order", "asc")
	query.Set("limit", strconv.Itoa(limit))
	query.Set("cursor", fmt.Sprintf("%d", fromLedger))

	endpoint := fmt.Sprintf("%s/transactions?%s", c.baseURL, query.Encode())
	out := make([]Transaction, 0, limit)

	for endpoint != "" {
		page, err := c.fetchPage(ctx, endpoint)
		if err != nil {
			return nil, err
		}
		for _, tx := range page.Records {
			if tx.Ledger > toLedger {
				return out, nil
			}
			if tx.Ledger >= fromLedger {
				out = append(out, tx)
			}
		}
		endpoint = page.NextHref
	}
	return out, nil
}

func (c *Client) fetchPage(ctx context.Context, endpoint string) (TransactionPage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return TransactionPage{}, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return TransactionPage{}, fmt.Errorf("horizon request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return TransactionPage{}, fmt.Errorf("horizon status %d", resp.StatusCode)
	}

	var envelope txPageEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return TransactionPage{}, fmt.Errorf("decode horizon page: %w", err)
	}
	return TransactionPage{
		Records:  envelope.Embedded.Records,
		NextHref: envelope.Links.Next.Href,
	}, nil
}
