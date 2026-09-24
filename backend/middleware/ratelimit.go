package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type loginLimiter struct {
	mu     sync.Mutex
	fails  map[string][]time.Time
	limit  int
	window time.Duration
}

var globalLoginLimiter = &loginLimiter{
	fails:  map[string][]time.Time{},
	limit:  20,
	window: 15 * time.Minute,
}

// LoginRateLimit 仅统计失败次数；成功登录会清零。
func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !globalLoginLimiter.canAttempt(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "登录尝试过多，请稍后再试（或重启后端清空限制）",
			})
			return
		}
		c.Next()
		status := c.Writer.Status()
		if status == http.StatusUnauthorized || status == http.StatusForbidden {
			globalLoginLimiter.recordFail(ip)
			return
		}
		if status == http.StatusOK {
			globalLoginLimiter.clear(ip)
		}
	}
}

func ClearLoginLimit(ip string) {
	globalLoginLimiter.clear(ip)
}

func (l *loginLimiter) canAttempt(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(key)
	return len(l.fails[key]) < l.limit
}

func (l *loginLimiter) recordFail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(key)
	l.fails[key] = append(l.fails[key], time.Now())
}

func (l *loginLimiter) clear(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fails, key)
}

func (l *loginLimiter) pruneLocked(key string) {
	now := time.Now()
	cut := now.Add(-l.window)
	arr := l.fails[key]
	kept := arr[:0]
	for _, t := range arr {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(l.fails, key)
		return
	}
	l.fails[key] = kept
}
