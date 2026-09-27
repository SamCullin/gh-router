// Package gitcredential reads the request that git sends to a credential
// helper, so the router can choose an account before the native helper runs.
package gitcredential

import (
	"bufio"
	"bytes"
	"strings"
)

// Request holds the fields of a git credential request that affect routing.
// Raw keeps the original bytes so the native helper receives them unchanged.
type Request struct {
	Protocol string
	Host     string
	Path     string
	Raw      []byte
}

// Parse reads key=value lines up to the first blank line, as described in
// git-credential(1). Unknown keys are ignored.
func Parse(input []byte) Request {
	request := Request{Raw: input}
	scanner := bufio.NewScanner(bytes.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch key {
		case "protocol":
			request.Protocol = value
		case "host":
			request.Host = value
		case "path":
			request.Path = value
		}
	}
	return request
}

// IsGitHub reports whether the request targets github.com. A request without
// a host is treated as github.com because the native helper only serves the
// hosts it is configured for.
func (request Request) IsGitHub() bool {
	host := strings.ToLower(strings.TrimSpace(request.Host))
	return host == "" || host == "github.com"
}

// Repository returns OWNER/REPO from the request path when git sends one
// (credential.useHttpPath), or an empty string.
func (request Request) Repository() string {
	parts := strings.Split(strings.Trim(request.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	return parts[0] + "/" + strings.TrimSuffix(parts[1], ".git")
}
