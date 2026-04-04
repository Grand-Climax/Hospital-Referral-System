package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter uses Redis to limit requests from a single IP to a specific endpoint.
func RateLimiter(client *redis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		path := c.FullPath()
		key := fmt.Sprintf("ratelimit:%s:%s", path, ip)

		// Increment the counter for this IP/Path combo
		count, err := client.Incr(c.Request.Context(), key).Result()
		if err != nil {
			// Fail open if Redis is down
			c.Next()
			return
		}

		// If it's the first request, set the expiration window
		if count == 1 {
			client.Expire(c.Request.Context(), key, window)
		}

		// Reject if max requests exceeded
		if count > int64(maxRequests) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "Too many requests. Please try again later.",
			})
			return
		}

		c.Next()
	}
}
