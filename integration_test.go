//go:build integration

// Integration tests against the real LoopiaAPI. They are excluded from normal
// test runs and need a LoopiaAPI user and a domain in its account:
//
//	LOOPIA_USER=user@loopiaapi LOOPIA_PASS=... LOOPIA_DOMAIN=example.com \
//		go test -tags integration -run Integration -v ./...
//
// Only read methods and reversible changes are used: a temporary subdomain
// named loopia-go-test-<timestamp> with a TXT record is created and removed
// again. Give the API user access to no more than the methods called here.
// A run makes about 15 calls; LoopiaAPI allows 60 per minute.
package loopia

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"
)

func integrationClient(t *testing.T) (*Client, string) {
	t.Helper()
	user, pass, domain := os.Getenv("LOOPIA_USER"), os.Getenv("LOOPIA_PASS"), os.Getenv("LOOPIA_DOMAIN")
	if user == "" || pass == "" || domain == "" {
		t.Skip("set LOOPIA_USER, LOOPIA_PASS and LOOPIA_DOMAIN to run integration tests")
	}
	return New(user, pass), domain
}

func integrationContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	return ctx
}

func TestIntegrationRead(t *testing.T) {
	c, domain := integrationClient(t)
	ctx := integrationContext(t)

	t.Run("GetDomains", func(t *testing.T) {
		domains, err := c.GetDomains(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(domains, func(d Domain) bool { return d.Name == domain }) {
			t.Errorf("%s not among the account's domains: %+v", domain, domains)
		}
		t.Logf("%d domains", len(domains))
	})

	t.Run("GetDomain", func(t *testing.T) {
		d, err := c.GetDomain(ctx, domain)
		if err != nil {
			t.Fatal(err)
		}
		if d.Name != domain {
			t.Errorf("Name = %q, want %q", d.Name, domain)
		}
		t.Logf("%+v", *d)
	})

	t.Run("GetSubdomains", func(t *testing.T) {
		subs, err := c.GetSubdomains(ctx, domain)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("subdomains: %v", subs)
	})

	t.Run("GetZoneRecords", func(t *testing.T) {
		records, err := c.GetZoneRecords(ctx, domain, "@")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%d records on @", len(records))
	})

	t.Run("DomainIsFree", func(t *testing.T) {
		free, err := c.DomainIsFree(ctx, domain)
		if err != nil {
			t.Fatal(err)
		}
		if free {
			t.Errorf("%s reported as free although it is in the account", domain)
		}
	})

	t.Run("GetCreditsAmount", func(t *testing.T) {
		amount, err := c.GetCreditsAmount(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("credits: %v", amount)
	})

	t.Run("GetUnpaidInvoices", func(t *testing.T) {
		invoices, err := c.GetUnpaidInvoices(ctx, true)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%d unpaid invoices", len(invoices))
	})
}

func TestIntegrationBadCredentials(t *testing.T) {
	_, domain := integrationClient(t)
	c := New("loopia-go-invalid@loopiaapi", "invalid")

	_, err := c.GetSubdomains(integrationContext(t), domain)
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("err = %v (%T), want ErrAuth", err, err)
	}
	t.Logf("bad credentials: %v", err)
}

func TestIntegrationZoneRecordLifecycle(t *testing.T) {
	c, domain := integrationClient(t)
	ctx := integrationContext(t)
	sub := fmt.Sprintf("loopia-go-test-%d", time.Now().Unix())

	if err := c.AddSubdomain(ctx, domain, sub); err != nil {
		t.Fatalf("AddSubdomain: %v", err)
	}
	t.Cleanup(func() {
		// Use a fresh context so cleanup runs even if the test timed out.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := c.RemoveSubdomain(ctx, domain, sub); err != nil {
			t.Errorf("cleanup: RemoveSubdomain %s.%s: %v", sub, domain, err)
		}
	})

	subs, err := c.GetSubdomains(ctx, domain)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(subs, sub) {
		t.Fatalf("new subdomain %q not listed: %v", sub, subs)
	}

	record := Record{Type: "TXT", TTL: 300, Value: "loopia-go integration test"}
	if err := c.AddZoneRecord(ctx, domain, sub, &record); err != nil {
		t.Fatalf("AddZoneRecord: %v", err)
	}
	if record.ID == 0 {
		t.Fatal("AddZoneRecord did not set record.ID")
	}
	t.Logf("created record %d on %s.%s", record.ID, sub, domain)

	record.Value = "loopia-go integration test (updated)"
	record.TTL = 600
	if err := c.UpdateZoneRecord(ctx, domain, sub, record); err != nil {
		t.Fatalf("UpdateZoneRecord: %v", err)
	}
	got, err := c.GetZoneRecord(ctx, domain, sub, record.ID)
	if err != nil {
		t.Fatalf("GetZoneRecord: %v", err)
	}
	if *got != record {
		t.Errorf("after update got %+v, want %+v", *got, record)
	}

	if err := c.RemoveZoneRecord(ctx, domain, sub, record.ID); err != nil {
		t.Fatalf("RemoveZoneRecord: %v", err)
	}
	if _, err := c.GetZoneRecord(ctx, domain, sub, record.ID); !errors.Is(err, ErrRecordNotFound) {
		t.Errorf("after removal: err = %v, want ErrRecordNotFound", err)
	}
}
