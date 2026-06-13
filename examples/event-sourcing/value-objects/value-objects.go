package valueobjects

import (
	"github.com/oriolf/simple-app/types"
	"github.com/oriolf/simple-app/validators"
)

type Name string

func NewName(validator validators.Validator, field string) Name {
	s := validator.ValidateStringNonEmpty(field)
	return Name(s)
}

type NIF string

func NewNIF(validator validators.Validator, field string) NIF {
	s := validator.ValidateSpanishDNI(field)
	return NIF(s)
}

func NewDate(validator validators.Validator, field string) types.Date {
	return validator.ValidateDate(field)
}
