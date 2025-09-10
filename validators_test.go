package app

import (
	"fmt"
	"testing"
	"time"
)

type ValidationTestCase struct {
	input          any
	expectedError  string
	expectedString string
	expectedDate   Date
}

func TestValidateStringNonEmpty(t *testing.T) {
	for i, tc := range []ValidationTestCase{
		{input: nil, expectedString: "", expectedError: "El camp ha de ser una cadena de text"},
		{input: "", expectedString: "", expectedError: "El camp no pot estar buit"},
		{input: "a", expectedString: "a"},
		{input: " a  ", expectedString: "a"},
	} {
		params := map[string]any{"": tc.input}
		got, errors := ValidateStringNonEmpty(nil, params, "")
		if got != tc.expectedString {
			t.Errorf("[%d] Expected result «%s», got «%s»", i, tc.expectedString, got)
		}
		checkValidationErrors(t, i, tc.expectedError, errors...)
	}
}

func TestValidateDate(t *testing.T) {
	yearRangeError := fmt.Sprintf("L'any ha de ser posterior a 1900 i anterior a %d", time.Now().Year()+5)
	for i, tc := range []ValidationTestCase{
		{input: nil, expectedDate: Date{}, expectedError: "El camp ha de ser una cadena de text"},
		{input: "", expectedDate: Date{}, expectedError: "El camp ha de ser una data completa en format YYYY-mm-dd"},
		{input: "2000", expectedDate: Date{Year: 2000}, expectedError: "El camp ha de ser una data completa en format YYYY-mm-dd"},
		{input: "1899-01-01", expectedDate: Date{1899, 1, 1}, expectedError: yearRangeError},
		{input: "2500-01-01", expectedDate: Date{2500, 1, 1}, expectedError: yearRangeError},
		{input: "2000-00-01", expectedDate: Date{2000, 0, 1}, expectedError: "El mes ha de ser vàlid (rang 1-12)"},
		{input: "2000-13-01", expectedDate: Date{2000, 13, 1}, expectedError: "El mes ha de ser vàlid (rang 1-12)"},
		{input: "2001-02-29", expectedDate: Date{2001, 2, 29}, expectedError: "El camp ha de ser una data vàlida"},
		{input: "2001-01-32", expectedDate: Date{2001, 1, 32}, expectedError: "El camp ha de ser una data vàlida"},
		{input: "2001-04-31", expectedDate: Date{2001, 4, 31}, expectedError: "El camp ha de ser una data vàlida"},
		{input: "2000-02-29", expectedDate: Date{2000, 2, 29}},
		{input: "2000-01-01", expectedDate: Date{2000, 1, 1}},
	} {
		params := map[string]any{"": tc.input}
		got, errors := ValidateDate(nil, params, "")
		if got != tc.expectedDate {
			t.Errorf("[%d] Expected result «%#v», got «%#v»", i, tc.expectedDate, got)
		}
		checkValidationErrors(t, i, tc.expectedError, errors...)
	}
}

func TestValidateSpanishDNI(t *testing.T) {
	for i, tc := range []ValidationTestCase{
		{input: nil, expectedString: "", expectedError: "El camp ha de ser una cadena de text"},
		{input: " a", expectedString: "A", expectedError: "El DNI ha de tindre exactament 8 dígits i una lletra"},
		{input: "aaaaaaaaa", expectedString: "AAAAAAAAA", expectedError: "El primers vuit dígits del DNI han de ser números"},
		{input: "000000001", expectedString: "000000001", expectedError: "La lletra del DNI no és correcta"},
		{input: "00000000A", expectedString: "00000000A", expectedError: "La lletra del DNI no és correcta, o algun dígit no és correcte"},
		{input: "00000000T", expectedString: "00000000T"},
	} {
		params := map[string]any{"": tc.input}
		got, errors := ValidateSpanishDNI(nil, params, "")
		if got != tc.expectedString {
			t.Errorf("[%d] Expected result «%s», got «%s»", i, tc.expectedString, got)
		}
		checkValidationErrors(t, i, tc.expectedError, errors...)
	}
}

func checkValidationErrors(t *testing.T, i int, expected string, errors ...ApiErrors) {
	if expected == "" {
		if errors != nil {
			t.Errorf("[%d] Expected no error, but got %v", i, errors[0][""])
		}
	} else if expected != "" {
		if errors == nil {
			t.Errorf("[%d] Expected error %s, but got no error", i, expected)
		} else {
			errs := errors[0][""]
			if len(errs) != 1 {
				t.Errorf("[%d] Expected one error, but got %d: %v", i, len(errs), errs)
			} else if errs[0] != expected {
				t.Errorf("[%d] Expected «%s», got «%s»", i, expected, errs[0])
			}
		}
	}
}
