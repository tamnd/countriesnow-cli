package countriesnow_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/countriesnow-cli/countriesnow"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *countriesnow.Client {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	cfg := countriesnow.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return countriesnow.NewClient(cfg)
}

func TestCapitals_userAgent(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua == "" {
			t.Error("request carried no User-Agent header")
		}
		if !strings.Contains(ua, "countriesnow-cli") {
			t.Errorf("User-Agent %q does not contain countriesnow-cli", ua)
		}
		_, _ = w.Write([]byte(`{"error":false,"msg":"ok","data":[{"name":"Afghanistan","capital":"Kabul","iso2":"AF","iso3":"AFG"}]}`))
	})
	_, err := c.Capitals(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCapitals_parseNameAndCapital(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"error":false,"msg":"ok","data":[
			{"name":"France","capital":"Paris","iso2":"FR","iso3":"FRA"},
			{"name":"Japan","capital":"Tokyo","iso2":"JP","iso3":"JPN"}
		]}`))
	})
	items, err := c.Capitals(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Name != "France" || items[0].Capital != "Paris" {
		t.Errorf("items[0] = %+v", items[0])
	}
	if items[1].Name != "Japan" || items[1].Capital != "Tokyo" {
		t.Errorf("items[1] = %+v", items[1])
	}
}

func TestCurrencies_parseCurrency(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"error":false,"msg":"ok","data":[
			{"name":"United States","currency":"USD","iso2":"US","iso3":"USA"}
		]}`))
	})
	items, err := c.Currencies(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least 1 currency item")
	}
	if items[0].Currency != "USD" {
		t.Errorf("Currency = %q, want USD", items[0].Currency)
	}
}

func TestPopulation_latestYear(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"error":false,"msg":"ok","data":[
			{"country":"Nigeria","populationCounts":[
				{"year":2010,"value":152217341},
				{"year":2020,"value":206139587}
			]}
		]}`))
	})
	items, err := c.Population(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("expected population data")
	}
	if items[0].LatestYear != 2020 {
		t.Errorf("LatestYear = %d, want 2020", items[0].LatestYear)
	}
	if items[0].LatestValue != 206139587 {
		t.Errorf("LatestValue = %d, want 206139587", items[0].LatestValue)
	}
}

func TestCities_post(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		_, _ = w.Write([]byte(`{"error":false,"msg":"ok","data":["Tokyo","Osaka","Kyoto"]}`))
	})
	cities, err := c.Cities(context.Background(), "Japan")
	if err != nil {
		t.Fatal(err)
	}
	if len(cities) != 3 {
		t.Fatalf("got %d cities, want 3", len(cities))
	}
	if cities[0] != "Tokyo" {
		t.Errorf("cities[0] = %q, want Tokyo", cities[0])
	}
}

func TestCapitals_retry503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"error":false,"msg":"ok","data":[{"name":"Germany","capital":"Berlin","iso2":"DE","iso3":"DEU"}]}`))
	}))
	defer ts.Close()

	cfg := countriesnow.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := countriesnow.NewClient(cfg)

	items, err := c.Capitals(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 || items[0].Capital != "Berlin" {
		t.Errorf("unexpected result after retry: %+v", items)
	}
	if hits != 3 {
		t.Errorf("server hits = %d, want 3", hits)
	}
}
