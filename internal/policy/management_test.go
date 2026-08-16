package policy

import (
	"errors"
	"testing"
)

func TestValidateActorRequiresOperatorRole(t *testing.T) {
	if err := validateActor(MutationActor{Type: "operator", ID: "viewer-1", Role: "viewer"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("viewer error = %v, want ErrForbidden", err)
	}
	if err := validateActor(MutationActor{Type: "service", ID: "svc-1", Role: "operator"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("service actor error = %v, want ErrInvalidInput", err)
	}
	if err := validateActor(MutationActor{Type: "operator", ID: "operator-1", Role: "operator"}); err != nil {
		t.Fatalf("valid operator rejected: %v", err)
	}
}

func TestNormalizeRuleInputCanonicalizesActionAndEffect(t *testing.T) {
	input := normalizeRuleInput(RuleInput{Action: " CONNECT ", Effect: " ALLOW ", ChangeReason: " initial allow "})
	if input.Action != ActionConnect || input.Effect != EffectAllow || input.ChangeReason != "initial allow" {
		t.Fatalf("unexpected normalized input: %#v", input)
	}
}

func TestValidateRuleInputRejectsUndefinedAction(t *testing.T) {
	input := RuleInput{
		SourceSPIFFEID:      "spiffe://workload-trust.test/prod/frontend",
		DestinationSPIFFEID: "spiffe://workload-trust.test/prod/orders-api",
		Action:              "invoke",
		Effect:              EffectAllow,
		ChangeReason:        "action semantics are not defined",
	}
	if err := validateRuleInput(input); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestValidateRuleInputRejectsNonCanonicalSPIFFEID(t *testing.T) {
	input := RuleInput{
		SourceSPIFFEID:      "spiffe://workload-trust.test/prod//frontend",
		DestinationSPIFFEID: "spiffe://workload-trust.test/prod/orders-api",
		Action:              ActionConnect,
		Effect:              EffectAllow,
		ChangeReason:        "invalid source identity",
	}
	if err := validateRuleInput(input); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestValidateRuleInputAcceptsV1ConnectPolicy(t *testing.T) {
	input := RuleInput{
		SourceSPIFFEID:      "spiffe://workload-trust.test/prod/frontend",
		DestinationSPIFFEID: "spiffe://workload-trust.test/prod/orders-api",
		Action:              ActionConnect,
		Effect:              EffectDeny,
		ChangeReason:        "default deny exception review",
	}
	if err := validateRuleInput(input); err != nil {
		t.Fatalf("valid V1 policy rejected: %v", err)
	}
}
