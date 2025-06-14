package handler

import (
	"errors"
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/api"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service_error"
	"git.myservermanager.com/varakh/ecolinker/internal/server/validate"
	"git.myservermanager.com/varakh/ecolinker/internal/str"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"net/http"
)

func AbortWithValidatorPayload(c *gin.Context, err error) {
	var errs validator.ValidationErrors
	errors.As(err, &errs)

	errorMap := make(map[string]string)
	for _, v := range errs {
		key, txt := validate.ErrorToText(&v)
		errorMap[key] = txt
	}

	resErr := service_error.NewServiceError(service_error.ErrCodeIllegalArgument, fmt.Errorf("validation error: %v (%w)", str.ValuesString(errorMap), err))
	c.Header(api.HeaderContentType, api.HeaderContentTypeApplicationJson)
	_ = c.AbortWithError(http.StatusBadRequest, resErr)
	return
}

func ToHttpStatus(err error) int {
	var e *service_error.ServiceError
	switch {
	case errors.As(err, &e):
		if e.Status == service_error.ErrCodeIllegalArgument {
			return http.StatusBadRequest
		} else if e.Status == service_error.ErrCodeUnauthorized {
			return http.StatusUnauthorized
		} else if e.Status == service_error.ErrCodeForbidden {
			return http.StatusForbidden
		} else if e.Status == service_error.ErrCodeNotFound {
			return http.StatusNotFound
		} else if e.Status == service_error.ErrCodeMethodNotAllowed {
			return http.StatusMethodNotAllowed
		} else if e.Status == service_error.ErrCodeConflict {
			return http.StatusConflict
		} else if e.Status == service_error.ErrCodeGeneral {
			return http.StatusInternalServerError
		}
	default:
		return http.StatusInternalServerError
	}

	return -1
}

func CodeToStr(err error) string {
	var e *service_error.ServiceError
	ok := errors.As(err, &e)

	if ok {
		return string(e.Status)
	}

	return string(service_error.ErrCodeGeneral)
}
