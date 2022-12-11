package validator

type FieldError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

type FieldErrors []FieldError

type Validator struct {
	Errors      []string    `json:"errors,omitempty"`
	FieldErrors FieldErrors `json:"field_errors,omitempty"`
}

func (v *Validator) HasErrors() bool {
	return len(v.Errors) != 0 || len(v.FieldErrors) != 0
}

func (v *Validator) AddError(message string) {
	if v.Errors == nil {
		v.Errors = []string{}
	}

	v.Errors = append(v.Errors, message)
}

func (v *Validator) AddFieldError(field, message string) {
	if v.FieldErrors == nil {
		v.FieldErrors = []FieldError{}
	}

	if exists := v.FieldErrors.Contains(field); !exists {
		v.FieldErrors = append(v.FieldErrors, FieldError{
			Field: field,
			Error: message,
		})
	}
}

func (v *Validator) Check(ok bool, message string) {
	if !ok {
		v.AddError(message)
	}
}

func (v *Validator) CheckField(ok bool, key, message string) {
	if !ok {
		v.AddFieldError(key, message)
	}
}

func (fe FieldErrors) Contains(field string) bool {
	for _, f := range fe {
		if f.Field == field {
			return true
		}
	}
	return false
}
