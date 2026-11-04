package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"serveg/internal/http"
	"strings"
)

type ShareAccess uint8

const (
	AccessDenied ShareAccess = iota
	AccessReadOnly
	AccessReadWrite
)

// writing: share allows writing and client sent write password
// reading: share is public or client sent password (read or write)
func checkSharePasswords(share *SharedLocation, req *http.Request,
) ShareAccess {
	reqPass := getBasicAuthPassword(req)
	canWrite := share.WritePassword != "" &&
		(reqPass != "" && passwordMatches(reqPass, share.WritePassword))
	if canWrite {
		return AccessReadWrite
	}
	canRead := share.ReadPassword == "" ||
		(reqPass != "" && passwordMatches(reqPass, share.ReadPassword))
	if canRead {
		return AccessReadOnly
	}
	return AccessDenied
}

func getBasicAuthPassword(req *http.Request) string {
	header := req.Headers["authorization"]
	scheme, b64, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Basic") {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return ""
	}
	// try get password from username:password
	_, password, _ := strings.Cut(string(decoded), ":")
	return password
}

func passwordMatches(got, want string) bool {
	// hash first: ConstantTimeCompare returns early when lengths differ,
	// which would leak the password length through timing
	g := sha256.Sum256([]byte(got))
	w := sha256.Sum256([]byte(want))
	return subtle.ConstantTimeCompare(g[:], w[:]) == 1
}
