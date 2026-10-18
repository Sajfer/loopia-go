package loopia

import "context"

// GetSubdomains returns the names of all subdomains of a domain, such as "@",
// "*" and "www".
func (c *Client) GetSubdomains(ctx context.Context, domain string) ([]string, error) {
	var subdomains []string
	if err := c.call(ctx, "getSubdomains", true, &subdomains, domain); err != nil {
		return nil, err
	}
	return subdomains, nil
}

// AddSubdomain connects a subdomain to a domain name.
func (c *Client) AddSubdomain(ctx context.Context, domain, subdomain string) error {
	return c.callStatus(ctx, "addSubdomain", true, domain, subdomain)
}

// RemoveSubdomain removes a subdomain from a domain name.
func (c *Client) RemoveSubdomain(ctx context.Context, domain, subdomain string) error {
	return c.callStatus(ctx, "removeSubdomain", true, domain, subdomain)
}
