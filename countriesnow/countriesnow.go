// Package countriesnow is the library behind the countriesnow command line:
// the HTTP client, request shaping, and the typed data models for the
// countriesnow.space API (country data: capitals, currencies, flags, population,
// cities — no key required).
//
// The Client is the spine every command shares. It sets a real User-Agent,
// paces requests so a busy session stays polite, and retries the transient
// failures (429 and 5xx) that any public API throws under load.
package countriesnow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Host is the API hostname this package talks to.
const Host = "countriesnow.space"

// Config holds all tuneable parameters for a Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns the production configuration for countriesnow.space.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://countriesnow.space/api/v0.1",
		UserAgent: "countriesnow-cli/0.1.0 (github.com/tamnd/countriesnow-cli)",
		Rate:      200 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// CountryInfo holds merged data from multiple endpoints about one country.
type CountryInfo struct {
	Name     string `json:"name"`
	Capital  string `json:"capital,omitempty"`
	Currency string `json:"currency,omitempty"`
	ISO2     string `json:"iso2,omitempty"`
	ISO3     string `json:"iso3,omitempty"`
	Flag     string `json:"flag,omitempty"`
}

// PopYear is one year's population count.
type PopYear struct {
	Year  int `json:"year"`
	Value int `json:"value"`
}

// CountryPop holds population data for one country.
type CountryPop struct {
	Country          string    `json:"country"`
	PopulationCounts []PopYear `json:"populationCounts"`
	LatestYear       int       `json:"latest_year,omitempty"`
	LatestValue      int       `json:"latest_value,omitempty"`
}

// Client talks to countriesnow.space over HTTP.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured from cfg.
func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

// raw API envelope.
type apiEnvelope struct {
	Error bool            `json:"error"`
	Msg   string          `json:"msg"`
	Data  json.RawMessage `json:"data"`
}

// Capitals returns a list of countries with their capitals.
func (c *Client) Capitals(ctx context.Context, limit int) ([]CountryInfo, error) {
	type item struct {
		Name    string `json:"name"`
		Capital string `json:"capital"`
		ISO2    string `json:"iso2"`
		ISO3    string `json:"iso3"`
	}
	body, err := c.get(ctx, c.cfg.BaseURL+"/countries/capital")
	if err != nil {
		return nil, err
	}
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("countriesnow: decode capitals: %w", err)
	}
	var items []item
	if err := json.Unmarshal(env.Data, &items); err != nil {
		return nil, fmt.Errorf("countriesnow: decode capitals data: %w", err)
	}
	out := make([]CountryInfo, 0, len(items))
	for i, it := range items {
		if limit > 0 && i >= limit {
			break
		}
		out = append(out, CountryInfo{Name: it.Name, Capital: it.Capital, ISO2: it.ISO2, ISO3: it.ISO3})
	}
	return out, nil
}

// Currencies returns a list of countries with their currencies.
func (c *Client) Currencies(ctx context.Context, limit int) ([]CountryInfo, error) {
	type item struct {
		Name     string `json:"name"`
		Currency string `json:"currency"`
		ISO2     string `json:"iso2"`
		ISO3     string `json:"iso3"`
	}
	body, err := c.get(ctx, c.cfg.BaseURL+"/countries/currency")
	if err != nil {
		return nil, err
	}
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("countriesnow: decode currencies: %w", err)
	}
	var items []item
	if err := json.Unmarshal(env.Data, &items); err != nil {
		return nil, fmt.Errorf("countriesnow: decode currencies data: %w", err)
	}
	out := make([]CountryInfo, 0, len(items))
	for i, it := range items {
		if limit > 0 && i >= limit {
			break
		}
		out = append(out, CountryInfo{Name: it.Name, Currency: it.Currency, ISO2: it.ISO2, ISO3: it.ISO3})
	}
	return out, nil
}

// Flags returns a list of countries with their flag image URLs.
func (c *Client) Flags(ctx context.Context, limit int) ([]CountryInfo, error) {
	type item struct {
		Name string `json:"name"`
		Flag string `json:"flag"`
	}
	body, err := c.get(ctx, c.cfg.BaseURL+"/countries/flag/images")
	if err != nil {
		return nil, err
	}
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("countriesnow: decode flags: %w", err)
	}
	var items []item
	if err := json.Unmarshal(env.Data, &items); err != nil {
		return nil, fmt.Errorf("countriesnow: decode flags data: %w", err)
	}
	out := make([]CountryInfo, 0, len(items))
	for i, it := range items {
		if limit > 0 && i >= limit {
			break
		}
		out = append(out, CountryInfo{Name: it.Name, Flag: it.Flag})
	}
	return out, nil
}

// Population returns a list of countries with their latest population figure.
func (c *Client) Population(ctx context.Context, limit int) ([]CountryPop, error) {
	type item struct {
		Country          string    `json:"country"`
		PopulationCounts []PopYear `json:"populationCounts"`
	}
	body, err := c.get(ctx, c.cfg.BaseURL+"/countries/population")
	if err != nil {
		return nil, err
	}
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("countriesnow: decode population: %w", err)
	}
	var items []item
	if err := json.Unmarshal(env.Data, &items); err != nil {
		return nil, fmt.Errorf("countriesnow: decode population data: %w", err)
	}
	out := make([]CountryPop, 0, len(items))
	for i, it := range items {
		if limit > 0 && i >= limit {
			break
		}
		cp := CountryPop{Country: it.Country, PopulationCounts: it.PopulationCounts}
		// pick the most recent year's value
		for _, py := range it.PopulationCounts {
			if py.Year > cp.LatestYear {
				cp.LatestYear = py.Year
				cp.LatestValue = py.Value
			}
		}
		out = append(out, cp)
	}
	return out, nil
}

// Cities returns the cities in a given country via a POST request.
func (c *Client) Cities(ctx context.Context, country string) ([]string, error) {
	body, err := c.post(ctx, c.cfg.BaseURL+"/countries/cities", map[string]string{"country": country})
	if err != nil {
		return nil, err
	}
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("countriesnow: decode cities: %w", err)
	}
	var cities []string
	if err := json.Unmarshal(env.Data, &cities); err != nil {
		return nil, fmt.Errorf("countriesnow: decode cities data: %w", err)
	}
	return cities, nil
}

func (c *Client) get(ctx context.Context, u string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, http.MethodGet, u, nil)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("countriesnow: get %s: %w", u, lastErr)
}

func (c *Client) post(ctx context.Context, u string, payload any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("countriesnow: marshal body: %w", err)
	}
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, http.MethodPost, u, b)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("countriesnow: post %s: %w", u, lastErr)
}

func (c *Client) do(ctx context.Context, method, u string, payload []byte) (body []byte, retry bool, err error) {
	c.pace()
	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// pace blocks until at least Rate has passed since the previous request.
func (c *Client) pace() {
	if c.cfg.Rate <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
