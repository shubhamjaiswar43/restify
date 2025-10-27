package helper

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ValidateStruct validates a struct.
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// ValidateStructExcept validates a struct except specify keys.
func ValidateStructExcept(s interface{}, exceptKeys ...string) error {
	return validate.StructExcept(s, exceptKeys...)
}

// WriteValidationError returns detailed, human-readable validation errors in JSON.
func WriteValidationError(w http.ResponseWriter, err error) {
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		errorsMap := make(map[string]string)
		for _, e := range validationErrors {
			field := e.Field()
			switch e.Tag() {
			case "required":
				errorsMap[field] = fmt.Sprintf("%s is required", field)
			case "email":
				errorsMap[field] = fmt.Sprintf("%s must be a valid email address", field)
			case "min":
				errorsMap[field] = fmt.Sprintf("%s must be at least %s characters long", field, e.Param())
			case "max":
				errorsMap[field] = fmt.Sprintf("%s must not exceed %s characters", field, e.Param())
			case "oneof":
				errorsMap[field] = fmt.Sprintf("%s must be one of: %s", field, e.Param())
			default:
				errorsMap[field] = fmt.Sprintf("%s is invalid (%s)", field, e.Tag())
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"errors": errorsMap})
		return
	}

	http.Error(w, err.Error(), http.StatusBadRequest)
}
