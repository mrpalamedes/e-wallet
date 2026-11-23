package wallet

import (
	"context"
	"errors"
	"ewallet/internal/auth"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type finder interface {
	FindByUserID(ctx context.Context, userID string) (Wallet, error)
}

type Handler struct {
	wallet finder
}

func NewHandler(wallet finder) *Handler {
	return &Handler{wallet: wallet}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/wallet/balance", h.balance)
}

type balanceResponse struct {
	UserID  string `json:"userID"`
	Balance int64  `json:"balance"`
}

func (h *Handler) balance(c *gin.Context) {
	ctx := c.Request.Context()

	userID, ok := auth.UserID(c)
	if !ok {
		slog.ErrorContext(ctx, "wallet balance: route registered without auth middleware")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	wallet, err := h.wallet.FindByUserID(ctx, userID)
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
		return
	case err != nil:
		slog.ErrorContext(ctx, "find wallet", "userID", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, balanceResponse{
		UserID:  wallet.UserID,
		Balance: wallet.Balance,
	})
}
