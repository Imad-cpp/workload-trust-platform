package operatorauth

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
)

const MinTokenBytes = 32

type Principal struct {
	ActorType string
	ActorID   string
	Role      Role
}

type Authenticator interface {
	Authenticate(*http.Request) (Principal, bool)
}

type StaticBearer struct {
	tokenDigest [sha256.Size]byte
	principal   Principal
}

func NewStaticBearer(token, actorID string, role Role) (*StaticBearer, error) {
	if len(token) < MinTokenBytes {
		return nil, errors.New("operator token must be at least 32 bytes")
	}
	if strings.TrimSpace(actorID) == "" {
		return nil, errors.New("operator actor ID is required")
	}
	if !role.Valid() {
		return nil, errors.New("operator role is invalid")
	}

	return &StaticBearer{
		tokenDigest: sha256.Sum256([]byte(token)),
		principal: Principal{
			ActorType: "operator",
			ActorID:   actorID,
			Role:      role,
		},
	}, nil
}

func (a *StaticBearer) Authenticate(r *http.Request) (Principal, bool) {
	header := r.Header.Get("Authorization")
	scheme, credential, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || credential == "" || strings.ContainsAny(credential, " \t\r\n") {
		return Principal{}, false
	}

	candidate := sha256.Sum256([]byte(credential))
	if subtle.ConstantTimeCompare(candidate[:], a.tokenDigest[:]) != 1 {
		return Principal{}, false
	}
	return a.principal, true
}
