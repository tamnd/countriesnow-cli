package countriesnow

import (
	"testing"
)

// These tests are offline: they exercise the URI driver's pure string functions.
// The client's HTTP behaviour is covered in countriesnow_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "countriesnow" {
		t.Errorf("Scheme = %q, want countriesnow", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "countriesnow" {
		t.Errorf("Identity.Binary = %q, want countriesnow", info.Identity.Binary)
	}
}

func TestClassify_country(t *testing.T) {
	typ, id, err := Domain{}.Classify("Japan")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if typ != "country" {
		t.Errorf("type = %q, want country", typ)
	}
	if id != "Japan" {
		t.Errorf("id = %q, want Japan", id)
	}
}

func TestClassify_empty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("expected error on empty input, got nil")
	}
}

func TestLocate_country(t *testing.T) {
	got, err := Domain{}.Locate("country", "Japan")
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if got == "" {
		t.Error("Locate returned empty URL")
	}
}

func TestLocate_city(t *testing.T) {
	got, err := Domain{}.Locate("city", "Tokyo")
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if got == "" {
		t.Error("Locate returned empty URL")
	}
}

func TestLocate_badType(t *testing.T) {
	_, err := Domain{}.Locate("page", "foo")
	if err == nil {
		t.Error("expected error for unknown type, got nil")
	}
}
