package verification

import (
	"context"
	"log/slog"

	"github.com/gothi/vouchrs/src/internal/domain/port"
)

type mockVerifier struct {
	logger *slog.Logger
}

// NewMockVerifier returns a VerificationService that always passes.
// Use when VERIFICATION_DEV_MODE=true so Chrome is not required.
func NewMockVerifier(logger *slog.Logger) port.VerificationService {
	return &mockVerifier{logger: logger}
}

func (m *mockVerifier) Verify(_ context.Context, brandSlug, cardCode string) (*port.VerificationResult, error) {
	m.logger.Info("DEV MODE — skipping verification", "brand", brandSlug, "card_code", cardCode)
	return &port.VerificationResult{
		IsValid:      true,
		Balance:      0,
		ResponseHash: "mock",
	}, nil
}
