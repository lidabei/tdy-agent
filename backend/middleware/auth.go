package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/tdy-manager/backend/models"
	"github.com/tdy-manager/backend/services"
	"gorm.io/gorm"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func SignToken(secret string, u models.User, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			return
		}
		tokenStr := strings.TrimPrefix(h, "Bearer ")
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "登录已失效"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != models.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			return
		}
		c.Next()
	}
}

// RequirePerm 要求具备指定权限点之一（需在 RequireAdmin 之后）
func RequirePerm(db *gorm.DB, codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != models.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			return
		}
		uid := CurrentUserID(c)
		perms, ok := c.Get("permissions")
		var list []string
		if ok {
			list, _ = perms.([]string)
		} else {
			var err error
			list, _, err = services.LoadUserPermissions(db, uid)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限校验失败"})
				return
			}
			c.Set("permissions", list)
		}
		set := make(map[string]struct{}, len(list))
		for _, p := range list {
			set[p] = struct{}{}
		}
		for _, code := range codes {
			if _, hit := set[code]; hit {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "无操作权限"})
	}
}

func CurrentUserID(c *gin.Context) uint {
	v, _ := c.Get("user_id")
	id, _ := v.(uint)
	return id
}

func RequireEnabled(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var u models.User
		if err := db.Select("id", "enabled").First(&u, CurrentUserID(c)).Error; err != nil || !u.Enabled {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "账号已禁用"})
			return
		}
		c.Next()
	}
}
