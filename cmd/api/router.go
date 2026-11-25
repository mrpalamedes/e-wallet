package main

import (
	"ewallet/internal/auth"
	"ewallet/internal/config"
	"ewallet/internal/transaction"
	"ewallet/internal/user"
	"ewallet/internal/wallet"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newRouter(cfg config.Config, pool *pgxpool.Pool) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	token := auth.NewJWT(cfg.JWTSecret, cfg.JWTTTL)

	userRepository := user.NewRepository(pool)
	userService := user.NewService(userRepository, token)
	userHandler := user.NewHandler(userService)

	walletRepo := wallet.NewRepository(pool)
	walletHandler := wallet.NewHandler(walletRepo)

	txRepo := transaction.NewRepository(pool)
	txService := transaction.NewService(txRepo)
	txHandler := transaction.NewHandler(txService)

	v1 := r.Group("/api/v1")
	userHandler.RegisterRoutes(v1)

	protected := v1.Group("")
	protected.Use(auth.Middleware(token))
	walletHandler.RegisterRoutes(protected)
	txHandler.RegisterRoutes(protected)

	return r
}
