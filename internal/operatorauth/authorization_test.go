package operatorauth

import "testing"

func TestViewerCannotWriteRegistrations(t *testing.T) {
	principal := Principal{ActorType: "operator", ActorID: "viewer-1", Role: RoleViewer}
	if (RBAC{}).Allowed(principal, PermissionRegistrationsWrite) {
		t.Fatal("viewer unexpectedly received registration write permission")
	}
	if !(RBAC{}).Allowed(principal, PermissionWorkloadsRead) {
		t.Fatal("viewer should retain workload read permission")
	}
}

func TestOperatorCanWriteRegistrations(t *testing.T) {
	principal := Principal{ActorType: "operator", ActorID: "operator-1", Role: RoleOperator}
	if !(RBAC{}).Allowed(principal, PermissionRegistrationsWrite) {
		t.Fatal("operator should receive registration write permission")
	}
}

func TestUnknownRoleFailsClosed(t *testing.T) {
	principal := Principal{ActorType: "operator", ActorID: "unknown", Role: Role("owner")}
	if (RBAC{}).Allowed(principal, PermissionWorkloadsRead) {
		t.Fatal("unknown role unexpectedly received permission")
	}
}

func TestParseRole(t *testing.T) {
	role, err := ParseRole(" OPERATOR ")
	if err != nil {
		t.Fatalf("ParseRole() error = %v", err)
	}
	if role != RoleOperator {
		t.Fatalf("role = %q, want %q", role, RoleOperator)
	}
	if _, err := ParseRole("owner"); err == nil {
		t.Fatal("expected unsupported role to fail")
	}
}
