package loopia

import "context"

// AccountType identifies the type of a Loopia account.
type AccountType string

// Account types.
const (
	AccountLoopiaDomain        AccountType = "LOOPIADOMAIN"
	AccountLoopiaDNS           AccountType = "LOOPIADNS"
	AccountEmailPrivate        AccountType = "EMAIL_PRIVATE"
	AccountStarter             AccountType = "STARTER"
	AccountHostingPrivate      AccountType = "HOSTING_PRIVATE"
	AccountHostingBusiness     AccountType = "HOSTING_BUSINESS"
	AccountHostingBusinessPlus AccountType = "HOSTING_BUSINESS_PLUS"
)

// DomainConfiguration is the configuration applied to a new domain name.
type DomainConfiguration string

// Domain configurations.
const (
	DomainNoConfig        DomainConfiguration = "NO_CONFIG"
	DomainParking         DomainConfiguration = "PARKING"
	DomainHostingUnix     DomainConfiguration = "HOSTING_UNIX"
	DomainHostingAutobahn DomainConfiguration = "HOSTING_AUTOBAHN"
	DomainHostingWindows  DomainConfiguration = "HOSTING_WINDOWS"
)

// OrderStatus is the status of an order created by [Client.CreateNewAccount].
type OrderStatus string

// Order statuses.
const (
	OrderDeleted   OrderStatus = "DELETED"
	OrderPending   OrderStatus = "PENDING"
	OrderProcessed OrderStatus = "PROCESSED"
)

// Contact is a domain owner, administrative or billing contact (contact).
type Contact struct {
	FirstName string `xmlrpc:"firstname"`
	LastName  string `xmlrpc:"lastname"`
	// Company is empty for a private individual.
	Company string `xmlrpc:"company"`
	Street  string `xmlrpc:"street"`
	Street2 string `xmlrpc:"street2"`
	Zip     string `xmlrpc:"zip"`
	City    string `xmlrpc:"city"`
	// CountryISO2 is a two-letter ISO 3166-1 country code, e.g. "se".
	CountryISO2 string `xmlrpc:"country_iso2"`
	// OrgNo is the organisation number, formatted like 556633-9304.
	OrgNo string `xmlrpc:"orgno"`
	// NoridPID is required for .NO domains ordered by private individuals.
	NoridPID string `xmlrpc:"norid_pid,omitempty"`
	// Phone, Cell and Fax are formatted like 021-128222.
	Phone string `xmlrpc:"phone"`
	Cell  string `xmlrpc:"cell"`
	Fax   string `xmlrpc:"fax"`
	Email string `xmlrpc:"email"`
}

// Customer is a customer connected to a reseller (customer_obj).
type Customer struct {
	// Company is empty for a private individual.
	Company string `xmlrpc:"company"`
	// Name is the name of the contact or account owner.
	Name           string      `xmlrpc:"name"`
	CustomerNumber string      `xmlrpc:"customer_number"`
	AccountType    AccountType `xmlrpc:"account_type"`
}

// Order is the status of an order created by [Client.CreateNewAccount]
// (order_status_obj).
type Order struct {
	Status OrderStatus `xmlrpc:"order_status"`
	// CustomerNumber is empty until the account has been created.
	CustomerNumber string `xmlrpc:"customer_number"`
}

// NewAccount holds the parameters of [Client.CreateNewAccount].
type NewAccount struct {
	// Domain is the username of the new account, and the domain registered
	// when BuyDomain is set. ACE-coded if it is an IDN domain.
	Domain string
	// Owner owns the account and domain.
	Owner Contact
	// BillingContactReseller makes the reseller the billing contact instead
	// of sending invoices directly to the customer.
	BillingContactReseller bool
	// TechContactReseller makes the reseller the technical contact instead of
	// sending technical information directly to the customer.
	TechContactReseller bool
	// BuyDomain registers Domain.
	BuyDomain           bool
	DomainConfiguration DomainConfiguration
	AccountType         AccountType
	// EndUserHasAcceptedTerms must be true: the end customer must have
	// accepted any relevant agreement.
	EndUserHasAcceptedTerms bool
}

// CreateNewAccount creates a new Loopia account, and registers its domain if
// requested. It returns the order reference, which can be passed to
// [Client.GetOrderStatus]. Reseller only.
func (c *Client) CreateNewAccount(ctx context.Context, a NewAccount) (string, error) {
	var res struct {
		Status         Status `xmlrpc:"status"`
		OrderReference string `xmlrpc:"order_reference"`
	}
	err := c.call(ctx, "createNewAccount", false, &res,
		a.Domain,
		a.Owner,
		a.BillingContactReseller,
		a.TechContactReseller,
		a.BuyDomain,
		string(a.DomainConfiguration),
		string(a.AccountType),
		a.EndUserHasAcceptedTerms,
	)
	if err != nil {
		return "", err
	}
	if err := statusErr(res.Status); err != nil {
		return res.OrderReference, err
	}
	return res.OrderReference, nil
}

// GetCustomers returns all customers connected to the reseller.
func (c *Client) GetCustomers(ctx context.Context) ([]Customer, error) {
	var customers []Customer
	if err := c.call(ctx, "getCustomers", false, &customers); err != nil {
		return nil, err
	}
	return customers, nil
}

// GetOrderStatus returns the status of an order created by
// [Client.CreateNewAccount]. Reseller only.
func (c *Client) GetOrderStatus(ctx context.Context, orderReference string) (*Order, error) {
	var o Order
	if err := c.call(ctx, "getOrderStatus", false, &o, orderReference); err != nil {
		return nil, err
	}
	return &o, nil
}

// TransferCreditsByCurrency transfers LoopiaPrePAID credits deposited after
// 2015-12-04 from one account to another. currency is a three-letter ISO 4217
// code; LoopiaAPI currently supports SEK and NOK. Reseller only.
func (c *Client) TransferCreditsByCurrency(ctx context.Context, senderNumber, recipientNumber string, amount float64, currency string) error {
	return c.callStatus(ctx, "transferCreditsByCurrency", false, senderNumber, recipientNumber, amount, currency)
}
