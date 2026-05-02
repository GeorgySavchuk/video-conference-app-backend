package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		allowedDomains := []string{
			"hellconf.netlify.app",
			"localhost",
			"127.0.0.1",
		}
		if extra := strings.TrimSpace(os.Getenv("ALLOWED_ORIGIN_SUFFIXES")); extra != "" {
			for _, s := range strings.Split(extra, ",") {
				s = strings.TrimSpace(s)
				if s != "" {
					allowedDomains = append(allowedDomains, s)
				}
			}
		}

		allowedSchemes := []string{
			"http://",
			"https://",
		}

		if gin.Mode() == gin.DebugMode {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			domainAllowed := false
			schemeAllowed := false

			for _, domain := range allowedDomains {
				if strings.HasSuffix(origin, domain) {
					domainAllowed = true
					break
				}
			}

			for _, scheme := range allowedSchemes {
				if strings.HasPrefix(origin, scheme) {
					schemeAllowed = true
					break
				}
			}

			if domainAllowed && schemeAllowed {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			} else if origin == "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With, X-Api-Key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if strings.Contains(c.Request.Header.Get("Upgrade"), "websocket") {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
