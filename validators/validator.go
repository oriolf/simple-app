package validators

import (
	"net/mail"
	"strconv"
	"strings"

	app "github.com/oriolf/simple-app"
)

type validator struct {
	errors app.ApiErrors
	params map[string]any
}

func NewValidator(params map[string]any) validator {
	return validator{params: params, errors: app.ApiErrors{Fields: make(map[string][]string)}}
}

func (v validator) HasError(field string) bool {
	return len(v.errors.Fields[field]) > 0
}

func (v validator) AddError(field, msg string) {
	v.errors.Fields[field] = append(v.errors.Fields[field], msg)
}

func (v validator) Errors() app.ApiErrors { return v.errors }

func (v *validator) validateStringPresent(field string) string {
	s, ok := v.params[field]
	if !ok {
		v.AddError(field, "El camp ha d'estar present")
		return ""
	}
	switch s := s.(type) {
	case string:
		return strings.TrimSpace(s)
	}
	v.AddError(field, "El camp ha de ser una cadena de text")
	return ""
}

func (v *validator) validateNumberPresent(field string) float64 {
	n, ok := v.params[field]
	if !ok {
		v.AddError(field, "El camp ha d'estar present")
		return 0
	}
	switch n := n.(type) {
	case float64:
		return n
	case string:
		num, err := strconv.ParseFloat(n, 64)
		if err != nil {
			v.AddError(field, err.Error())
			return 0
		}
		return num
	}
	v.AddError(field, "El camp ha de ser un número")
	return 0
}

func (v *validator) ValidateInt(field string) int {
	value := v.validateNumberPresent(field)
	if v.HasError(field) {
		return 0
	}
	if float64(int(value)) != value {
		v.AddError(field, "El camp ha de ser un enter")
		return 0
	}
	return int(value)
}

func (v *validator) ValidatePositiveInt(field string) int {
	value := v.ValidateInt(field)
	if v.HasError(field) {
		return 0
	}
	if value <= 0 {
		v.AddError(field, "El número ha de ser positiu")
		return 0
	}
	return value
}

func (v *validator) ValidateStringNonEmpty(field string) string {
	value := v.validateStringPresent(field)
	if v.HasError(field) {
		return ""
	}
	if value == "" {
		v.AddError(field, "El camp no pot estar buit")
		return ""
	}
	return value
}

func (v *validator) ValidatePassword(field string) string {
	return v.ValidateStringNonEmpty(field)
}

func (v *validator) ValidateEmail(field string) string {
	email := v.ValidateStringNonEmpty(field)
	if v.HasError(field) {
		return email
	}

	if _, err := mail.ParseAddress(email); err != nil {
		v.AddError(field, "L'adreça de correu electrònic no té un format vàlid")
	}

	return email
}
