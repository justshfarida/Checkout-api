package validation

import "fmt"

type FieldErrors map[string]string

type Result struct {
	Errors FieldErrors
}

func NewResult() *Result {
	return &Result{Errors: FieldErrors{}}
}

func (r *Result) Valid() bool {
	return len(r.Errors) == 0
}

func (r *Result) add(field, msg string) {
	r.Errors[field] = msg
}

func (r *Result) Required(field, value string) {
	if value == "" {
		r.add(field, "is required")
	}
}

func (r *Result) MinLen(field, value string, min int) {
	if len(value) < min {
		r.add(field, fmt.Sprintf("must be at least %d characters", min))
	}
}

func (r *Result) MaxLen(field, value string, max int) {
	if len(value) > max {
		r.add(field, fmt.Sprintf("must be at most %d characters", max))
	}
}

func (r *Result) Positive(field string, value int) {
	if value <= 0 {
		r.add(field, "must be greater than 0")
	}
}

func (r *Result) Min(field string, value, min int) {
	if value < min {
		r.add(field, fmt.Sprintf("must be at least %d", min))
	}
}

func (r *Result) Max(field string, value, max int) {
	if value > max {
		r.add(field, fmt.Sprintf("must be at most %d", max))
	}
}

func (r *Result) OneOf(field, value string, allowed ...string) {
	if value == "" {
		return // let Required handle emptiness separately
	}
	for _, a := range allowed {
		if value == a {
			return
		}
	}
	r.add(field, fmt.Sprintf("must be one of: %v", allowed))
}
