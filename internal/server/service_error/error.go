package service_error

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrValidationNotEmpty           = NewServiceError(ErrCodeIllegalArgument, errors.New("assert: empty values are not allowed"))
	ErrValidationNotBlank           = NewServiceError(ErrCodeIllegalArgument, errors.New("assert: blank values are not allowed"))
	ErrValidationTimeFormatDateTime = NewServiceError(ErrCodeIllegalArgument, errors.New(fmt.Sprintf("assert: not a valid time, expecting format '%s'", time.DateTime)))
	ErrResourceNotFound             = NewServiceError(ErrCodeNotFound, errors.New("resource not found"))
	ErrResourceConflict             = NewServiceError(ErrCodeConflict, errors.New("resource already exists"))
	ErrDatabaseRowsExpected         = NewServiceDatabaseError(errors.New("action failed, expected affected rows, but got none"))
)

type ErrorCode string

const (
	ErrCodeIllegalArgument  ErrorCode = "IllegalArgument"
	ErrCodeUnauthorized     ErrorCode = "Unauthorized"
	ErrCodeForbidden        ErrorCode = "Forbidden"
	ErrCodeNotFound         ErrorCode = "NotFound"
	ErrCodeMethodNotAllowed ErrorCode = "MethodNotAllowed"
	ErrCodeConflict         ErrorCode = "Conflict"
	ErrCodeGeneral          ErrorCode = "General"
)

// NewServiceError returns an error that formats as the given text and aligns with builtin error
func NewServiceError(status ErrorCode, err error) error {
	return &ServiceError{status, fmt.Errorf("service error (%v): %w", status, err)}
}

// NewServiceDatabaseError returns an error that formats as the given text and aligns with builtin error
func NewServiceDatabaseError(error error) error {
	return NewServiceError(ErrCodeGeneral, fmt.Errorf("database error: %w", error))
}

type ServiceError struct {
	Status ErrorCode
	Cause  error
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%v", e.Cause)
}
