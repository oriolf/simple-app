package app

import (
	"fmt"
	"strconv"
	"time"
)

// TODO test all validate functions
func Validate(errors ...ApiErrors) ApiErrors {
	result := make(ApiErrors)
	for _, err := range errors {
		for k, v := range err {
			result[k] = append(result[k], v...)
		}
	}
	return result
}

func ValidateStringNonEmpty(field, value string) ApiErrors {
	if value == "" {
		return ApiErrors{field: []string{"El camp no pot estar buit"}}
	}
	return nil
}

func ValidateDate(field string, value Date) ApiErrors {
	maxYear := uint(time.Now().Year() + 5)
	if value.Year < 1900 || value.Year > maxYear {
		return ApiErrors{field: []string{fmt.Sprintf("L'any ha de ser posterior a 1900 i anterior a %d", maxYear)}}
	}

	if value.Month < 1 || value.Month > 12 {
		return ApiErrors{field: []string{"El mes ha de ser vàlid (rang 1-12)"}}
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
		return ApiErrors{field: []string{"El camp ha de ser una data vàlida"}}
	}

	return nil
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

func ValidateSpanishDNI(field, value string) ApiErrors {
	if len(value) != 9 {
		return ApiErrors{field: []string{"El DNI ha de tindre exactament 8 dígits i una lletra majúscula"}}
	}

	digits := value[:8]
	number, err := strconv.Atoi(digits)
	if err != nil {
		return ApiErrors{field: []string{"El primers vuit dígits del DNI han de ser números"}}
	}

	controlDigit := value[8:]
	controlNumber, ok := spanishDNIControlDigits[controlDigit]
	if !ok {
		return ApiErrors{field: []string{"La lletra del DNI no és correcta"}}
	}

	if controlNumber != number%23 {
		return ApiErrors{field: []string{"La lletra del DNI no és correcta, o algun dígit no és correcte"}}
	}

	return nil
}
