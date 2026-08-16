package registration

import (
	"errors"
	"testing"
)

func TestValidateActorRequiresWriteOperator(t *testing.T) {
	cases := []MutationActor{
		{Type: "operator", ID: "local-viewer", Role: "viewer"},
		{Type: "operator", ID: "local-admin", Role: ""},
		{Type: "service", ID: "local-admin", Role: "operator"},
		{Type: "operator", ID: "", Role: "operator"},
	}
	for _, actor := range cases {
		if err := validateActor(actor); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("validateActor(%#v) error = %v, want ErrInvalidInput", actor, err)
		}
	}

	if err := validateActor(MutationActor{Type: "operator", ID: "local-admin", Role: "operator"}); err != nil {
		t.Fatalf("valid operator rejected: %v", err)
	}
}

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
		{name: "empty path segment", state: "present", selectors: []string{"docker:label:service:api"}, parent: "spiffe://workload-trust.test/spire//agent", ttl: 300},
		{name: "dot path segment", state: "present", selectors: []string{"docker:label:service:api"}, parent: "spiffe://workload-trust.test/spire/./agent", ttl: 300},
		{name: "dotdot path segment", state: "present", selectors: []string{"docker:label:service:api"}, parent: "spiffe://workload-trust.test/spire/../agent", ttl: 300},
		{name: "trailing slash", state: "present", selectors: []string{"docker:label:service:api"}, parent: "spiffe://workload-trust.test/spire/agent/", ttl: 300},
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
