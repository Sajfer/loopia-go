// Package loopia is a client for LoopiaAPI, the XML-RPC API for the Loopia
// Customer Zone and Loopia Reseller Zone.
//
// See https://support.loopia.com/wiki/loopiaapi/ for the upstream
// documentation. Every documented method is available on [Client]; methods
// that resellers can run against a customer's account honour the customer
// number set with [Client.ForCustomer].
//
// LoopiaAPI allows at most 60 calls per minute (15 of which may be domain
// searches). Exceeding the limit results in a [StatusError] wrapping
// [StatusRateLimited] or a [FaultError] with code [FaultRateLimited].
package loopia

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/kolo/xmlrpc"
)

// DefaultEndpoint is the URL of the LoopiaAPI XML-RPC service.
const DefaultEndpoint = "https://api.loopia.se/RPCSERV"

// maxResponseSize caps how much of a response body is read.
const maxResponseSize = 10 << 20

// Client talks to LoopiaAPI. It is safe for concurrent use.
type Client struct {
	username       string
	password       string
	endpoint       string
	httpClient     *http.Client
	customerNumber string
}

// Option configures a [Client].
type Option func(*Client)

// WithEndpoint overrides the XML-RPC endpoint, for example to use one of
// Loopia's regional APIs or a test server.
func WithEndpoint(url string) Option {
	return func(c *Client) { c.endpoint = url }
}

// WithHTTPClient sets the HTTP client used for requests. The default is
// [http.DefaultClient].
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// New returns a client authenticating with the given LoopiaAPI credentials.
// API users are created in the Loopia Customer Zone under
// Account Settings → LoopiaAPI.
func New(username, password string, opts ...Option) *Client {
	c := &Client{
		username:   username,
		password:   password,
		endpoint:   DefaultEndpoint,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ForCustomer returns a copy of the client that acts on the account with the
// given customer number. Only resellers can act on other accounts; an empty
// customer number means the API user's own account.
func (c *Client) ForCustomer(customerNumber string) *Client {
	cc := *c
	cc.customerNumber = customerNumber
	return &cc
}

// CustomerNumber returns the customer number set with [Client.ForCustomer].
func (c *Client) CustomerNumber() string {
	return c.customerNumber
}

// call invokes method with the credentials prepended to args and decodes the
// reply into reply. When withCustomer is set and the client has a customer
// number, it is sent after the credentials, as the methods that accept the
// optional customer_number parameter expect.
func (c *Client) call(ctx context.Context, method string, withCustomer bool, reply any, args ...any) error {
	params := []any{c.username, c.password}
	if withCustomer && c.customerNumber != "" {
		params = append(params, c.customerNumber)
	}
	params = append(params, args...)

	body, err := xmlrpc.EncodeMethodCall(method, params...)
	if err != nil {
		return fmt.Errorf("loopia: encode %s: %w", method, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("loopia: %s: %w", method, err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("loopia: %s: %w", method, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return fmt.Errorf("loopia: %s: read response: %w", method, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("loopia: %s: unexpected HTTP status %s", method, resp.Status)
	}

	return decodeResponse(method, xmlrpc.Response(data), reply)
}

func decodeResponse(method string, r xmlrpc.Response, reply any) error {
	if err := r.Err(); err != nil {
		var fault xmlrpc.FaultError
		if errors.As(err, &fault) {
			return &FaultError{Code: fault.Code, Message: fault.String}
		}
		return fmt.Errorf("loopia: %s: decode fault: %w", method, err)
	}

	// Methods returning data reply with a bare status string, or an array
	// holding one, when the call fails (e.g. ["AUTH_ERROR"]).
	var generic any
	if err := r.Unmarshal(&generic); err != nil {
		return fmt.Errorf("loopia: %s: decode response: %w", method, err)
	}
	if s, ok := errorStatus(generic); ok {
		if _, wantStatus := reply.(*string); !wantStatus {
			return &StatusError{Status: s}
		}
	}

	if err := r.Unmarshal(reply); err != nil {
		return fmt.Errorf("loopia: %s: decode response: %w", method, err)
	}
	return nil
}

// errorStatus reports whether v is a non-OK status, either as a string or as
// a single-element array holding one.
func errorStatus(v any) (Status, bool) {
	if a, ok := v.([]any); ok && len(a) == 1 {
		v = a[0]
	}
	s, ok := v.(string)
	if !ok || !Status(s).isError() {
		return "", false
	}
	return Status(s), true
}

// callStatus invokes a method that returns a status and converts any status
// other than OK into a *StatusError.
func (c *Client) callStatus(ctx context.Context, method string, withCustomer bool, args ...any) error {
	var status string
	if err := c.call(ctx, method, withCustomer, &status, args...); err != nil {
		return err
	}
	return statusErr(Status(status))
}

func statusErr(s Status) error {
	if s == StatusOK {
		return nil
	}
	return &StatusError{Status: s}
}
