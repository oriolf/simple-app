package app

import "testing"

type ValidationTestCase struct {
	s        string
	expected string
}

func TestValidateStringNonEmpty(t *testing.T) {
	for i, tc := range []ValidationTestCase{
		{s: "", expected: "El camp no pot estar buit"},
		{s: "a"},
	} {
		errors := ValidateStringNonEmpty("", tc.s)
		checkValidationResult(t, i, tc.expected, errors)
	}
}

// TODO right now, if JSON decode fails, we miss all information about which
// field fails, and the opportunity to return adequate error messages;
// validation should begin earlier, so that for example if we try to unmarshal
// "" to the field "joined_on" of type Date we can say that we expect a valid
// date in format "YYYY-mm-dd"
// UnmarshalTypeError from "encoding/json" would be great for that, but we
// don't seem to be getting it
func TestValidateDate(t *testing.T) {
	for i, tc := range []ValidationTestCase{
		{s: "", expected: "EOF"},
		{s: "2000", expected: "unexpected EOF"},
		{s: "2000-01-01"},
	} {
		var d Date
		if err := d.FromString(tc.s); err != nil {
			if tc.expected == "" {
				t.Errorf("[%d] Expected valid date, but found: %s", i, err)
			} else if tc.expected != err.Error() {
				t.Errorf("[%d] Expected error «%s», got error «%s».", i, tc.expected, err)
			}
		} else {
			errors := ValidateDate("", d)
			checkValidationResult(t, i, tc.expected, errors)
		}
	}
}

func checkValidationResult(t *testing.T, i int, expected string, errors ApiErrors) {
	if expected == "" {
		if errors != nil {
			t.Errorf("[%d] Expected no error, but got %v", i, errors[""])
		}
	} else if expected != "" {
		if errors == nil {
			t.Errorf("[%d] Expected error %s, but got no error", i, expected)
		} else {
			errs := errors[""]
			if len(errs) != 1 {
				t.Errorf("[%d] Expected one error, but got %d: %v", i, len(errs), errs)
			} else if errs[0] != expected {
				t.Errorf("[%d] Expected «%s», got «%s»", i, expected, errs[0])
			}
		}
	}
}
