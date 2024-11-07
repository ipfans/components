package password

import "testing"

func TestCreateHashAndVerify(t *testing.T) {
	m := New()
	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "empty password",
			password: "",
		},
		{
			name:     "simple password",
			password: "password123",
		},
		{
			name:     "complex password",
			password: "P@ssw0rd!@#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := m.Generate(tt.password)

			if !m.Verify(tt.password, hash) {
				t.Error("Verify() should return true for correct password")
			}

			if m.Verify(tt.password+"wrong", hash) {
				t.Error("Verify() should return false for incorrect password")
			}
		})
	}
}
