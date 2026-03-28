package cache

import "fmt"

// Typed key builders — centralise all Redis key formats here.

func OTPKey(contact string) string        { return fmt.Sprintf("otp:%s", contact) }
func OTPAttemptsKey(contact string) string { return fmt.Sprintf("otp_attempts:%s", contact) }
func RevokedTokenKey(token string) string  { return fmt.Sprintf("revoked_token:%s", token) }
func ListingLockKey(listingID string) string { return fmt.Sprintf("listing_lock:%s", listingID) }
func BrandsCacheKey() string               { return "brands:active" }
func PoolGroupsCacheKey() string           { return "pool_groups:active" }
