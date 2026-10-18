package loopia

import (
	"context"
	"errors"
)

// Record is a DNS record in the zone of a domain name (record_obj).
type Record struct {
	// ID identifies the record when updating or removing it.
	ID int64 `xmlrpc:"record_id"`
	// TTL is the record's time to live in seconds. LoopiaDNS defaults to 3600.
	TTL int `xmlrpc:"ttl"`
	// Type is the record type, for instance A or CNAME.
	Type string `xmlrpc:"type"`
	// Value is the record data (rdata), e.g. the IPv4 address of an A record.
	Value string `xmlrpc:"rdata"`
	// Priority is the priority of MX records; lower values are used first.
	Priority int `xmlrpc:"priority"`
}

// ErrRecordIDUnknown is returned by [Client.AddZoneRecord] when the record was
// created but its ID could not be determined.
var ErrRecordIDUnknown = errors.New("loopia: record created but its ID could not be determined")

// ErrRecordNotFound is returned by [Client.GetZoneRecord] when no record has
// the requested ID.
var ErrRecordNotFound = errors.New("loopia: zone record not found")

// GetZoneRecords returns all DNS records of a subdomain.
func (c *Client) GetZoneRecords(ctx context.Context, domain, subdomain string) ([]Record, error) {
	var records []Record
	if err := c.call(ctx, "getZoneRecords", true, &records, domain, subdomain); err != nil {
		return nil, err
	}
	return records, nil
}

// GetZoneRecord returns the DNS record of a subdomain with the given ID, or
// [ErrRecordNotFound]. LoopiaAPI has no method for fetching a single record,
// so this fetches all records of the subdomain.
func (c *Client) GetZoneRecord(ctx context.Context, domain, subdomain string, id int64) (*Record, error) {
	records, err := c.GetZoneRecords(ctx, domain, subdomain)
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		if r.ID == id {
			return &r, nil
		}
	}
	return nil, ErrRecordNotFound
}

// AddZoneRecord adds a DNS record to a subdomain. record.ID is ignored.
//
// LoopiaAPI does not return the ID of the new record, so AddZoneRecord looks
// it up afterwards and stores it in record.ID, picking the newest record with
// the same values. If no such record is found, it returns
// [ErrRecordIDUnknown]; the record has still been created.
func (c *Client) AddZoneRecord(ctx context.Context, domain, subdomain string, record *Record) error {
	add := *record
	add.ID = 0
	if err := c.callStatus(ctx, "addZoneRecord", true, domain, subdomain, add); err != nil {
		return err
	}

	records, err := c.GetZoneRecords(ctx, domain, subdomain)
	if err != nil {
		return errors.Join(ErrRecordIDUnknown, err)
	}
	var id int64
	for _, r := range records {
		rid := r.ID
		r.ID = 0
		if r == add {
			id = max(id, rid)
		}
	}
	if id == 0 {
		return ErrRecordIDUnknown
	}
	record.ID = id
	return nil
}

// UpdateZoneRecord updates the DNS record with ID record.ID.
func (c *Client) UpdateZoneRecord(ctx context.Context, domain, subdomain string, record Record) error {
	return c.callStatus(ctx, "updateZoneRecord", true, domain, subdomain, record)
}

// RemoveZoneRecord removes the DNS record with the given ID from a subdomain.
func (c *Client) RemoveZoneRecord(ctx context.Context, domain, subdomain string, id int64) error {
	return c.callStatus(ctx, "removeZoneRecord", true, domain, subdomain, id)
}
