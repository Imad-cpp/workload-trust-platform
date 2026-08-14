package registration

import (
	"errors"
	"testing"
)

func TestValidateDesiredFieldsRejectsUnsafeInputs(t *testing.T) {
	validParent := "spiffe://workload-trust.test/spire/agent/test"
	cases := []struct {
		name      string
		state     string
		selectors []string
		parent    string
		ttl       int32
	}{
		{name: "unknown state", state: "deleted", selectors: []string{"docker:label:service:api"}, parent: validParent, ttl: 300},
		{name: "no selectors", state: "present", selectors: nil, parent: validParent, ttl: 300},
		{name: "duplicate selector", state: "present", selectors: []string{"docker:label:service:api", "docker:label:service:api"}, parent: validParent, ttl: 300},
		{name: "control character", state: "present", selectors: []string{"docker:label:service:api\nadmin"}, parent: validParent, ttl: 300},
		{name: "non spiffe parent", state: "present", selectors: []string{"docker:label:service:api"}, parent: "https://example.test/agent", ttl: 300},
		{name: "uppercase trust domain", state: "present", selectors: []string{"docker:label:service:api"}, parent: "spiffe://EXAMPLE.test/agent", ttl: 300},
		{name: "ttl too short", state: "present", selectors: []string{"docker:label:service:api"}, parent: validParent, ttl: 59},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateDesiredFields(tc.state, tc.selectors, tc.parent, tc.ttl); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestValidateDesiredFieldsAcceptsColonRichSelectorValue(t *testing.T) {
	err := validateDesiredFields(
		"present",
		[]string{"docker:label:com.workload-trust.environment:lab"},
		"spiffe://workload-trust.test/spire/agent/test",
		300,
	)
	if err != nil {
		t.Fatalf("validateDesiredFields() error = %v", err)
	}
}
