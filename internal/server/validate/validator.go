package validate

import (
	"errors"
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service_error"
	"git.myservermanager.com/varakh/ecolinker/internal/str"
	"github.com/go-playground/validator/v10"
)

var (
	v = validator.New(validator.WithRequiredStructEnabled())
)

// ValidOrServiceError validates the input struct and returns a serviceError if there are any validation errors
func ValidOrServiceError(i interface{}) error {
	if err := v.Struct(i); err != nil {
		var errs validator.ValidationErrors
		errors.As(err, &errs)

		errorMap := make(map[string]string)
		for _, val := range errs {
			key, txt := ErrorToText(&val)
			errorMap[key] = txt
		}
		return service_error.NewServiceError(service_error.ErrCodeIllegalArgument, fmt.Errorf("validation error: %v (%w)", str.ValuesString(errorMap), err))
	}

	return nil
}

// ValidOrError validates the input struct and returns an err if there are any validation errors
func ValidOrError(i interface{}) error {
	if err := v.Struct(i); err != nil {
		var errs validator.ValidationErrors
		errors.As(err, &errs)

		errorMap := make(map[string]string)
		for _, val := range errs {
			key, txt := ErrorToText(&val)
			errorMap[key] = txt
		}
		return fmt.Errorf("validation error: %v (%w)", str.ValuesString(errorMap), err)
	}

	return nil
}

func ValidatorErrorToMap(err error) map[string]string {
	var errs validator.ValidationErrors

	errorMap := make(map[string]string)

	ok := errors.As(err, &errs)

	if !ok {
		return errorMap
	}

	for _, v := range errs {
		key, txt := ErrorToText(&v)
		errorMap[key] = txt
	}

	return errorMap
}

func ErrorToText(e *validator.FieldError) (string, string) {
	x := *e

	switch x.Tag() {
	case "required":
		return x.Field(), fmt.Sprintf("%s is required", x.Field())
	case "max":
		return x.Field(), fmt.Sprintf("%s cannot be longer than %s", x.Field(), x.Param())
	case "min":
		return x.Field(), fmt.Sprintf("%s must be longer than %s", x.Field(), x.Param())
	case "len":
		return x.Field(), fmt.Sprintf("%s must be %s characters long", x.Field(), x.Param())
	case "uuid4":
		return x.Field(), fmt.Sprintf("%s must a valid uuidv4", x.Field())
	}
	return x.Field(), fmt.Sprintf("%s is not valid", x.Field())
}
