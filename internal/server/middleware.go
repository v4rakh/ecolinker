package server

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"git.myservermanager.com/varakh/ecolinker/internal/api"
	httpcommons "git.myservermanager.com/varakh/ecolinker/internal/http"
	"git.myservermanager.com/varakh/ecolinker/internal/meta"
	"git.myservermanager.com/varakh/ecolinker/internal/server/config"
	"git.myservermanager.com/varakh/ecolinker/internal/service_error"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	headerAppName    = "X-App-Name"
	headerAppVersion = "X-App-Version"
)

// middlewareCors applies CORS configuration
func middlewareCors(c *config.Cors) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     c.AllowOrigins,
		AllowMethods:     c.AllowMethods,
		AllowHeaders:     c.AllowHeaders,
		AllowCredentials: c.AllowCredentials,
		ExposeHeaders:    c.ExposeHeaders,
	})
}

// middlewareLogging logs access
func middlewareLogging(lc *config.Logging) gin.HandlerFunc {
	var err error
	var logLevel zerolog.Level
	if logLevel, err = zerolog.ParseLevel(lc.LevelRequests); err != nil {
		logLevel = zerolog.Disabled
	}
	return func(c *gin.Context) {
		c.Next()
		log.WithLevel(logLevel).Msgf("Handled request %s %s: %d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
	}
}

// middlewarePanicRecoveryHandler recovers meta from panics, logs them and returns proper response
// logs the error and stack trace using zerolog.Logger, and returns a 500 response.
func middlewarePanicRecoveryHandler(lc *config.Logging) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stackTrace := debug.Stack()
				panicErr := fmt.Errorf("%v", err)

				log.Ctx(c.Request.Context()).
					Error().
					Err(panicErr).
					Str(lc.EncodingStacktraceKey, string(stackTrace)).
					Msg("panic")

				c.Header(httpcommons.HeaderContentType, httpcommons.HeaderContentTypeApplicationJson)

				// this is a panic, thus treat this carefully as it may contain sensitive information
				c.AbortWithStatusJSON(http.StatusInternalServerError, api.NewErrorResponseWithStatusAndMessage(service_error.ErrCodeGeneral.String(), "panic"))
			}
		}()

		c.Next()
	}
}

// middlewareAppName adds custom HTTP header to each request
func middlewareAppName() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header(headerAppName, meta.Name)
		c.Next()
	}
}

// middlewareAppVersion adds custom HTTP header to each request
func middlewareAppVersion() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header(headerAppVersion, meta.Version)
		c.Next()
	}
}

// middlewareGlobalNotFound adds a global not found in the same style as the API responds
func middlewareGlobalNotFound() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusNotFound, api.NewErrorResponseWithStatusAndMessage(service_error.ErrCodeNotFound.String(), "page not found"))
	}
}

// middlewareGlobalMethodNotAllowed adds a global method not allowed in the same style as the API responds
func middlewareGlobalMethodNotAllowed() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusMethodNotAllowed, api.NewErrorResponseWithStatusAndMessage(service_error.ErrCodeMethodNotAllowed.String(), "method not allowed"))
	}
}

// middlewareEnforceJsonContentType enforces JSON content type
func middlewareEnforceJsonContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodOptions && !strings.HasPrefix(c.GetHeader(httpcommons.HeaderContentType), httpcommons.HeaderContentTypeApplicationJson) {
			c.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponseWithStatusAndMessage(service_error.ErrCodeIllegalArgument.String(), "content-type must be application/json"))
			return
		}
		c.Next()
	}
}

// middlewareErrorTransformer transforms errors into proper responses (does not overwrite any given status)
func middlewareErrorTransformer() gin.HandlerFunc {
	return func(c *gin.Context) {
		// call next first, so this is the last in chain
		c.Next()

		if len(c.Errors) > 0 {
			// status -1 doesn't overwrite existing status code
			c.Header(httpcommons.HeaderContentType, httpcommons.HeaderContentTypeApplicationJson)

			// unknown/general errors can contain sensitive information, thus treat carefully
			if svcErr, ok := errors.AsType[*service_error.ServiceError](c.Errors.Last().Err); ok {
				errorBody := svcErr.Error()
				if svcErr.Status == service_error.ErrCodeGeneral {
					errorBody = svcErr.Status.String()

					log.Ctx(c.Request.Context()).
						Error().
						Err(svcErr).
						Msg("service error")
				}

				c.JSON(-1, api.NewErrorResponseWithStatusAndMessage(svcErr.Status.String(), errorBody))
				return
			}

			// this is an uncaught error, usually everything returns a service_error, thus treat this carefully as it may contain sensitive information
			log.Ctx(c.Request.Context()).
				Error().
				Err(c.Errors.Last().Err).
				Msg("internal server error")

			c.JSON(-1, api.NewErrorResponseWithStatusAndMessage(service_error.ErrCodeGeneral.String(), "internal server error"))
			return
		}
	}
}
