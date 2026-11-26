package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
	coreerrors "github.com/makmanu/golang_todoapp/internal/core/errors"
)

const MaxBodyBytes = 1 << 20

var requestValidator = validator.New(validator.WithRequiredStructEnabled())

type validatable interface{ Validate() error }

func DecodeAndValidateRequest(r *http.Request, dest any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxBodyBytes+1))
	if err != nil {
		return fmt.Errorf("read JSON: %w: %w", coreerrors.ErrInvalidArgument, err)
	}
	if len(body) > MaxBodyBytes {
		return fmt.Errorf("request body too large: %w", coreerrors.ErrInvalidArgument)
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 || body[0] != '{' || !utf8.Valid(body) {
		return fmt.Errorf("body must be a JSON object: %w", coreerrors.ErrInvalidArgument)
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return fmt.Errorf("decode JSON: %w: %w", coreerrors.ErrInvalidArgument, err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return fmt.Errorf("body must contain one JSON object: %w", coreerrors.ErrInvalidArgument)
	}
	if v, ok := dest.(validatable); ok {
		return wrapValidationErr(v.Validate())
	}
	return wrapValidationErr(requestValidator.Struct(dest))
}
func wrapValidationErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("validate request: %w: %w", coreerrors.ErrInvalidArgument, err)
}
