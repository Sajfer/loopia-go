# loopia-go

[![Go Reference](https://pkg.go.dev/badge/github.com/Sajfer/loopia-go.svg)](https://pkg.go.dev/github.com/Sajfer/loopia-go)

Go client for [LoopiaAPI](https://support.loopia.com/wiki/loopiaapi/), the
XML-RPC API for the Loopia Customer Zone and Reseller Zone.

```sh
go get github.com/Sajfer/loopia-go
```

## Usage

Create an API user in the Loopia Customer Zone under
**Account Settings → LoopiaAPI**, then:

```go
client := loopia.New("user@loopiaapi", "password")

records, err := client.GetZoneRecords(ctx, "example.com", "@")
if err != nil {
	log.Fatal(err)
}

record := loopia.Record{Type: "A", TTL: 3600, Value: "192.0.2.10"}
err = client.AddZoneRecord(ctx, "example.com", "www", &record) // fills record.ID
```

Resellers can act on a customer's account:

```go
domains, err := client.ForCustomer("C12345").GetDomains(ctx)
```

### Errors

When LoopiaAPI answers with a status other than `OK`, methods return a
`*loopia.StatusError`. XML-RPC faults are returned as `*loopia.FaultError`.
Both work with `errors.Is`:

```go
switch {
case errors.Is(err, loopia.ErrAuth):
	// wrong username or password
case errors.Is(err, loopia.ErrRateLimited):
	// more than 60 calls per minute
case errors.Is(err, &loopia.FaultError{Code: loopia.FaultDomainNotAvailable}):
	// a specific fault code
}
```

Loopia reports some problems either as a status or as a fault (invalid
credentials come back as fault 401, not `AUTH_ERROR`), so `ErrAuth`,
`ErrRateLimited` and `ErrInsufficientFunds` match both forms.

### Options

```go
loopia.New(user, pass,
	loopia.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
	loopia.WithEndpoint("https://api.loopia.rs/RPCSERV"),
)
```

## Supported methods

| LoopiaAPI                   | Go                                   |
| --------------------------- | ------------------------------------ |
| `addDomain`                 | `AddDomain`                          |
| `addSubdomain`              | `AddSubdomain`                       |
| `addZoneRecord`             | `AddZoneRecord`                      |
| `domainIsFree`              | `DomainIsFree`                       |
| `getCreditsAmount`          | `GetCreditsAmount`                   |
| `getDomain`                 | `GetDomain`                          |
| `getDomains`                | `GetDomains`                         |
| `getInvoice`                | `GetInvoice`                         |
| `getSubdomains`             | `GetSubdomains`                      |
| `getUnpaidInvoices`         | `GetUnpaidInvoices`                  |
| `getZoneRecords`            | `GetZoneRecords`, `GetZoneRecord`    |
| `orderDomain`               | `OrderDomain`                        |
| `payInvoiceUsingCredits`    | `PayInvoiceUsingCredits`             |
| `removeDomain`              | `RemoveDomain`                       |
| `removeSubdomain`           | `RemoveSubdomain`                    |
| `removeZoneRecord`          | `RemoveZoneRecord`                   |
| `transferDomain`            | `TransferDomain`                     |
| `updateDNSServers`          | `UpdateDNSServers`                   |
| `updateZoneRecord`          | `UpdateZoneRecord`                   |
| `createNewAccount` (reseller)         | `CreateNewAccount`         |
| `getCustomers` (reseller)             | `GetCustomers`             |
| `getOrderStatus` (reseller)           | `GetOrderStatus`           |
| `transferCreditsByCurrency` (reseller)| `TransferCreditsByCurrency`|

## Upgrading from the pre-module version

- Import path is `github.com/Sajfer/loopia-go`; `API` is now `Client` and
  `New` no longer returns an error.
- Every method takes a `context.Context` first.
- Status methods return `error` instead of `(*Status, error)`. A non-`OK`
  status is now always an error (previously some were silently dropped).
- `GetSubdomains` returns `[]string`; `Subdomain` and `GetSubdomain` are gone
  (use `slices.Contains`).
- `RemoveSubDomain` is renamed `RemoveSubdomain`.
- `AddZoneRecord` returns `ErrRecordIDUnknown` if the record was created but
  its ID could not be found.

## License

MIT
