package contextkey

type ctxKey string

const (
	UserID    ctxKey = "user_id"
	UserRole  ctxKey = "user_role"
	UserEmail ctxKey = "user_email"
	RequestID ctxKey = "request_id"
)
