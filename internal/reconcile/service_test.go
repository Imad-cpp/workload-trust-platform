package reconcile

import (
	"context"
	"testing"

	"github.com/Imad-cpp/workload-trust-platform/internal/audit"
	"github.com/Imad-cpp/workload-trust-platform/internal/registration"
	"github.com/Imad-cpp/workload-trust-platform/internal/spiremgmt"
)

type ruleStoreStub struct {
	rules       []registration.Rule
	convergedID *string
	errorCode   string
}

func (s *ruleStoreStub) List(context.Context) ([]registration.Rule, error) { return s.rules, nil }
func (s *ruleStoreStub) MarkConverged(_ context.Context, _ string, entryID *string) error {
	s.convergedID = entryID
	return nil
}
func (s *ruleStoreStub) MarkError(_ context.Context, _, code string) error {
	s.errorCode = code
	return nil
}

type spireStub struct {
	getEntry     spiremgmt.Entry
	getErr       error
	createResult spiremgmt.CreateResult
	createErr    error
	updated      spiremgmt.Entry
	updateErr    error
	deletedID    string
	deleteErr    error
	createCalls  int
	updateCalls  int
}

func (s *spireStub) GetEntry(context.Context, string) (spiremgmt.Entry, error) { return s.getEntry, s.getErr }
func (s *spireStub) CreateEntry(_ context.Context, _ spiremgmt.Entry) (spiremgmt.CreateResult, error) {
	s.createCalls++
	return s.createResult, s.createErr
}
func (s *spireStub) UpdateEntry(_ context.Context, _ spiremgmt.Entry) (spiremgmt.Entry, error) {
	s.updateCalls++
	return s.updated, s.updateErr
}
func (s *spireStub) DeleteEntry(_ context.Context, id string) error {
	s.deletedID = id
	return s.deleteErr
}

type auditStub struct{ events []audit.Event }

func (s *auditStub) Append(_ context.Context, event audit.Event) (audit.Event, error) {
	s.events = append(s.events, event)
	return event, nil
}

func baseRule() registration.Rule {
	parent := "spiffe://workload-trust.test/spire/agent/test"
	return registration.Rule{
		ID:                 "123e4567-e89b-12d3-a456-426614174001",
		OrganizationID:     "123e4567-e89b-12d3-a456-426614174000",
		WorkloadID:         "123e4567-e89b-12d3-a456-426614174002",
		WorkloadSPIFFEID:   "spiffe://workload-trust.test/lab/frontend",
		Selectors:          []string{"docker:label:service:frontend", "docker:label:environment:lab"},
		DesiredState:       "present",
		ParentSPIFFEID:     &parent,
		X509SVIDTTLSeconds: 300,
		ReconcileStatus:    "pending",
	}
}

func TestCreateAndBindOwnedEntry(t *testing.T) {
	rule := baseRule()
	entry := spiremgmt.Entry{ID: "entry-1", Hint: ownerHintPrefix + rule.ID}
	store := &ruleStoreStub{rules: []registration.Rule{rule}}
	spire := &spireStub{createResult: spiremgmt.CreateResult{Entry: entry, Disposition: spiremgmt.CreateDispositionCreated}}
	audits := &auditStub{}
	service, _ := New(store, spire, audits)

	summary, err := service.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if summary.Changed != 1 || store.convergedID == nil || *store.convergedID != "entry-1" {
		t.Fatalf("unexpected result: summary=%#v entry=%v", summary, store.convergedID)
	}
	if len(audits.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(audits.events))
	}
}

func TestExistingForeignEntryIsNeverAdopted(t *testing.T) {
	rule := baseRule()
	store := &ruleStoreStub{rules: []registration.Rule{rule}}
	spire := &spireStub{createResult: spiremgmt.CreateResult{
		Entry:       spiremgmt.Entry{ID: "foreign-entry", Hint: "other-controller"},
		Disposition: spiremgmt.CreateDispositionExisting,
	}}
	audits := &auditStub{}
	service, _ := New(store, spire, audits)

	summary, err := service.Run(context.Background())
	if err == nil {
		t.Fatal("expected ownership conflict")
	}
	if summary.Failed != 1 || store.errorCode != "ownership_mismatch" || store.convergedID != nil {
		t.Fatalf("unexpected conflict state: summary=%#v error=%q", summary, store.errorCode)
	}
	if spire.updateCalls != 0 || spire.deletedID != "" {
		t.Fatal("foreign entry was mutated")
	}
}

func TestAbsentOwnedEntryIsDeleted(t *testing.T) {
	rule := baseRule()
	rule.DesiredState = "absent"
	entryID := "entry-owned"
	rule.SPIREEntryID = &entryID
	store := &ruleStoreStub{rules: []registration.Rule{rule}}
	spire := &spireStub{getEntry: spiremgmt.Entry{ID: entryID, Hint: ownerHintPrefix + rule.ID}}
	service, _ := New(store, spire, &auditStub{})

	summary, err := service.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if summary.Changed != 1 || spire.deletedID != entryID || store.convergedID != nil {
		t.Fatalf("unexpected delete result: summary=%#v deleted=%q bound=%v", summary, spire.deletedID, store.convergedID)
	}
}

func TestOwnedDriftIsUpdated(t *testing.T) {
	rule := baseRule()
	entryID := "entry-owned"
	rule.SPIREEntryID = &entryID
	actual := spiremgmt.Entry{
		ID:             entryID,
		SPIFFEID:       rule.WorkloadSPIFFEID,
		ParentSPIFFEID: *rule.ParentSPIFFEID,
		Selectors:      []string{"docker:label:service:old"},
		X509SVIDTTL:    300,
		Hint:           ownerHintPrefix + rule.ID,
	}
	updated := actual
	updated.Selectors = append([]string(nil), rule.Selectors...)
	store := &ruleStoreStub{rules: []registration.Rule{rule}}
	spire := &spireStub{getEntry: actual, updated: updated}
	service, _ := New(store, spire, &auditStub{})

	summary, err := service.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if summary.Changed != 1 || spire.updateCalls != 1 {
		t.Fatalf("unexpected update result: summary=%#v calls=%d", summary, spire.updateCalls)
	}
}

func TestMissingParentFailsClosed(t *testing.T) {
	rule := baseRule()
	rule.ParentSPIFFEID = nil
	store := &ruleStoreStub{rules: []registration.Rule{rule}}
	service, _ := New(store, &spireStub{}, &auditStub{})

	if _, err := service.Run(context.Background()); err == nil {
		t.Fatal("expected invalid desired state error")
	}
	if store.errorCode != "invalid_desired_state" {
		t.Fatalf("error code = %q", store.errorCode)
	}
}
