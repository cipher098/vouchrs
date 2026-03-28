package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// Decode decodes and validates a JSON request body into dst.
// Returns a descriptive error suitable for returning to the client.
func Decode(r *http.Request, dst interface{}) error {
	if r.ContentLength == 0 {
		return errors.New("request body is required")
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if err := validate.Struct(dst); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return fmt.Errorf("validation error: %s", ve[0].Translate(nil))
		}
		return err
	}

	return nil
}
