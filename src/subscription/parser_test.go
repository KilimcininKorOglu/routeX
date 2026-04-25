package subscription

import (
	"strings"
	"testing"
)

func TestParseListPlainText(t *testing.T) {
	input := `# Comment line
example.com
test.org
duplicate.com
duplicate.com

another.net
`
	domains, err := ParseList(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"another.net", "duplicate.com", "example.com", "test.org"}
	if len(domains) != len(expected) {
		t.Fatalf("expected %d domains, got %d: %v", len(expected), len(domains), domains)
	}
	for i, d := range domains {
		if d != expected[i] {
			t.Errorf("domain[%d] = %q, want %q", i, d, expected[i])
		}
	}
}

func TestParseListHostsFormat(t *testing.T) {
	input := `127.0.0.1 localhost
0.0.0.0 ads.example.com
127.0.0.1 tracker.example.org
0.0.0.0 localhost.localdomain
`
	domains, err := ParseList(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"ads.example.com", "tracker.example.org"}
	if len(domains) != len(expected) {
		t.Fatalf("expected %d domains, got %d: %v", len(expected), len(domains), domains)
	}
	for i, d := range domains {
		if d != expected[i] {
			t.Errorf("domain[%d] = %q, want %q", i, d, expected[i])
		}
	}
}

func TestParseListAdGuardFormat(t *testing.T) {
	input := `! AdGuard comment
||ads.example.com^
||tracker.example.org^
||analytics.test.net^
`
	domains, err := ParseList(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"ads.example.com", "analytics.test.net", "tracker.example.org"}
	if len(domains) != len(expected) {
		t.Fatalf("expected %d domains, got %d: %v", len(expected), len(domains), domains)
	}
	for i, d := range domains {
		if d != expected[i] {
			t.Errorf("domain[%d] = %q, want %q", i, d, expected[i])
		}
	}
}

func TestParseListMixedFormats(t *testing.T) {
	input := `# Mixed format list
example.com
0.0.0.0 ads.example.com
||tracker.example.org^
! Another comment
plain.domain.net
127.0.0.1 hosts.example.com
`
	domains, err := ParseList(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"ads.example.com", "example.com", "hosts.example.com", "plain.domain.net", "tracker.example.org"}
	if len(domains) != len(expected) {
		t.Fatalf("expected %d domains, got %d: %v", len(expected), len(domains), domains)
	}
	for i, d := range domains {
		if d != expected[i] {
			t.Errorf("domain[%d] = %q, want %q", i, d, expected[i])
		}
	}
}

func TestParseListEmptyInput(t *testing.T) {
	domains, err := ParseList(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(domains) != 0 {
		t.Fatalf("expected 0 domains, got %d", len(domains))
	}
}

func TestParseListAllComments(t *testing.T) {
	input := `# Only comments
! Another comment
# Yet another
`
	domains, err := ParseList(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(domains) != 0 {
		t.Fatalf("expected 0 domains, got %d", len(domains))
	}
}

func TestParseListInvalidLines(t *testing.T) {
	input := `valid.example.com
not a domain
/path/to/file
user@email.com
http://url.example.com
single
`
	domains, err := ParseList(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"valid.example.com"}
	if len(domains) != len(expected) {
		t.Fatalf("expected %d domains, got %d: %v", len(expected), len(domains), domains)
	}
}

func TestParseListCaseNormalization(t *testing.T) {
	input := `Example.COM
EXAMPLE.com
example.com
`
	domains, err := ParseList(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("expected 1 domain after dedup, got %d: %v", len(domains), domains)
	}
	if domains[0] != "example.com" {
		t.Errorf("expected lowercase, got %q", domains[0])
	}
}
