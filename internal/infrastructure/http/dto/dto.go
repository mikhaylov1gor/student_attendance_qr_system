// Package dto — request/response структуры API + единый validator.
package dto

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"

	"attendance/internal/infrastructure/http/httperr"
)

var (
	validateOnce sync.Once
	validate     *validator.Validate
)

// Validator возвращает ленивый singleton.
func Validator() *validator.Validate {
	validateOnce.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())
	})
	return validate
}

// Decode разбирает JSON-тело и валидирует структуру.
// При ошибке пишет 400 в ответ и возвращает ошибку — хендлер просто выходит.
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	defer r.Body.Close()

	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20)) // 1 MiB
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		httperr.Write(w, http.StatusBadRequest, "invalid_json", "request body is not valid json: "+err.Error())
		return err
	}

	if err := Validator().Struct(dst); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			details := map[string]any{}
			for _, fe := range ve {
				details[fe.Field()] = fmt.Sprintf("failed %q (%v)", fe.Tag(), fe.Param())
			}
			httperr.WriteDetails(w, http.StatusBadRequest, "validation_failed", "validation error", details)
			return err
		}
		httperr.Write(w, http.StatusBadRequest, "validation_failed", err.Error())
		return err
	}
	return nil
}

// NormalizeEmail — lowercase + trim.
func NormalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
