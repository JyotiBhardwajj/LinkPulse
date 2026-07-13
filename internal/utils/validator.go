// Package utils provides common helper functions.
package utils

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"

	"linkpulse/internal/models"

	"github.com/go-playground/validator/v10"
)

// IsValidURL validates if a string is a well-formed HTTP/HTTPS URL.
func IsValidURL(toTest string) bool {
	u, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.Host == "" || !strings.Contains(u.Host, ".") {
		return false
	}
	return true
}

// FormatValidationErrors parses validation errors into a sorted list of ValidationError DTOs.
func FormatValidationErrors(err error, req interface{}) []models.ValidationError {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return []models.ValidationError{
			{
				Field:   "body",
				Rule:    "malformed",
				Message: "Malformed request payload",
			},
		}
	}

	details := make([]models.ValidationError, 0, len(ve))
	t := reflect.TypeOf(req)
	if t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for _, fe := range ve {
		field := fe.Field()
		if t != nil && t.Kind() == reflect.Struct {
			if structField, ok := t.FieldByName(fe.StructField()); ok {
				if jsonTag := structField.Tag.Get("json"); jsonTag != "" {
					field = strings.Split(jsonTag, ",")[0]
				} else if formTag := structField.Tag.Get("form"); formTag != "" {
					field = strings.Split(formTag, ",")[0]
				}
			}
		}

		message := formatMessage(field, fe.Tag(), fe.Param())
		details = append(details, models.ValidationError{
			Field:   field,
			Rule:    fe.Tag(),
			Message: message,
		})
	}

	// Sort validation errors deterministically by Field name alphabetically
	sort.Slice(details, func(i, j int) bool {
		return details[i].Field < details[j].Field
	})

	return details
}

func formatMessage(field, tag, param string) string {
	switch tag {
	case "required":
		return fmt.Sprintf("The %s field is required", field)
	case "email":
		return fmt.Sprintf("The %s field must be a valid email address", field)
	case "url":
		return fmt.Sprintf("The %s field must be a valid absolute URL (http or https)", field)
	case "uuid":
		return fmt.Sprintf("The %s field must be a valid UUID", field)
	case "min":
		return fmt.Sprintf("The %s field must be at least %s characters or value", field, param)
	case "max":
		return fmt.Sprintf("The %s field must be at most %s characters or value", field, param)
	case "oneof":
		return fmt.Sprintf("The %s field must be one of: %s", field, strings.Join(strings.Split(param, " "), ", "))
	case "datetime":
		return fmt.Sprintf("The %s field must be a valid RFC3339 datetime (e.g. 2006-01-02T15:04:05Z)", field)
	default:
		return fmt.Sprintf("The %s field failed validation constraint: %s", field, tag)
	}
}
