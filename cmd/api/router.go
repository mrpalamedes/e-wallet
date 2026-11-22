package main

import (
	"ewallet/internal/user"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newRouter(pool *pgxpool.Pool) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	userRepository := user.NewRepository(pool)
	userService := user.NewUserService(userRepository)
	userHandler := user.NewHandler(userService)

	v1 := r.Group("/api/v1")
	userHandler.RegisterRoutes(v1)

	return r
}
