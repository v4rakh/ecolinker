package server

import (
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/internal/api"
	httpcommons "git.myservermanager.com/varakh/ecolinker/internal/http"
	"git.myservermanager.com/varakh/ecolinker/internal/meta"
	"git.myservermanager.com/varakh/ecolinker/internal/server/config"
	"git.myservermanager.com/varakh/ecolinker/internal/server/handler"
	"git.myservermanager.com/varakh/ecolinker/internal/service_error"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"runtime/debug"
	"strings"
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
				log.Error().Str(lc.EncodingStacktraceKey, string(debug.Stack())).Msgf("panic recovered: %v", err)
				c.Header(httpcommons.HeaderContentType, httpcommons.HeaderContentTypeApplicationJson)
				c.AbortWithStatusJSON(http.StatusInternalServerError, api.NewErrorResponseWithStatusAndMessage(string(service_error.ErrCodeGeneral), fmt.Sprintf("%s", err)))
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
		c.AbortWithStatusJSON(http.StatusNotFound, api.NewErrorResponseWithStatusAndMessage(string(service_error.ErrCodeNotFound), "page not found"))
		return
	}
}

// middlewareGlobalMethodNotAllowed adds a global method not allowed in the same style as the API responds
func middlewareGlobalMethodNotAllowed() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusMethodNotAllowed, api.NewErrorResponseWithStatusAndMessage(string(service_error.ErrCodeMethodNotAllowed), "method not allowed"))
		return
	}
}

// middlewareEnforceJsonContentType enforces JSON content type
func middlewareEnforceJsonContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodOptions && !strings.HasPrefix(c.GetHeader(httpcommons.HeaderContentType), httpcommons.HeaderContentTypeApplicationJson) {
			c.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponseWithStatusAndMessage(string(service_error.ErrCodeIllegalArgument), "content-type must be application/json"))
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
			c.JSON(-1, api.NewErrorResponseWithStatusAndMessage(handler.CodeToStr(c.Errors.Last()), c.Errors.Last().Error()))
			return
		}
	}
}
