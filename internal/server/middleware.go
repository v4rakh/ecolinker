package server

import (
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/api"
	"git.myservermanager.com/varakh/ecolinker/internal/app"
	"git.myservermanager.com/varakh/ecolinker/internal/server/config"
	"git.myservermanager.com/varakh/ecolinker/internal/server/handler"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service_error"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
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

// middlewareAppName adds custom HTTP header to each request
func middlewareAppName() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header(api.HeaderAppName, app.Name)
		c.Next()
	}
}

// middlewareAppVersion adds custom HTTP header to each request
func middlewareAppVersion() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header(api.HeaderAppVersion, app.Version)
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
		if c.Request.Method != http.MethodOptions && !strings.HasPrefix(c.GetHeader(api.HeaderContentType), api.HeaderContentTypeApplicationJson) {
			c.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponseWithStatusAndMessage(string(service_error.ErrCodeIllegalArgument), "content-type must be application/json"))
			return
		}
		c.Next()
	}
}

// middlewareErrorHandler handles global error handling, does not overwrite any given status (see -1)
func middlewareErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// call next first, so this is the last in chain
		c.Next()

		if len(c.Errors) > 0 {
			// status -1 doesn't overwrite existing status code
			c.Header(api.HeaderContentType, api.HeaderContentTypeApplicationJson)
			c.JSON(-1, api.NewErrorResponseWithStatusAndMessage(handler.CodeToStr(c.Errors.Last()), c.Errors.Last().Error()))
			return
		}
	}
}

// middlewareErrorRecoveryHandler recovers from panics, returning a 500 error
func middlewareErrorRecoveryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, api.NewErrorResponseWithStatusAndMessage(string(service_error.ErrCodeGeneral), fmt.Sprintf("%s", err)))
			}
		}()
		c.Next()
	}
}
