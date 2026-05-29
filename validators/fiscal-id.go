package validators

import (
	"strconv"
	"strings"
)

var spanishDNIControlCharacters = []string{
	"T",
	"R",
	"W",
	"A",
	"G",
	"M",
	"Y",
	"F",
	"P",
	"D",
	"X",
	"B",
	"N",
	"J",
	"Z",
	"S",
	"Q",
	"V",
	"H",
	"L",
	"C",
	"K",
	"E",
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

	controlCharacter := value[8:]
	if controlCharacter != spanishDNIControlCharacters[number%23] {
		v.AddError(field, "La lletra del DNI no és correcta, o algun dígit no és correcte")
		return value
	}

	return value
}

func ComputeSpanishDNIControlCharacter(dni string) string {
	number, err := strconv.Atoi(dni)
	if err != nil {
		return ""
	}

	return spanishDNIControlCharacters[number%23]
}
