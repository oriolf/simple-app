package app

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
