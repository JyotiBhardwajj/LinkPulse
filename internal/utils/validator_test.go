package utils

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

type testStruct struct {
	EmailVal    string `json:"email_val" validate:"required,email"`
	PasswordVal string `json:"password_val" validate:"required,min=8"`
}

func TestValidationFormatting(t *testing.T) {
	validate := validator.New()

	t.Run("FormatValidationErrors - Success", func(t *testing.T) {
		req := testStruct{
			EmailVal:    "invalid-email",
			PasswordVal: "short",
		}

		err := validate.Struct(req)
		assert.Error(t, err)

		formatted := FormatValidationErrors(err, req)

		assert.Equal(t, 2, len(formatted))
		// Sorted alphabetically: email_val before password_val
		assert.Equal(t, "email_val", formatted[0].Field)
		assert.Equal(t, "email", formatted[0].Rule)
		assert.Contains(t, formatted[0].Message, "must be a valid email address")

		assert.Equal(t, "password_val", formatted[1].Field)
		assert.Equal(t, "min", formatted[1].Rule)
		assert.Contains(t, formatted[1].Message, "must be at least 8 characters")
	})

	t.Run("FormatValidationErrors - Malformed", func(t *testing.T) {
		var malformedErr error = assert.AnError
		formatted := FormatValidationErrors(malformedErr, nil)
		assert.Equal(t, 1, len(formatted))
		assert.Equal(t, "body", formatted[0].Field)
		assert.Equal(t, "malformed", formatted[0].Rule)
	})
}
