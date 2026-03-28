package apperror

import "errors"

// Sentinel errors — use errors.Is() to check.
var (
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrBadRequest        = errors.New("bad request")
	ErrUnprocessable     = errors.New("unprocessable")
	ErrInternal          = errors.New("internal error")

	// Domain-specific
	ErrListingLocked     = errors.New("listing is currently locked by another buyer")
	ErrListingNotLive    = errors.New("listing is not available for purchase")
	ErrLockExpired       = errors.New("payment lock has expired")
	ErrDuplicateCard     = errors.New("this card code is already listed")
	ErrVerificationFailed = errors.New("card verification failed")
	ErrCardTampered      = errors.New("card balance changed since listing — possible fraud")
	ErrInsufficientBalance = errors.New("insufficient wallet balance")
	ErrOTPInvalid         = errors.New("OTP is invalid or expired")
	ErrOTPTooManyAttempts = errors.New("too many OTP attempts, please try again later")
	ErrOTPCooldown        = errors.New("please wait 60 seconds before requesting another OTP")
	ErrOTPIPLimit         = errors.New("too many OTP requests from this IP, please try again later")
	ErrListingLimitReached = errors.New("daily listing limit reached")
	ErrNotListingOwner   = errors.New("you are not the owner of this listing")
	ErrNotTransactionParty = errors.New("you are not a party to this transaction")
	ErrPaymentNotConfirmed = errors.New("payment has not been confirmed")
	ErrAlreadyCompleted  = errors.New("transaction is already completed")

	// ErrCacheMiss is returned by CacheService.Get when the key does not exist.
	ErrCacheMiss = errors.New("cache miss")
)

// AppError wraps a sentinel error with a human-readable message.
type AppError struct {
	Code    error
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Code
}

// New creates an AppError with a custom message.
func New(code error, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Is allows errors.Is() to match AppError against sentinel codes.
func Is(err, target error) bool {
	return errors.Is(err, target)
}
