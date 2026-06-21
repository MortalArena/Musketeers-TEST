package validation

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrEmptyString      = errors.New("string cannot be empty")
	ErrTooShort         = errors.New("string too short")
	ErrTooLong          = errors.New("string too long")
	ErrInvalidFormat    = errors.New("invalid format")
	ErrInvalidDID       = errors.New("invalid DID format")
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrInvalidURL       = errors.New("invalid URL format")
	ErrInvalidPort      = errors.New("invalid port number")
	ErrNegativeNumber   = errors.New("number cannot be negative")
	ErrExceedsLimit     = errors.New("value exceeds limit")
)

// DIDValidator validates DID format
type DIDValidator struct {
	didPrefix string
}

// NewDIDValidator creates new DID validator
func NewDIDValidator(prefix string) *DIDValidator {
	return &DIDValidator{didPrefix: prefix}
}

// ValidateDID validates DID format
func (v *DIDValidator) ValidateDID(did string) error {
	if did == "" {
		return fmt.Errorf("%w: DID cannot be empty", ErrEmptyString)
	}
	
	if !strings.HasPrefix(did, v.didPrefix) {
		return fmt.Errorf("%w: must start with %s", ErrInvalidDID, v.didPrefix)
	}
	
	if len(did) < 10 {
		return fmt.Errorf("%w: too short", ErrInvalidDID)
	}
	
	if len(did) > 256 {
		return fmt.Errorf("%w: too long", ErrInvalidDID)
	}
	
	return nil
}

// StringValidator validates string input
type StringValidator struct {
	minLength int
	maxLength int
	allowEmpty bool
	pattern   *regexp.Regexp
}

// NewStringValidator creates new string validator
func NewStringValidator(minLength, maxLength int, allowEmpty bool, pattern string) (*StringValidator, error) {
	var regex *regexp.Regexp
	if pattern != "" {
		var err error
		regex, err = regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}
	
	return &StringValidator{
		minLength:  minLength,
		maxLength:  maxLength,
		allowEmpty: allowEmpty,
		pattern:    regex,
	}, nil
}

// Validate validates string
func (v *StringValidator) Validate(s string) error {
	if s == "" {
		if !v.allowEmpty {
			return ErrEmptyString
		}
		return nil
	}
	
	if len(s) < v.minLength {
		return fmt.Errorf("%w: minimum length is %d", ErrTooShort, v.minLength)
	}
	
	if v.maxLength > 0 && len(s) > v.maxLength {
		return fmt.Errorf("%w: maximum length is %d", ErrTooLong, v.maxLength)
	}
	
	if v.pattern != nil && !v.pattern.MatchString(s) {
		return ErrInvalidFormat
	}
	
	return nil
}

// EmailValidator validates email format
type EmailValidator struct {
	allowEmpty bool
}

// NewEmailValidator creates new email validator
func NewEmailValidator(allowEmpty bool) *EmailValidator {
	return &EmailValidator{allowEmpty: allowEmpty}
}

// Validate validates email
func (v *EmailValidator) Validate(email string) error {
	if email == "" {
		if !v.allowEmpty {
			return ErrEmptyString
		}
		return nil
	}
	
	// Simple email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	
	return nil
}

// PortValidator validates port number
type PortValidator struct {
	minPort int
	maxPort int
}

// NewPortValidator creates new port validator
func NewPortValidator(minPort, maxPort int) *PortValidator {
	return &PortValidator{
		minPort: minPort,
		maxPort: maxPort,
	}
}

// Validate validates port
func (v *PortValidator) Validate(port int) error {
	if port < 0 {
		return ErrNegativeNumber
	}
	
	if port < v.minPort {
		return fmt.Errorf("%w: minimum port is %d", ErrInvalidPort, v.minPort)
	}
	
	if port > v.maxPort {
		return fmt.Errorf("%w: maximum port is %d", ErrInvalidPort, v.maxPort)
	}
	
	return nil
}

// NumberValidator validates number range
type NumberValidator struct {
	min      int64
	max      int64
	allowNegative bool
}

// NewNumberValidator creates new number validator
func NewNumberValidator(min, max int64, allowNegative bool) *NumberValidator {
	return &NumberValidator{
		min:      min,
		max:      max,
		allowNegative: allowNegative,
	}
}

// Validate validates number
func (v *NumberValidator) Validate(n int64) error {
	if n < 0 && !v.allowNegative {
		return ErrNegativeNumber
	}
	
	if n < v.min {
		return fmt.Errorf("%w: minimum value is %d", ErrExceedsLimit, v.min)
	}
	
	if v.max > 0 && n > v.max {
		return fmt.Errorf("%w: maximum value is %d", ErrExceedsLimit, v.max)
	}
	
	return nil
}

// ValidateNonEmpty validates that string is not empty
func ValidateNonEmpty(s string) error {
	if s == "" {
		return ErrEmptyString
	}
	return nil
}

// ValidateLength validates string length
func ValidateLength(s string, min, max int) error {
	if len(s) < min {
		return fmt.Errorf("%w: minimum length is %d", ErrTooShort, min)
	}
	if max > 0 && len(s) > max {
		return fmt.Errorf("%w: maximum length is %d", ErrTooLong, max)
	}
	return nil
}

// SanitizeString removes potentially dangerous characters
func SanitizeString(s string) string {
	// Remove null bytes and other control characters
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSpace(s)
	return s
}
