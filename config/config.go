package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	App       AppConfig
	DB        DBConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Cipher    CipherConfig
	R2        R2Config
	Fast2SMS  Fast2SMSConfig
	Resend    ResendConfig
	PhonePe   PhonePeConfig
	Razorpay  RazorpayConfig
	Google    GoogleConfig
	Admin     AdminConfig
	Qwikcilver QwikcilverConfig
	OTP       OTPConfig
	Asynq     AsynqConfig
}

type AppConfig struct {
	Env     string
	Port    string
	BaseURL string
}

type DBConfig struct {
	DSN string
}

type RedisConfig struct {
	URL      string // takes precedence when set (e.g. rediss://... for Upstash)
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTLMin  int
	RefreshTTLDay int
}

type CipherConfig struct {
	Key string // 32-byte hex-encoded key
}

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	AccessKeySecret string
	Bucket          string
	PublicURL       string
}

type Fast2SMSConfig struct {
	APIKey string
}

type ResendConfig struct {
	APIKey string
	From   string
}

type PhonePeConfig struct {
	MerchantID  string
	SaltKey     string
	SaltIndex   string
	Env         string // UAT or PROD
}

type RazorpayConfig struct {
	KeyID         string
	KeySecret     string
	AccountNumber string
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type AdminConfig struct {
	Emails []string
}

type QwikcilverConfig struct {
	TimeoutSeconds int
	Headless       bool
}

type OTPConfig struct {
	Length     int
	TTLMinutes int
	DevMode    bool // if true, print OTP to logs instead of sending via SMS/email
}

type AsynqConfig struct {
	Concurrency int
}

// Load reads configuration from environment variables.
// It returns an error if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{}

	cfg.App = AppConfig{
		Env:     getEnv("APP_ENV", "development"),
		Port:    getEnvAny("APP_PORT", "PORT", "8080"),
		BaseURL: getEnv("APP_BASE_URL", "http://localhost:8080"),
	}

	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		return nil, fmt.Errorf("DB_DSN is required")
	}
	cfg.DB = DBConfig{DSN: dbDSN}

	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	cfg.Redis = RedisConfig{
		URL:      os.Getenv("REDIS_URL"),
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       redisDB,
	}

	accessTTL, _ := strconv.Atoi(getEnv("JWT_ACCESS_TTL_MINUTES", "60"))
	refreshTTL, _ := strconv.Atoi(getEnv("JWT_REFRESH_TTL_DAYS", "30"))
	cfg.JWT = JWTConfig{
		AccessSecret:  requireEnv("JWT_ACCESS_SECRET"),
		RefreshSecret: requireEnv("JWT_REFRESH_SECRET"),
		AccessTTLMin:  accessTTL,
		RefreshTTLDay: refreshTTL,
	}

	cfg.Cipher = CipherConfig{Key: requireEnv("CIPHER_KEY")}

	cfg.R2 = R2Config{
		AccountID:       requireEnv("R2_ACCOUNT_ID"),
		AccessKeyID:     requireEnv("R2_ACCESS_KEY_ID"),
		AccessKeySecret: requireEnv("R2_ACCESS_KEY_SECRET"),
		Bucket:          getEnv("R2_BUCKET", "cardswap"),
		PublicURL:       requireEnv("R2_PUBLIC_URL"),
	}

	cfg.Fast2SMS = Fast2SMSConfig{APIKey: requireEnv("FAST2SMS_API_KEY")}

	cfg.Resend = ResendConfig{
		APIKey: requireEnv("RESEND_API_KEY"),
		From:   getEnv("RESEND_FROM", "CardSwap <noreply@cardswap.in>"),
	}

	cfg.PhonePe = PhonePeConfig{
		MerchantID: requireEnv("PHONEPE_MERCHANT_ID"),
		SaltKey:    requireEnv("PHONEPE_SALT_KEY"),
		SaltIndex:  getEnv("PHONEPE_SALT_INDEX", "1"),
		Env:        getEnv("PHONEPE_ENV", "UAT"),
	}

	cfg.Razorpay = RazorpayConfig{
		KeyID:         requireEnv("RAZORPAY_KEY_ID"),
		KeySecret:     requireEnv("RAZORPAY_KEY_SECRET"),
		AccountNumber: requireEnv("RAZORPAY_ACCOUNT_NUMBER"),
	}

	cfg.Google = GoogleConfig{
		ClientID:     requireEnv("GOOGLE_CLIENT_ID"),
		ClientSecret: requireEnv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  requireEnv("GOOGLE_REDIRECT_URL"),
	}

	adminEmails := os.Getenv("ADMIN_EMAILS")
	if adminEmails == "" {
		return nil, fmt.Errorf("ADMIN_EMAILS is required")
	}
	cfg.Admin = AdminConfig{
		Emails: strings.Split(adminEmails, ","),
	}

	qTimeout, _ := strconv.Atoi(getEnv("QWIKCILVER_TIMEOUT_SECONDS", "30"))
	qHeadless, _ := strconv.ParseBool(getEnv("QWIKCILVER_HEADLESS", "true"))
	cfg.Qwikcilver = QwikcilverConfig{
		TimeoutSeconds: qTimeout,
		Headless:       qHeadless,
	}

	otpLen, _ := strconv.Atoi(getEnv("OTP_LENGTH", "6"))
	otpTTL, _ := strconv.Atoi(getEnv("OTP_TTL_MINUTES", "10"))
	otpDevMode, _ := strconv.ParseBool(getEnv("OTP_DEV_MODE", "false"))
	cfg.OTP = OTPConfig{
		Length:     otpLen,
		TTLMinutes: otpTTL,
		DevMode:    otpDevMode,
	}

	concurrency, _ := strconv.Atoi(getEnv("ASYNQ_CONCURRENCY", "10"))
	cfg.Asynq = AsynqConfig{Concurrency: concurrency}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvAny returns the value of the first key that is set, falling back to the default.
func getEnvAny(keys ...string) string {
	// last element is the fallback
	fallback := keys[len(keys)-1]
	for _, key := range keys[:len(keys)-1] {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		// Panic is intentional — missing required config is a startup failure.
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}
