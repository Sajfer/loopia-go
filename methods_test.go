package loopia

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// TestStatusMethods checks the request sent by each method that returns only
// a status, both on success and when the API reports a failure.
func TestStatusMethods(t *testing.T) {
	ctx := context.Background()
	record := Record{ID: 14096733, TTL: 300, Type: "A", Value: "1.1.1.1", Priority: 0}
	recordStruct := map[string]any{
		"record_id": int64(14096733), "ttl": int64(300), "type": "A", "rdata": "1.1.1.1", "priority": int64(0),
	}

	tests := []struct {
		name   string
		call   func(*Client) error
		method string
		params []any
	}{
		{
			name:   "AddDomain",
			call:   func(c *Client) error { return c.AddDomain(ctx, "example.com") },
			method: "addDomain",
			params: []any{"example.com"},
		},
		{
			name:   "OrderDomain",
			call:   func(c *Client) error { return c.OrderDomain(ctx, "example.com", true) },
			method: "orderDomain",
			params: []any{"example.com", true},
		},
		{
			name:   "TransferDomain",
			call:   func(c *Client) error { return c.TransferDomain(ctx, "example.com", "auth-123") },
			method: "transferDomain",
			params: []any{"example.com", "auth-123"},
		},
		{
			name:   "RemoveDomain",
			call:   func(c *Client) error { return c.RemoveDomain(ctx, "example.com", false) },
			method: "removeDomain",
			params: []any{"example.com", false},
		},
		{
			name: "UpdateDNSServers",
			call: func(c *Client) error {
				return c.UpdateDNSServers(ctx, "example.com", []string{"ns1.loopia.se", "ns2.loopia.se"})
			},
			method: "updateDNSServers",
			params: []any{"example.com", []any{"ns1.loopia.se", "ns2.loopia.se"}},
		},
		{
			name:   "AddSubdomain",
			call:   func(c *Client) error { return c.AddSubdomain(ctx, "example.com", "www") },
			method: "addSubdomain",
			params: []any{"example.com", "www"},
		},
		{
			name:   "RemoveSubdomain",
			call:   func(c *Client) error { return c.RemoveSubdomain(ctx, "example.com", "api") },
			method: "removeSubdomain",
			params: []any{"example.com", "api"},
		},
		{
			name:   "UpdateZoneRecord",
			call:   func(c *Client) error { return c.UpdateZoneRecord(ctx, "example.com", "api", record) },
			method: "updateZoneRecord",
			params: []any{"example.com", "api", recordStruct},
		},
		{
			name:   "RemoveZoneRecord",
			call:   func(c *Client) error { return c.RemoveZoneRecord(ctx, "example.com", "api", 14096733) },
			method: "removeZoneRecord",
			params: []any{"example.com", "api", int64(14096733)},
		},
		{
			name:   "PayInvoiceUsingCredits",
			call:   func(c *Client) error { return c.PayInvoiceUsingCredits(ctx, "123456") },
			method: "payInvoiceUsingCredits",
			params: []any{"123456"},
		},
		{
			name: "TransferCreditsByCurrency",
			call: func(c *Client) error {
				return c.TransferCreditsByCurrency(ctx, "C1", "C2", 99.5, "SEK")
			},
			method: "transferCreditsByCurrency",
			params: []any{"C1", "C2", 99.5, "SEK"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, f := newFake(t, "OK")
			if err := tt.call(c); err != nil {
				t.Fatalf("err = %v", err)
			}
			checkCall(t, f.only(), tt.method, tt.params...)
		})
		t.Run(tt.name+"/failure", func(t *testing.T) {
			c, _ := newFake(t, "BAD_INDATA")
			if err := tt.call(c); !errors.Is(err, ErrBadIndata) {
				t.Fatalf("err = %v, want ErrBadIndata", err)
			}
		})
	}
}

func TestUpdateDNSServersRequiresTwo(t *testing.T) {
	c, _ := newFake(t)
	if err := c.UpdateDNSServers(context.Background(), "example.com", []string{"ns1.loopia.se"}); err == nil {
		t.Fatal("expected error with a single name server")
	}
}

func TestDomainIsFree(t *testing.T) {
	tests := []struct {
		reply   string
		want    bool
		wantErr error
	}{
		{"OK", true, nil},
		{"DOMAIN_OCCUPIED", false, nil},
		{"BAD_INDATA", false, ErrBadIndata},
		{"RATE_LIMITED", false, ErrRateLimited},
	}
	for _, tt := range tests {
		t.Run(tt.reply, func(t *testing.T) {
			c, f := newFake(t, tt.reply)
			got, err := c.DomainIsFree(context.Background(), "example.com")
			if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil && err != nil) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("free = %v, want %v", got, tt.want)
			}
			checkCall(t, f.only(), "domainIsFree", "example.com")
		})
	}
}

