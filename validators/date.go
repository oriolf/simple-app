package validators

import (
	"fmt"
	"strings"
	"time"

	app "github.com/oriolf/simple-app"
	"github.com/oriolf/simple-app/types"
)

func (v *Validator) ValidateDate(field string) types.Date {
	s := v.validateStringPresent(field)
	if v.HasError(field) {
		return types.Date{}
	}

	var value types.Date
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
	if app.InSlice(uint(value.Month), []uint{4, 6, 9, 11}) {
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
