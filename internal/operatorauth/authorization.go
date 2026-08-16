package operatorauth

import (
	"fmt"
	"strings"
)

type Role string

const (
	RoleViewer   Role = "viewer"
	RoleOperator Role = "operator"
)

type Permission string

const (
	PermissionWorkloadsRead      Permission = "workloads:read"
	PermissionRegistrationsRead  Permission = "registrations:read"
	PermissionRegistrationsWrite Permission = "registrations:write"
	PermissionPoliciesRead       Permission = "policies:read"
	PermissionPoliciesWrite      Permission = "policies:write"
	PermissionPoliciesActivate   Permission = "policies:activate"
	PermissionDiagnosticsRead    Permission = "diagnostics:read"
)

type Authorizer interface {
	Allowed(Principal, Permission) bool
}

type RBAC struct{}

func ParseRole(raw string) (Role, error) {
	role := Role(strings.ToLower(strings.TrimSpace(raw)))
	if !role.Valid() {
		return "", fmt.Errorf("unsupported operator role %q", raw)
	}
	return role, nil
}

func (r Role) Valid() bool {
	return r == RoleViewer || r == RoleOperator
}

func (RBAC) Allowed(principal Principal, permission Permission) bool {
	if principal.ActorType != "operator" || principal.ActorID == "" || !principal.Role.Valid() {
		return false
	}

	switch principal.Role {
	case RoleViewer:
		return permission == PermissionWorkloadsRead ||
			permission == PermissionRegistrationsRead ||
			permission == PermissionPoliciesRead ||
			permission == PermissionDiagnosticsRead
	case RoleOperator:
		return permission == PermissionWorkloadsRead ||
			permission == PermissionRegistrationsRead ||
			permission == PermissionRegistrationsWrite ||
			permission == PermissionPoliciesRead ||
			permission == PermissionPoliciesWrite ||
			permission == PermissionPoliciesActivate ||
			permission == PermissionDiagnosticsRead
	default:
		return false
	}
}
