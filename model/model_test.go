package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestNewAccessTokenJwt tests the NewAccessTokenJwt function
func TestNewAccessTokenJwt(t *testing.T) {
	// Define test cases
	tests := []struct {
		name        string
		accessToken string
		wantErr     bool
	}{
		{
			name:        "valid access token",
			accessToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImNiYjEwNWI4LWI0NTktNDY2NC1hMzQ1LTdiZWI0NzAyMTAwNiIsInR5cGUiOiJhY2Nlc3NfdG9rZW4iLCJyb2xlIjoidXNlciIsInByZW1pdW0iOmZhbHNlLCJpc3MiOiJpd2FyYSIsImlhdCI6MTcyNjkzNzcxOCwiZXhwIjoxNzI2OTQxMzE4fQ.z_fl2pytNXJkZ0d4AZ84dlKWm9CRZvk4GGTL1ntPwOw",
			wantErr:     false,
		},
		{
			name:        "invalid access token",
			accessToken: "invalid.token.here",
			wantErr:     true,
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAccessTokenJwt(tt.accessToken)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				// You can add additional assertions to check the contents of got
			}
		})
	}
}
