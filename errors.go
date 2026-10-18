package loopia

import "fmt"

// Status is the status code returned by methods that do not return data.
type Status string

// Status codes returned by LoopiaAPI.
const (
	StatusOK                Status = "OK"
	StatusAuthError         Status = "AUTH_ERROR"
	StatusDomainOccupied    Status = "DOMAIN_OCCUPIED"
	StatusRateLimited       Status = "RATE_LIMITED"
	StatusBadIndata         Status = "BAD_INDATA"
	StatusUnknownError      Status = "UNKNOWN_ERROR"
	StatusInsufficientFunds Status = "INSUFFICIENT_FUNDS"
)

func (s Status) isError() bool {
	switch s {
	case StatusAuthError, StatusDomainOccupied, StatusRateLimited,
		StatusBadIndata, StatusUnknownError, StatusInsufficientFunds:
		return true
	}
	return false
}

// StatusError is returned when LoopiaAPI answers with a status other than OK.
// Use [errors.Is] with the Err* variables to check for a specific status;
// ErrAuth, ErrRateLimited and ErrInsufficientFunds also match the equivalent
// [FaultError].
type StatusError struct {
	Status Status
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("loopia: %s", e.Status)
}

// Is reports whether target is a *StatusError with the same status.
func (e *StatusError) Is(target error) bool {
	t, ok := target.(*StatusError)
	return ok && t.Status == e.Status
}

// Sentinel errors for each non-OK status, for use with [errors.Is].
var (
	ErrAuth              = &StatusError{Status: StatusAuthError}
	ErrDomainOccupied    = &StatusError{Status: StatusDomainOccupied}
	ErrRateLimited       = &StatusError{Status: StatusRateLimited}
	ErrBadIndata         = &StatusError{Status: StatusBadIndata}
	ErrUnknown           = &StatusError{Status: StatusUnknownError}
	ErrInsufficientFunds = &StatusError{Status: StatusInsufficientFunds}
)

// Fault codes returned by LoopiaAPI in XML-RPC fault responses.
const (
	FaultUnknown                  = 404
	FaultUnauthorized             = 401
	FaultRateLimited              = 429
	FaultMethodNotFound           = 620
	FaultBadSignature             = 623
	FaultParseError               = 631
	FaultNoMethod                 = 632
	FaultMissingValue             = 633
	FaultInvalidMethod            = 634
	FaultInvalidParameter         = 637
	FaultMethodNotAllowed         = 639
	FaultDomainNotAvailable       = 40001
	FaultDomainInvalid            = 40004
	FaultDomainConfigInvalid      = 40011
	FaultAccountTypeInvalid       = 40014
	FaultDomainConfigNotAvailable = 40015
	FaultCustomerNotConnected     = 40401
	FaultAccountParametersInvalid = 50001
	FaultInsufficientFunds        = 60001
)

// FaultError is an XML-RPC fault returned by LoopiaAPI. Code is one of the
// Fault* constants.
type FaultError struct {
	Code    int
	Message string
}

func (e *FaultError) Error() string {
	return fmt.Sprintf("loopia: fault %d: %s", e.Code, e.Message)
}

// faultStatus maps fault codes to the status with the same meaning. LoopiaAPI
// reports some problems either way; invalid credentials, for example, come
// back as fault 401 rather than AUTH_ERROR.
var faultStatus = map[int]Status{
	FaultUnauthorized:      StatusAuthError,
	FaultRateLimited:       StatusRateLimited,
	FaultInsufficientFunds: StatusInsufficientFunds,
}

// Is reports whether target is a *FaultError with the same code, or a
// *StatusError with the same meaning, so that errors.Is(err, ErrAuth) also
// matches fault 401.
func (e *FaultError) Is(target error) bool {
	switch t := target.(type) {
	case *FaultError:
		return t.Code == e.Code
	case *StatusError:
		s, ok := faultStatus[e.Code]
		return ok && s == t.Status
	}
	return false
}
