package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantPanic bool
	}{
		{
			name: "valid HS256",
			cfg: Config{
				SecretKey: "test-secret",
				Algorithm: "HS256",
				Expire:    time.Hour,
			},
			wantPanic: false,
		},
		{
			name: "invalid algorithm",
			cfg: Config{
				SecretKey: "test-secret-key-must-be-at-least-32-bytes",
				Algorithm: "INVALID",
				Expire:    time.Hour,
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				require.Panics(t, func() { New(tt.cfg) })
				return
			}
			require.NotPanics(t, func() { New(tt.cfg) })
		})
	}
}

func TestManager_Generate_And_Parser(t *testing.T) {
	cfg := Config{
		SecretKey: "test-secret-key-must-be-at-least-32-bytes",
		Algorithm: "HS256",
		Expire:    time.Hour,
	}
	manager := New(cfg)

	// Test Generate
	userID := uint(123)
	token, id, err := manager.Generate(userID)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, id)

	// Test Parser
	claims, err := manager.Parse(token)
	require.NoError(t, err)
	require.Equal(t, "123", claims.Subject)
	require.NotNil(t, claims.IssuedAt)
	require.NotNil(t, claims.Expiry)
	require.Equal(t, id, claims.ID)

	// Test invalid token
	_, err = manager.Parse("invalid-token")
	require.Error(t, err)
}

func TestManager_Different_Algorithms(t *testing.T) {
	tests := []struct {
		name      string
		algorithm string
		key       string
	}{
		{
			name:      "HS256",
			algorithm: "HS256",
			key:       "test-secret-key-must-be-at-least-32-bytes",
		},
		{
			name:      "HS384",
			algorithm: "HS384",
			key:       "test-secret-key-must-be-at-least-48-bytes-long-enough-key",
		},
		{
			name:      "HS512",
			algorithm: "HS512",
			key:       "test-secret-key-must-be-at-least-64-bytes-long-enough-key-for-hs512-algo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				SecretKey: tt.key,
				Algorithm: tt.algorithm,
				Expire:    time.Hour,
			}
			manager := New(cfg)

			token, _, err := manager.Generate(123)
			require.NoError(t, err)

			claims, err := manager.Parse(token)
			require.NoError(t, err)
			require.Equal(t, "123", claims.Subject)
		})
	}
}
