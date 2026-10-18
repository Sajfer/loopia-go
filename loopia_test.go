package loopia

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewDefaults(t *testing.T) {
	c := New("u", "p")
	if c.endpoint != DefaultEndpoint {
		t.Errorf("endpoint = %q, want %q", c.endpoint, DefaultEndpoint)
	}
	if c.httpClient != http.DefaultClient {
		t.Error("httpClient is not http.DefaultClient")
	}
	if c.CustomerNumber() != "" {
		t.Errorf("CustomerNumber() = %q, want empty", c.CustomerNumber())
	}
}

func TestWithHTTPClient(t *testing.T) {
	var used bool
	hc := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		used = true
		return http.DefaultTransport.RoundTrip(r)
	})}
	f := &fakeLoopia{t: t, replies: []any{"OK"}}
	srv := httptest.NewServer(f)
	defer srv.Close()

	c := New(testUser, testPass, WithEndpoint(srv.URL), WithHTTPClient(hc))
	if err := c.AddSubdomain(context.Background(), "example.com", "www"); err != nil {
		t.Fatal(err)
	}
	if !used {
		t.Error("custom HTTP client was not used")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestForCustomer(t *testing.T) {
	c, f := newFake(t, "OK", "OK")
	rc := c.ForCustomer("C12345")

	if rc.CustomerNumber() != "C12345" {
		t.Errorf("CustomerNumber() = %q", rc.CustomerNumber())
	}
	if c.CustomerNumber() != "" {
		t.Error("ForCustomer modified the original client")
	}

	ctx := context.Background()
	if err := rc.AddSubdomain(ctx, "example.com", "www"); err != nil {
		t.Fatal(err)
	}
	if err := c.AddSubdomain(ctx, "example.com", "www"); err != nil {
		t.Fatal(err)
	}
	checkCall(t, f.calls[0], "addSubdomain", "C12345", "example.com", "www")
	checkCall(t, f.calls[1], "addSubdomain", "example.com", "www")
}

func TestForCustomerNotSentToMethodsWithoutCustomerNumber(t *testing.T) {
	c, f := newFake(t, "OK")
	if _, err := c.ForCustomer("C1").DomainIsFree(context.Background(), "example.com"); err != nil {
		t.Fatal(err)
	}
	checkCall(t, f.only(), "domainIsFree", "example.com")
}

func TestStatusErrors(t *testing.T) {
	tests := []struct {
		status Status
		want   error
	}{
		{StatusAuthError, ErrAuth},
		{StatusDomainOccupied, ErrDomainOccupied},
		{StatusRateLimited, ErrRateLimited},
		{StatusBadIndata, ErrBadIndata},
		{StatusUnknownError, ErrUnknown},
		{StatusInsufficientFunds, ErrInsufficientFunds},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			c, _ := newFake(t, string(tt.status))
			err := c.AddSubdomain(context.Background(), "example.com", "www")
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			var se *StatusError
			if !errors.As(err, &se) || se.Status != tt.status {
				t.Errorf("errors.As StatusError = %v", se)
			}
			if errors.Is(err, &StatusError{Status: "OTHER"}) {
				t.Error("errors.Is matched a different status")
			}
		})
	}
}

func TestStatusErrorFromFixture(t *testing.T) {
	c, _ := newFake(t, fixture(t, "auth_error.xml"))
	err := c.AddSubdomain(context.Background(), "example.com", "test")
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("err = %v, want ErrAuth", err)
	}
}

func TestUnknownStatusIsError(t *testing.T) {
	c, _ := newFake(t, "SOMETHING_NEW")
	err := c.AddSubdomain(context.Background(), "example.com", "www")
	var se *StatusError
	if !errors.As(err, &se) || se.Status != "SOMETHING_NEW" {
		t.Fatalf("err = %v, want StatusError SOMETHING_NEW", err)
	}
}

// Methods returning data reply with a status instead when they fail.
func TestStatusInsteadOfData(t *testing.T) {
	tests := []struct {
		name  string
		reply any
	}{
		{"bare string", "AUTH_ERROR"},
		{"array", []any{"AUTH_ERROR"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newFake(t, tt.reply, tt.reply, tt.reply, tt.reply)
			ctx := context.Background()
			if _, err := c.GetSubdomains(ctx, "example.com"); !errors.Is(err, ErrAuth) {
				t.Errorf("GetSubdomains err = %v, want ErrAuth", err)
			}
			if _, err := c.GetZoneRecords(ctx, "example.com", "@"); !errors.Is(err, ErrAuth) {
				t.Errorf("GetZoneRecords err = %v, want ErrAuth", err)
			}
			if _, err := c.GetDomain(ctx, "example.com"); !errors.Is(err, ErrAuth) {
				t.Errorf("GetDomain err = %v, want ErrAuth", err)
			}
			if _, err := c.GetCreditsAmount(ctx); !errors.Is(err, ErrAuth) {
				t.Errorf("GetCreditsAmount err = %v, want ErrAuth", err)
			}
		})
	}
}

func TestFaultError(t *testing.T) {
	c, _ := newFake(t, fault{Code: FaultRateLimited, Message: "Rate limit exceeded."})
	_, err := c.GetDomains(context.Background())

	var fe *FaultError
	if !errors.As(err, &fe) {
		t.Fatalf("err = %v (%T), want *FaultError", err, err)
	}
	if fe.Code != FaultRateLimited || fe.Message != "Rate limit exceeded." {
		t.Errorf("fault = %+v", fe)
	}
	if !errors.Is(err, &FaultError{Code: FaultRateLimited}) {
		t.Error("errors.Is did not match fault code")
	}
	if errors.Is(err, &FaultError{Code: FaultUnauthorized}) {
		t.Error("errors.Is matched a different fault code")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("Error() = %q, want fault code", err.Error())
	}
}

func TestFaultFromDocumentedResponse(t *testing.T) {
	// The fault response example from the LoopiaAPI documentation.
	c, _ := newFake(t, rawXML(`<?xml version="1.0" encoding="UTF-8"?>
<methodResponse>
<fault>
<value>
<struct>
<member>
<name>faultCode</name>
<value>
<int>40001</int>
</value>
</member>
<member>
<name>faultString</name>
<value>
<string>The domain is not available.</string>
</value>
</member>
</struct>
</value>
</fault>
</methodResponse>`))
	err := c.OrderDomain(context.Background(), "example.com", true)
	if !errors.Is(err, &FaultError{Code: FaultDomainNotAvailable}) {
		t.Fatalf("err = %v, want fault 40001", err)
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()

	err := New("u", "p", WithEndpoint(srv.URL)).AddSubdomain(context.Background(), "example.com", "www")
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("err = %v, want HTTP 502 error", err)
	}
}

func TestMalformedResponse(t *testing.T) {
	c, _ := newFake(t, rawXML("this is not xml"))
	if _, err := c.GetDomains(context.Background()); err == nil {
		t.Fatal("expected error for malformed response")
	}
}

func TestContextCancellation(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-block:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(block)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := New("u", "p", WithEndpoint(srv.URL)).AddSubdomain(ctx, "example.com", "www")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestSpecialCharactersAreEscaped(t *testing.T) {
	c, f := newFake(t, "OK")
	c.password = `p<a&s>s"`
	if err := c.AddSubdomain(context.Background(), "example.com", "www"); err != nil {
		t.Fatal(err)
	}
	if got := f.only().Params[1]; got != `p<a&s>s"` {
		t.Errorf("password = %q", got)
	}
}