func TestGetDomains(t *testing.T) {
	c, f := newFake(t, []any{
		map[string]any{
			"domain": "example.com", "paid": 1, "registered": 1,
			"renewal_status": "NORMAL", "expiration_date": "2027-01-31", "reference_no": 0,
		},
		map[string]any{
			"domain": "example.se", "paid": 0, "registered": false,
			"renewal_status": "NOT_HANDLED_BY_LOOPIA", "expiration_date": "UNKNOWN", "reference_no": 123456,
		},
	})
	got, err := c.GetDomains(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Domain{
		{Name: "example.com", Paid: true, Registered: true, RenewalStatus: "NORMAL", ExpirationDate: "2027-01-31", ReferenceNo: "0"},
		{Name: "example.se", Paid: false, Registered: false, RenewalStatus: "NOT_HANDLED_BY_LOOPIA", ExpirationDate: "UNKNOWN", ReferenceNo: "123456"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
	checkCall(t, f.only(), "getDomains")
}

func TestGetDomainsEmpty(t *testing.T) {
	c, _ := newFake(t, []any{})
	got, err := c.GetDomains(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %d domains, want 0", len(got))
	}
}

func TestGetDomainsForCustomer(t *testing.T) {
	c, f := newFake(t, []any{})
	if _, err := c.ForCustomer("C1").GetDomains(context.Background()); err != nil {
		t.Fatal(err)
	}
	checkCall(t, f.only(), "getDomains", "C1")
}

func TestGetDomain(t *testing.T) {
	c, f := newFake(t, map[string]any{
		"domain": "example.com", "paid": true, "registered": 1,
		"renewal_status": "NORMAL", "expiration_date": "2027-01-31", "reference_no": "98765",
	})
	got, err := c.GetDomain(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}
	want := &Domain{Name: "example.com", Paid: true, Registered: true, RenewalStatus: "NORMAL", ExpirationDate: "2027-01-31", ReferenceNo: "98765"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
	checkCall(t, f.only(), "getDomain", "example.com")
}

func TestGetSubdomains(t *testing.T) {
	c, f := newFake(t, fixture(t, "subdomains.xml"))
	got, err := c.GetSubdomains(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"*", "@", "www", "cdn", "registry", "media", "app", "join"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	checkCall(t, f.only(), "getSubdomains", "example.com")
}

var fixtureRecords = []Record{
	{ID: 14096733, TTL: 300, Type: "A", Value: "1.1.1.1", Priority: 0},
	{ID: 15838493, TTL: 300, Type: "CNAME", Value: "test.example.com.", Priority: 0},
}

func TestGetZoneRecords(t *testing.T) {
	c, f := newFake(t, fixture(t, "zone_records.xml"))
	got, err := c.GetZoneRecords(context.Background(), "example.com", "@")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, fixtureRecords) {
		t.Errorf("got  %+v\nwant %+v", got, fixtureRecords)
	}
	checkCall(t, f.only(), "getZoneRecords", "example.com", "@")
}

func TestGetZoneRecord(t *testing.T) {
	c, _ := newFake(t, fixture(t, "zone_records.xml"))
	got, err := c.GetZoneRecord(context.Background(), "example.com", "@", 15838493)
	if err != nil {
		t.Fatal(err)
	}
	if *got != fixtureRecords[1] {
		t.Errorf("got %+v, want %+v", *got, fixtureRecords[1])
	}
}

func TestGetZoneRecordNotFound(t *testing.T) {
	c, _ := newFake(t, fixture(t, "zone_records.xml"))
	_, err := c.GetZoneRecord(context.Background(), "example.com", "@", 1)
	if !errors.Is(err, ErrRecordNotFound) {
		t.Fatalf("err = %v, want ErrRecordNotFound", err)
	}
}

func TestAddZoneRecord(t *testing.T) {
	c, f := newFake(t, "OK", fixture(t, "zone_records.xml"))
	record := Record{ID: 999, TTL: 300, Type: "A", Value: "1.1.1.1"}
	if err := c.AddZoneRecord(context.Background(), "example.com", "api", &record); err != nil {
		t.Fatal(err)
	}
	if record.ID != 14096733 {
		t.Errorf("record.ID = %d, want 14096733", record.ID)
	}
	if len(f.calls) != 2 {
		t.Fatalf("got %d calls, want 2", len(f.calls))
	}
	// The caller's ID must not be sent when adding.
	checkCall(t, f.calls[0], "addZoneRecord", "example.com", "api", map[string]any{
		"record_id": int64(0), "ttl": int64(300), "type": "A", "rdata": "1.1.1.1", "priority": int64(0),
	})
	checkCall(t, f.calls[1], "getZoneRecords", "example.com", "api")
}

func TestAddZoneRecordPicksNewestDuplicate(t *testing.T) {
	rec := func(id int) map[string]any {
		return map[string]any{"record_id": id, "ttl": 3600, "type": "TXT", "rdata": "hello", "priority": 0}
	}
	c, _ := newFake(t, "OK", []any{rec(10), rec(30), rec(20)})
	record := Record{TTL: 3600, Type: "TXT", Value: "hello"}
	if err := c.AddZoneRecord(context.Background(), "example.com", "@", &record); err != nil {
		t.Fatal(err)
	}
	if record.ID != 30 {
		t.Errorf("record.ID = %d, want 30", record.ID)
	}
}

func TestAddZoneRecordIDUnknown(t *testing.T) {
	c, _ := newFake(t, "OK", fixture(t, "zone_records.xml"))
	record := Record{TTL: 300, Type: "A", Value: "9.9.9.9"}
	err := c.AddZoneRecord(context.Background(), "example.com", "api", &record)
	if !errors.Is(err, ErrRecordIDUnknown) {
		t.Fatalf("err = %v, want ErrRecordIDUnknown", err)
	}
	if record.ID != 0 {
		t.Errorf("record.ID = %d, want 0", record.ID)
	}
}

func TestAddZoneRecordLookupFails(t *testing.T) {
	c, _ := newFake(t, "OK", fault{Code: FaultRateLimited, Message: "Rate limit exceeded."})
	record := Record{TTL: 300, Type: "A", Value: "1.1.1.1"}
	err := c.AddZoneRecord(context.Background(), "example.com", "api", &record)
	if !errors.Is(err, ErrRecordIDUnknown) || !errors.Is(err, &FaultError{Code: FaultRateLimited}) {
		t.Fatalf("err = %v, want ErrRecordIDUnknown and rate limit fault", err)
	}
}

func TestAddZoneRecordFailure(t *testing.T) {
	c, f := newFake(t, "AUTH_ERROR")
	record := Record{TTL: 300, Type: "A", Value: "1.1.1.1"}
	if err := c.AddZoneRecord(context.Background(), "example.com", "api", &record); !errors.Is(err, ErrAuth) {
		t.Fatalf("err = %v, want ErrAuth", err)
	}
	f.only() // no lookup after a failed add
}

var invoiceReply = map[string]any{
	"reference_no": "123456",
	"total":        125.0,
	"credits":      25.5,
	"vat":          25,
	"to_pay":       99.5,
	"currency":     "SEK",
	"expires":      "2026-11-30",
	"items": []any{
		map[string]any{
			"product": "Domain example.se", "until": "2027-11-30",
			"fee": 100.0, "discount": 0, "items": 1, "subtotal": 100.0,
		},
	},
}

var wantInvoice = Invoice{
	ReferenceNo: "123456",
	Total:       125,
	Credits:     25.5,
	VAT:         25,
	ToPay:       99.5,
	Currency:    "SEK",
	Expires:     "2026-11-30",
	Items: []InvoiceItem{
		{Product: "Domain example.se", Until: "2027-11-30", Fee: 100, Discount: 0, Items: 1, Subtotal: 100},
	},
}

func TestGetInvoice(t *testing.T) {
	c, f := newFake(t, invoiceReply)
	got, err := c.GetInvoice(context.Background(), "123456", true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*got, wantInvoice) {
		t.Errorf("got  %+v\nwant %+v", *got, wantInvoice)
	}
	checkCall(t, f.only(), "getInvoice", "123456", true)
}

func TestGetUnpaidInvoices(t *testing.T) {
	c, f := newFake(t, []any{invoiceReply})
	got, err := c.ForCustomer("C1").GetUnpaidInvoices(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []Invoice{wantInvoice}) {
		t.Errorf("got  %+v\nwant %+v", got, []Invoice{wantInvoice})
	}
	checkCall(t, f.only(), "getUnpaidInvoices", "C1", false)
}

func TestGetCreditsAmount(t *testing.T) {
	tests := []struct {
		reply any
		want  float64
	}{
		{1234.5, 1234.5},
		{1234, 1234}, // whole amounts may be sent as <int>
	}
	for _, tt := range tests {
		c, f := newFake(t, tt.reply)
		got, err := c.GetCreditsAmount(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("reply %v: got %v, want %v", tt.reply, got, tt.want)
		}
		checkCall(t, f.only(), "getCreditsAmount")
	}
}

func TestCreateNewAccount(t *testing.T) {
	c, f := newFake(t, map[string]any{"status": "OK", "order_reference": "ORD-1"})
	ref, err := c.CreateNewAccount(context.Background(), NewAccount{
		Domain: "example.se",
		Owner: Contact{
			FirstName: "Anna", LastName: "Andersson", Street: "Kopparbergsvägen 8",
			Zip: "722 13", City: "Västerås", CountryISO2: "se", Phone: "021-128222", Email: "anna@example.se",
		},
		BillingContactReseller:  true,
		TechContactReseller:     false,
		BuyDomain:               true,
		DomainConfiguration:     DomainParking,
		AccountType:             AccountLoopiaDNS,
		EndUserHasAcceptedTerms: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ref != "ORD-1" {
		t.Errorf("ref = %q", ref)
	}
	checkCall(t, f.only(), "createNewAccount",
		"example.se",
		map[string]any{
			"firstname": "Anna", "lastname": "Andersson", "company": "", "street": "Kopparbergsvägen 8",
			"street2": "", "zip": "722 13", "city": "Västerås", "country_iso2": "se", "orgno": "",
			"phone": "021-128222", "cell": "", "fax": "", "email": "anna@example.se",
		},
		true, false, true, "PARKING", "LOOPIADNS", true,
	)
}

func TestCreateNewAccountNoridPID(t *testing.T) {
	c, f := newFake(t, map[string]any{"status": "OK", "order_reference": "ORD-2"})
	if _, err := c.CreateNewAccount(context.Background(), NewAccount{
		Domain: "example.no",
		Owner:  Contact{NoridPID: "N.PRI.123"},
	}); err != nil {
		t.Fatal(err)
	}
	owner := f.only().Params[3].(map[string]any)
	if owner["norid_pid"] != "N.PRI.123" {
		t.Errorf("norid_pid = %v", owner["norid_pid"])
	}
}

func TestCreateNewAccountStatusError(t *testing.T) {
	c, _ := newFake(t, map[string]any{"status": "BAD_INDATA", "order_reference": ""})
	if _, err := c.CreateNewAccount(context.Background(), NewAccount{Domain: "example.se"}); !errors.Is(err, ErrBadIndata) {
		t.Fatalf("err = %v, want ErrBadIndata", err)
	}
}

func TestCreateNewAccountIgnoresCustomerNumber(t *testing.T) {
	c, f := newFake(t, map[string]any{"status": "OK", "order_reference": "ORD-3"})
	if _, err := c.ForCustomer("C1").CreateNewAccount(context.Background(), NewAccount{Domain: "example.se"}); err != nil {
		t.Fatal(err)
	}
	if got := f.only().Params[2]; got != "example.se" {
		t.Errorf("third param = %v, want domain", got)
	}
}

func TestGetCustomers(t *testing.T) {
	c, f := newFake(t, []any{
		map[string]any{"company": "", "name": "Anna Andersson", "customer_number": "C1", "account_type": "LOOPIADNS"},
		map[string]any{"company": "Exempel AB", "name": "Bo Berg", "customer_number": "C2", "account_type": "HOSTING_BUSINESS"},
	})
	got, err := c.GetCustomers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Customer{
		{Name: "Anna Andersson", CustomerNumber: "C1", AccountType: AccountLoopiaDNS},
		{Company: "Exempel AB", Name: "Bo Berg", CustomerNumber: "C2", AccountType: AccountHostingBusiness},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
	checkCall(t, f.only(), "getCustomers")
}

func TestGetOrderStatus(t *testing.T) {
	c, f := newFake(t, map[string]any{"order_status": "PROCESSED", "customer_number": "C9"})
	got, err := c.GetOrderStatus(context.Background(), "ORD-1")
	if err != nil {
		t.Fatal(err)
	}
	if want := (Order{Status: OrderProcessed, CustomerNumber: "C9"}); *got != want {
		t.Errorf("got %+v, want %+v", *got, want)
	}
	checkCall(t, f.only(), "getOrderStatus", "ORD-1")
}
