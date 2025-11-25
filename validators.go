package app

import (
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

// TODO create type Validator and make validate calls be like res =
// validator.ValidateInt(key), avoiding passing errors and params

type validator struct {
	errors ApiErrors
	params map[string]any
}

func NewValidator(params map[string]any) validator {
	return validator{params: params, errors: make(ApiErrors)}
}

func (v validator) HasError(field string) bool {
	return len(v.errors[field]) > 0
}

func (v validator) AddError(field, msg string) {
	v.errors[field] = append(v.errors[field], msg)
}

func (v validator) Errors() ApiErrors { return v.errors }

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

func (v *validator) ValidateDate(field string) Date {
	s := v.validateStringPresent(field)
	if v.HasError(field) {
		return Date{}
	}

	var value Date
	err := (&value).Scan(s)
	if err != nil {
		if strings.Contains(err.Error(), "EOF") {
			v.AddError(field, "El camp ha de ser una data completa en format YYYY-mm-dd")
			return value
		}
		v.AddError(field, err.Error())
		return value
	}

	maxYear := uint(time.Now().Year() + 5)
	if value.Year < 1900 || value.Year > maxYear {
		v.AddError(field, fmt.Sprintf("L'any ha de ser posterior a 1900 i anterior a %d", maxYear))
		return value
	}

	if value.Month < 1 || value.Month > 12 {
		v.AddError(field, "El mes ha de ser vàlid (rang 1-12)")
		return value
	}

	var maxDay uint = 31
	if InSlice(uint(value.Month), []uint{4, 6, 9, 11}) {
		maxDay = 30
	} else if value.Month == 2 {
		maxDay = 28
		if isLeapYear(value.Year) {
			maxDay = 29
		}
	}
	if value.Day < 1 || value.Day > maxDay {
		v.AddError(field, "El camp ha de ser una data vàlida")
		return value
	}

	return value
}

func isLeapYear(year uint) bool {
	if year%400 == 0 {
		return true
	}
	return year%4 == 0 && year%100 != 0
}

var spanishDNIControlDigits = map[string]int{
	"T": 0,
	"R": 1,
	"W": 2,
	"A": 3,
	"G": 4,
	"M": 5,
	"Y": 6,
	"F": 7,
	"P": 8,
	"D": 9,
	"X": 10,
	"B": 11,
	"N": 12,
	"J": 13,
	"Z": 14,
	"S": 15,
	"Q": 16,
	"V": 17,
	"H": 18,
	"L": 19,
	"C": 20,
	"K": 21,
	"E": 22,
}

func (v *validator) ValidateSpanishDNI(field string) string {
	value := v.validateStringPresent(field)
	if v.HasError(field) {
		return value
	}

	value = strings.ToUpper(value)
	if len(value) != 9 {
		v.AddError(field, "El DNI ha de tindre exactament 8 dígits i una lletra")
		return value
	}

	digits := value[:8]
	number, err := strconv.Atoi(digits)
	if err != nil {
		v.AddError(field, "El primers vuit dígits del DNI han de ser números")
		return value
	}

	controlDigit := value[8:]
	controlNumber, ok := spanishDNIControlDigits[controlDigit]
	if !ok {
		v.AddError(field, "La lletra del DNI no és correcta")
		return value
	}

	if controlNumber != number%23 {
		v.AddError(field, "La lletra del DNI no és correcta, o algun dígit no és correcte")
		return value
	}

	return value
}
