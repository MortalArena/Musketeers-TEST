package validation

import (
	"testing"
)

func TestDIDValidator_ValidateDID(t *testing.T) {
	validator := NewDIDValidator("did:mskt:")
	
	tests := []struct {
		name    string
		did     string
		wantErr bool
	}{
		{
			name:    "valid DID",
			did:     "did:mskt:1234567890abcdef",
			wantErr: false,
		},
		{
			name:    "empty DID",
			did:     "",
			wantErr: true,
		},
		{
			name:    "wrong prefix",
			did:     "did:other:1234567890abcdef",
			wantErr: true,
		},
		{
			name:    "too short",
			did:     "did:mskt:12",
			wantErr: true,
		},
		{
			name:    "too long",
			did:     "did:mskt:" + string(make([]byte, 300)),
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateDID(tt.did)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStringValidator_Validate(t *testing.T) {
	validator, err := NewStringValidator(3, 50, false, "^[a-zA-Z0-9]+$")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}
	
	tests := []struct {
		name    string
		s       string
		wantErr bool
	}{
		{
			name:    "valid string",
			s:       "test123",
			wantErr: false,
		},
		{
			name:    "empty string",
			s:       "",
			wantErr: true,
		},
		{
			name:    "too short",
			s:       "ab",
			wantErr: true,
		},
		{
			name:    "too long",
			s:       string(make([]byte, 60)),
			wantErr: true,
		},
		{
			name:    "invalid pattern",
			s:       "test-123",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStringValidator_AllowEmpty(t *testing.T) {
	validator, err := NewStringValidator(3, 50, true, "")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}
	
	err = validator.Validate("")
	if err != nil {
		t.Errorf("Expected empty string to be allowed, got error: %v", err)
	}
}

func TestEmailValidator_Validate(t *testing.T) {
	validator := NewEmailValidator(false)
	
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid email",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
		{
			name:    "invalid email",
			email:   "invalid-email",
			wantErr: true,
		},
		{
			name:    "email without @",
			email:   "testexample.com",
			wantErr: true,
		},
		{
			name:    "email without domain",
			email:   "test@",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPortValidator_Validate(t *testing.T) {
	validator := NewPortValidator(1, 65535)
	
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{
			name:    "valid port",
			port:    8080,
			wantErr: false,
		},
		{
			name:    "negative port",
			port:    -1,
			wantErr: true,
		},
		{
			name:    "port too low",
			port:    0,
			wantErr: true,
		},
		{
			name:    "port too high",
			port:    70000,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNumberValidator_Validate(t *testing.T) {
	validator := NewNumberValidator(0, 100, false)
	
	tests := []struct {
		name    string
		n       int64
		wantErr bool
	}{
		{
			name:    "valid number",
			n:       50,
			wantErr: false,
		},
		{
			name:    "negative number",
			n:       -1,
			wantErr: true,
		},
		{
			name:    "number too low",
			n:       -10,
			wantErr: true,
		},
		{
			name:    "number too high",
			n:       150,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNumberValidator_AllowNegative(t *testing.T) {
	validator := NewNumberValidator(-100, 100, true)
	
	err := validator.Validate(-50)
	if err != nil {
		t.Errorf("Expected negative number to be allowed, got error: %v", err)
	}
}

func TestValidateNonEmpty(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		wantErr bool
	}{
		{
			name:    "non-empty string",
			s:       "test",
			wantErr: false,
		},
		{
			name:    "empty string",
			s:       "",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNonEmpty(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNonEmpty() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateLength(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		min     int
		max     int
		wantErr bool
	}{
		{
			name:    "valid length",
			s:       "test",
			min:     3,
			max:     10,
			wantErr: false,
		},
		{
			name:    "too short",
			s:       "ab",
			min:     3,
			max:     10,
			wantErr: true,
		},
		{
			name:    "too long",
			s:       "abcdefghijk",
			min:     3,
			max:     10,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLength(tt.s, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string",
			input:    "test",
			expected: "test",
		},
		{
			name:     "string with null byte",
			input:    "test\x00",
			expected: "test",
		},
		{
			name:     "string with CRLF",
			input:    "test\r\n",
			expected: "test\n",
		},
		{
			name:     "string with spaces",
			input:    "  test  ",
			expected: "test",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeString() = %v, want %v", result, tt.expected)
			}
		})
	}
}
