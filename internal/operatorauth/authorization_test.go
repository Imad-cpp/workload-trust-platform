package operatorauth

import "testing"

func TestViewerCannotWriteRegistrationsOrPolicies(t *testing.T) {
	principal := Principal{ActorType: "operator", ActorID: "viewer-1", Role: RoleViewer}
	for _, permission := range []Permission{
		PermissionRegistrationsWrite,
		PermissionPoliciesWrite,
		PermissionPoliciesActivate,
	} {
		if (RBAC{}).Allowed(principal, permission) {
			t.Fatalf("viewer unexpectedly received %q", permission)
		}
	}
	for _, permission := range []Permission{
		PermissionWorkloadsRead,
		PermissionRegistrationsRead,
		PermissionPoliciesRead,
		PermissionDiagnosticsRead,
	} {
		if !(RBAC{}).Allowed(principal, permission) {
			t.Fatalf("viewer should retain %q", permission)
		}
	}
}

func TestOperatorCanManageRegistrationsAndPolicies(t *testing.T) {
	principal := Principal{ActorType: "operator", ActorID: "operator-1", Role: RoleOperator}
	for _, permission := range []Permission{
		PermissionRegistrationsWrite,
		PermissionPoliciesWrite,
		PermissionPoliciesActivate,
		PermissionDiagnosticsRead,
	} {
		if !(RBAC{}).Allowed(principal, permission) {
			t.Fatalf("operator should receive %q", permission)
		}
	}
}

func TestUnknownRoleFailsClosed(t *testing.T) {
	principal := Principal{ActorType: "operator", ActorID: "unknown", Role: Role("owner")}
	if (RBAC{}).Allowed(principal, PermissionWorkloadsRead) ||
		(RBAC{}).Allowed(principal, PermissionPoliciesRead) ||
		(RBAC{}).Allowed(principal, PermissionDiagnosticsRead) ||
		(RBAC{}).Allowed(principal, PermissionPoliciesActivate) {
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
