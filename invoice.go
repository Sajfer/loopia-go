package loopia

import "context"

// Invoice describes an invoice (invoice_obj).
type Invoice struct {
	// ReferenceNo is the reference and invoice number.
	ReferenceNo string
	// Total is the invoice total.
	Total float64
	// Credits is the amount deducted using LoopiaPrePAID credits.
	Credits float64
	// VAT is the VAT amount.
	VAT float64
	// ToPay is the amount to pay, including tax where applicable.
	ToPay float64
	// Currency is the invoice currency.
	Currency string
	// Expires is the due date.
	Expires string
	// Items are the invoice line items.
	Items []InvoiceItem
}

// InvoiceItem is a single invoice line item (invoice_item_obj).
type InvoiceItem struct {
	// Product is the product description.
	Product string
	// Until is the period covered by the item, if applicable.
	Until string
	// Fee is the unit price.
	Fee float64
	// Discount is the discount deduction.
	Discount float64
	// Items is the quantity.
	Items float64
	// Subtotal is the total amount for the item.
	Subtotal float64
}

type invoiceObj struct {
	ReferenceNo any              `xmlrpc:"reference_no"`
	Total       any              `xmlrpc:"total"`
	Credits     any              `xmlrpc:"credits"`
	VAT         any              `xmlrpc:"vat"`
	ToPay       any              `xmlrpc:"to_pay"`
	Currency    string           `xmlrpc:"currency"`
	Expires     string           `xmlrpc:"expires"`
	Items       []invoiceItemObj `xmlrpc:"items"`
}

type invoiceItemObj struct {
	Product  string `xmlrpc:"product"`
	Until    string `xmlrpc:"until"`
	Fee      any    `xmlrpc:"fee"`
	Discount any    `xmlrpc:"discount"`
	Items    any    `xmlrpc:"items"`
	Subtotal any    `xmlrpc:"subtotal"`
}

func (o invoiceObj) toInvoice() Invoice {
	inv := Invoice{
		ReferenceNo: toString(o.ReferenceNo),
		Total:       toFloat(o.Total),
		Credits:     toFloat(o.Credits),
		VAT:         toFloat(o.VAT),
		ToPay:       toFloat(o.ToPay),
		Currency:    o.Currency,
		Expires:     o.Expires,
	}
	for _, it := range o.Items {
		inv.Items = append(inv.Items, InvoiceItem{
			Product:  it.Product,
			Until:    it.Until,
			Fee:      toFloat(it.Fee),
			Discount: toFloat(it.Discount),
			Items:    toFloat(it.Items),
			Subtotal: toFloat(it.Subtotal),
		})
	}
	return inv
}

// GetInvoice returns the invoice with the given reference number. withVAT
// selects whether amounts include VAT.
func (c *Client) GetInvoice(ctx context.Context, referenceNo string, withVAT bool) (*Invoice, error) {
	var raw invoiceObj
	if err := c.call(ctx, "getInvoice", true, &raw, referenceNo, withVAT); err != nil {
		return nil, err
	}
	inv := raw.toInvoice()
	return &inv, nil
}

// GetUnpaidInvoices returns all unpaid invoices of the account. withVAT
// selects whether amounts include VAT.
func (c *Client) GetUnpaidInvoices(ctx context.Context, withVAT bool) ([]Invoice, error) {
	var raw []invoiceObj
	if err := c.call(ctx, "getUnpaidInvoices", true, &raw, withVAT); err != nil {
		return nil, err
	}
	invoices := make([]Invoice, len(raw))
	for i, o := range raw {
		invoices[i] = o.toInvoice()
	}
	return invoices, nil
}

// PayInvoiceUsingCredits pays an invoice using LoopiaPrePAID credits,
// provided the account has enough credits.
func (c *Client) PayInvoiceUsingCredits(ctx context.Context, referenceNo string) error {
	return c.callStatus(ctx, "payInvoiceUsingCredits", true, referenceNo)
}

// GetCreditsAmount returns the current LoopiaPrePAID balance, in the
// account's currency, for amounts deposited after 2015-12-04.
func (c *Client) GetCreditsAmount(ctx context.Context) (float64, error) {
	var amount any
	if err := c.call(ctx, "getCreditsAmount", true, &amount); err != nil {
		return 0, err
	}
	return toFloat(amount), nil
}
