package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/calm195/go-template/internal/user"
)

// Readiness 检查依赖是否具备处理请求的条件。
type Readiness func(context.Context) error

// New 构建包含日志、异常恢复和健康检查端点的 HTTP 路由。
func New(log *zap.Logger, ready Readiness, users *user.Service) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.Use(func(c *gin.Context) {
		start := time.Now()
		var id [16]byte
		if _, err := rand.Read(id[:]); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		requestID := hex.EncodeToString(id[:])
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
		// 外层中间件在异常恢复和处理函数结束后记录最终状态。
		c.Next()
		log.Info("http request completed",
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", time.Since(start)),
		)
	}, func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				// panic 值可能包含敏感信息，因此只保留请求关联信息，不记录其内容。
				log.Error("request panic", zap.String("request_id", c.GetString("request_id")))
				c.AbortWithStatusJSON(500, gin.H{"error": "internal server error"})
			}
		}()
		c.Next()
	})
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := ready(ctx); err != nil {
			c.JSON(503, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ready"})
	})
	r.POST("/api/v1/users", func(c *gin.Context) {
		if users == nil {
			c.JSON(503, gin.H{"error": "database disabled"})
			return
		}
		var input struct {
			Name string `json:"name" binding:"required,max=100"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Name) == "" {
			c.JSON(400, gin.H{"error": "name must contain 1-100 characters"})
			return
		}
		u, err := users.Create(c.Request.Context(), strings.TrimSpace(input.Name))
		if err != nil {
			log.Error("create user failed", zap.String("request_id", c.GetString("request_id")))
			c.JSON(500, gin.H{"error": "internal server error"})
			return
		}
		// 仅在持久化成功后记录成功日志，且不记录用户输入。
		log.Info("create user completed",
			zap.String("request_id", c.GetString("request_id")),
			zap.Uint64("user_id", u.ID),
		)
		c.JSON(201, u)
	})
	return r
}
