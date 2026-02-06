package validation

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate

	unsafePattern = regexp.MustCompile(`[<>{}\\]|javascript:|on\w+=|data:|&lt;|&gt;|&#`)
)

func init() {
	validate = validator.New()
	validate.RegisterValidation("safe_content", validateSafeContent)
}

func ValidateRequest(s interface{}) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	errorsMap := make(map[string]string)
	for _, err := range err.(validator.ValidationErrors) {
		if err.Tag() == "safe_content" {
			errorsMap[err.Field()] = fmt.Sprintf("Field '%s' contains potentially unsafe content.",
				err.Field(),
			)
		} else {
			errorsMap[err.Field()] = fmt.Sprintf("Field '%s' failed validation: tag '%s'.",
				err.Field(),
				err.Tag(),
			)
		}
	}

	return errorsMap
}

func ValidateID(id string) (string, error) {
	if err := validate.Var(id, "required,uuid"); err != nil {
		return "", errors.New("ID must be a valid UUID")
	}
	return id, nil
}

func validateSafeContent(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return IsSafeContent(value)
}

func IsSafeContent(s string) bool {
	return !unsafePattern.MatchString(s)
}
