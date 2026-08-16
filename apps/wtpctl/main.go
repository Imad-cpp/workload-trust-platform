package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"
)

const (
	defaultAPIURL = "http://127.0.0.1:8080"
	minTokenBytes = 32
)

type apiClient struct {
	baseURL *url.URL
	token   string
	http    *http.Client
}

type errorEnvelope struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "wtpctl:", err)
		os.Exit(1)
	}
}

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return usageError()
	}

	baseURL := strings.TrimSpace(getenv("WTP_API_URL"))
	if baseURL == "" {
		baseURL = defaultAPIURL
	}
	parsed, err := validateBaseURL(baseURL)
	if err != nil {
		return err
	}

	client := &apiClient{
		baseURL: parsed,
		token:   getenv("WTP_OPERATOR_TOKEN"),
		http:    &http.Client{Timeout: 5 * time.Second},
	}

	switch args[0] {
	case "health":
		if len(args) != 1 {
			return fmt.Errorf("health takes no arguments")
		}
		return client.printGET(context.Background(), "/healthz", false, stdout)
	case "ready":
		if len(args) != 1 {
			return fmt.Errorf("ready takes no arguments")
		}
		return client.printGET(context.Background(), "/readyz", false, stdout)
	case "workloads":
		organizationID, err := parseOrganizationFlag("workloads", args[1:], stderr)
		if err != nil {
			return err
		}
		if err := validateToken(client.token); err != nil {
			return err
		}
		query := url.Values{"organization_id": []string{organizationID}}
		return client.printGET(context.Background(), "/v1/workloads?"+query.Encode(), true, stdout)
	case "status":
		organizationID, err := parseOrganizationFlag("status", args[1:], stderr)
		if err != nil {
			return err
		}
		if err := validateToken(client.token); err != nil {
			return err
		}
		query := url.Values{"organization_id": []string{organizationID}}
		return client.printGET(context.Background(), "/v1/diagnostics?"+query.Encode(), true, stdout)
	default:
		return usageError()
	}
}

func parseOrganizationFlag(command string, args []string, stderr io.Writer) (string, error) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	organizationID := fs.String("organization-id", "", "organization UUID")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() != 0 {
		return "", fmt.Errorf("%s received unexpected positional arguments", command)
	}
	if strings.TrimSpace(*organizationID) == "" {
		return "", fmt.Errorf("%s requires --organization-id", command)
	}
	return *organizationID, nil
}

func validateBaseURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, errors.New("WTP_API_URL is invalid")
	}
	if parsed.Scheme != "http" || parsed.Host == "" {
		return nil, errors.New("WTP_API_URL must use http with a loopback host")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, errors.New("WTP_API_URL must not contain credentials, path, query, or fragment")
	}
	host := parsed.Hostname()
	if host == "" {
		return nil, errors.New("WTP_API_URL must include a loopback host")
	}
	if !strings.EqualFold(host, "localhost") {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return nil, errors.New("WTP_API_URL must target localhost or a loopback IP")
		}
	}
	parsed.Path = ""
	return parsed, nil
}

func validateToken(token string) error {
	if len(token) < minTokenBytes {
		return fmt.Errorf("WTP_OPERATOR_TOKEN must be at least %d bytes", minTokenBytes)
	}
	if strings.IndexFunc(token, unicode.IsSpace) >= 0 {
		return errors.New("WTP_OPERATOR_TOKEN must not contain whitespace")
	}
	return nil
}

func (c *apiClient) printGET(ctx context.Context, path string, authenticated bool, output io.Writer) error {
	requestURL := *c.baseURL
	requestURL.RawQuery = ""
	requestURL.Fragment = ""
	if index := strings.IndexByte(path, '?'); index >= 0 {
		requestURL.Path = path[:index]
		requestURL.RawQuery = path[index+1:]
	} else {
		requestURL.Path = path
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return errors.New("could not construct management request")
	}
	if authenticated {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return errors.New("management API request failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return errors.New("could not read management API response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var envelope errorEnvelope
		if json.Unmarshal(body, &envelope) == nil && envelope.Error.Code != "" {
			return fmt.Errorf("management API returned HTTP %d (%s)", resp.StatusCode, envelope.Error.Code)
		}
		return fmt.Errorf("management API returned HTTP %d", resp.StatusCode)
	}
	if !json.Valid(body) {
		return errors.New("management API returned invalid JSON")
	}
	if _, err := output.Write(body); err != nil {
		return errors.New("could not write command output")
	}
	if len(body) == 0 || body[len(body)-1] != '\n' {
		_, _ = io.WriteString(output, "\n")
	}
	return nil
}

func usageError() error {
	return errors.New("usage: wtpctl health | ready | workloads --organization-id <uuid> | status --organization-id <uuid>")
}
