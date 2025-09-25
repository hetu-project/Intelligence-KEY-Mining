package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS returns a CORS middleware
func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{
		"Origin",
		"Content-Length",
		"Content-Type",
		"Authorization",
		"X-Requested-With",
	}
	config.AllowMethods = []string{
		"GET",
		"POST",
		"PUT",
		"DELETE",
		"OPTIONS",
	}
	return cors.New(config)
}
