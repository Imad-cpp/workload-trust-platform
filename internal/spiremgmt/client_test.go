package spiremgmt

import "testing"

func TestParseSPIFFEID(t *testing.T) {
	value, err := parseSPIFFEID("spiffe://workload-trust.test/lab/frontend")
	if err != nil {
		t.Fatalf("parseSPIFFEID() error = %v", err)
	}
	if value.TrustDomain != "workload-trust.test" || value.Path != "/lab/frontend" {
		t.Fatalf("unexpected SPIFFE ID: %#v", value)
	}
}

func TestParseSPIFFEIDRejectsNonCanonicalInput(t *testing.T) {
	for _, raw := range []string{
		"https://workload-trust.test/lab/frontend",
		"spiffe://WORKLOAD-TRUST.TEST/lab/frontend",
		"spiffe://workload-trust.test:443/lab/frontend",
		"spiffe://workload-trust.test/lab/frontend?debug=1",
	} {
		if _, err := parseSPIFFEID(raw); err == nil {
			t.Fatalf("%q unexpectedly accepted", raw)
		}
	}
}

func TestParseSelectorPreservesValueColons(t *testing.T) {
	selector, err := parseSelector("docker:label:service:frontend")
	if err != nil {
		t.Fatalf("parseSelector() error = %v", err)
	}
	if selector.Type != "docker" || selector.Value != "label:service:frontend" {
		t.Fatalf("unexpected selector: %#v", selector)
	}
}
