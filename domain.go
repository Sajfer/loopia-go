package loopia

import (
	"context"
	"errors"
)

// Domain describes a domain name in an account (domain_obj).
type Domain struct {
	// Name is the domain name, ACE-coded if it is an IDN domain.
	Name string
	// Paid reports whether the domain is marked as paid.
	Paid bool
	// Registered reports whether the domain is registered and ready to use.
	Registered bool
	// RenewalStatus is the renewal status of the domain at Loopia.
	RenewalStatus string
	// ExpirationDate is the expiry date, or "UNKNOWN" if the domain is not
	// renewed by Loopia.
	ExpirationDate string
	// ReferenceNo is the reference number of the invoice that must be paid to
	// activate the domain when Paid is false. It can be passed to
	// [Client.PayInvoiceUsingCredits].
	ReferenceNo string
}

type domainObj struct {
	Domain         string `xmlrpc:"domain"`
	Paid           any    `xmlrpc:"paid"`
	Registered     any    `xmlrpc:"registered"`
	RenewalStatus  string `xmlrpc:"renewal_status"`
	ExpirationDate string `xmlrpc:"expiration_date"`
	ReferenceNo    any    `xmlrpc:"reference_no"`
}

func (d domainObj) toDomain() Domain {
	return Domain{
		Name:           d.Domain,
		Paid:           toBool(d.Paid),
		Registered:     toBool(d.Registered),
		RenewalStatus:  d.RenewalStatus,
		ExpirationDate: d.ExpirationDate,
		ReferenceNo:    toString(d.ReferenceNo),
	}
}

// GetDomains returns all domain names in the account.
func (c *Client) GetDomains(ctx context.Context) ([]Domain, error) {
	var raw []domainObj
	if err := c.call(ctx, "getDomains", true, &raw); err != nil {
		return nil, err
	}
	domains := make([]Domain, len(raw))
	for i, d := range raw {
		domains[i] = d.toDomain()
	}
	return domains, nil
}

// GetDomain returns information about a domain name in the account, including
// the billing information needed to pay for it.
func (c *Client) GetDomain(ctx context.Context, domain string) (*Domain, error) {
	var raw domainObj
	if err := c.call(ctx, "getDomain", true, &raw, domain); err != nil {
		return nil, err
	}
	d := raw.toDomain()
	return &d, nil
}

// DomainIsFree reports whether a domain name is available for registration.
// The domain must be ACE-coded if it is an IDN domain.
func (c *Client) DomainIsFree(ctx context.Context, domain string) (bool, error) {
	err := c.callStatus(ctx, "domainIsFree", false, domain)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, ErrDomainOccupied):
		return false, nil
	default:
		return false, err
	}
}

// AddDomain adds an existing domain name to the account.
func (c *Client) AddDomain(ctx context.Context, domain string) error {
	return c.callStatus(ctx, "addDomain", true, domain)
}

// OrderDomain adds a new domain name to the account and registers it.
// acceptedTerms must reflect whether the account owner has agreed to Loopia's
// terms and conditions for registration.
func (c *Client) OrderDomain(ctx context.Context, domain string, acceptedTerms bool) error {
	return c.callStatus(ctx, "orderDomain", true, domain, acceptedTerms)
}

// TransferDomain starts a transfer of a domain name to the account, using the
// auth code from the previous registrar. Some TLDs charge a transfer fee.
func (c *Client) TransferDomain(ctx context.Context, domain, authCode string) error {
	return c.callStatus(ctx, "transferDomain", true, domain, authCode)
}

// RemoveDomain removes a domain name from the account, after which Loopia no
// longer renews it. If deactivate is true the domain is deactivated now and
// removed on its due date; otherwise it is removed immediately.
func (c *Client) RemoveDomain(ctx context.Context, domain string, deactivate bool) error {
	return c.callStatus(ctx, "removeDomain", true, domain, deactivate)
}

// UpdateDNSServers changes the name servers for a domain name. At least two
// name servers must be given.
func (c *Client) UpdateDNSServers(ctx context.Context, domain string, nameservers []string) error {
	if len(nameservers) < 2 {
		return errors.New("loopia: UpdateDNSServers requires at least two name servers")
	}
	return c.callStatus(ctx, "updateDNSServers", true, domain, nameservers)
}
