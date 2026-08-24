package config

import (
	"log/slog"
	"os"
	"strings"
)

// Config holds environment-driven configuration for FlowBook API.
// Supabase pooler 6543 is used via DATABASE_URL (pgbouncer=true).
type Config struct {
	Port                string
	DatabaseURL         string
	JWTSecret           string
	RefreshSecret       string
	AppEnv              string
	TestSecret          string
	AllowedOrigins      []string
	StripeSecretKey     string
	StripeWebhookSecret string
	ResendAPIKey        string
	ResendFromEmail     string
	FrontendURL         string
	OpenAIAPIKey        string
}

// Load reads environment variables and returns Config with defaults.
// It logs via slog JSON handler (caller should have set slog default).
func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	refreshSecret := os.Getenv("REFRESH_SECRET")
	if refreshSecret == "" {
		refreshSecret = jwtSecret
	}
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}
	testSecret := os.Getenv("TEST_SECRET")

	allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
	var allowedOrigins []string
	if allowedOriginsEnv != "" {
		parts := strings.Split(allowedOriginsEnv, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				allowedOrigins = append(allowedOrigins, p)
			}
		}
	}
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{
			"https://flowbook-xxx.vercel.app",
			"http://localhost:3000",
		}
	}

	// Stripe + Email + Frontend
	stripeKey := os.Getenv("STRIPE_SECRET_KEY")
	stripeWebhook := os.Getenv("STRIPE_WEBHOOK_SECRET")
	resendKey := os.Getenv("RESEND_API_KEY")
	resendFrom := os.Getenv("RESEND_FROM_EMAIL")
	if resendFrom == "" {
		resendFrom = "FlowBook <noreply@flowbook.example.com>"
	}
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = os.Getenv("NEXT_PUBLIC_API_URL")
		// fallback to vercel placeholder
		if frontendURL == "" {
			frontendURL = "https://flowbook-xxx.vercel.app"
		}
	}
	openAIKey := os.Getenv("OPENAI_API_KEY")

	cfg := Config{
		Port:                port,
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		JWTSecret:           jwtSecret,
		RefreshSecret:       refreshSecret,
		AppEnv:              appEnv,
		TestSecret:          testSecret,
		AllowedOrigins:      allowedOrigins,
		StripeSecretKey:     stripeKey,
		StripeWebhookSecret: stripeWebhook,
		ResendAPIKey:        resendKey,
		ResendFromEmail:     resendFrom,
		FrontendURL:         frontendURL,
		OpenAIAPIKey:        openAIKey,
	}

	if cfg.JWTSecret == "" && cfg.AppEnv != "test" {
		slog.Warn("JWT_SECRET not set", "env", cfg.AppEnv)
	}
	if cfg.DatabaseURL != "" && !strings.Contains(cfg.DatabaseURL, "6543") {
		slog.Warn("DATABASE_URL does not contain pooler 6543 — expected pooler transaction mode", "hint", "use ?pgbouncer=true with port 6543")
	}
	if cfg.StripeSecretKey == "" {
		slog.Info("STRIPE_SECRET_KEY not set — checkout will use mock URL (test smoke still passes)")
	}
	if cfg.StripeWebhookSecret == "" {
		slog.Info("STRIPE_WEBHOOK_SECRET not set — webhook signature verification disabled (only for local/test)")
	}
	if cfg.ResendAPIKey == "" {
		slog.Info("RESEND_API_KEY not set — emails will be logged mock (no external send)")
	}

	return cfg
}

// IsTest returns true if APP_ENV == test.
func (c Config) IsTest() bool {
	return c.AppEnv == "test"
}

// IsProduction returns true if APP_ENV == production.
func (c Config) IsProduction() bool {
	return c.AppEnv == "production"
}
