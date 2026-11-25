package payment

import (
	"context"
	"errors"
	"ewallet/internal/auth"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type service interface {
	Create(ctx context.Context, userID string, in CreateInput) (Session, error)
	Confirm(ctx context.Context, userID, paymentID string) (ConfirmResult, error)
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/transactions/pay", h.create)
	rg.POST("/transactions/pay/:id/confirm", h.confirm)
}

type createRequest struct {
	MerchantID  string `json:"merchantID" binding:"required,max=100"`
	Amount      int64  `json:"amount" binding:"required,min=1"`
	Description string `json:"description" binding:"max=255"`
}

type createResponse struct {
	TransactionID string    `json:"transactionID"`
	Status        Status    `json:"status"`
	MerchantID    string    `json:"merchantID"`
	Amount        int64     `json:"amount"`
	ExpiresAt     time.Time `json:"expiresAt"`
	Message       string    `json:"message"`
}

func (h *Handler) create(c *gin.Context) {
	userID, ok := auth.UserID(c)
	if !ok {
		respondInternalError(c, "create payment: route registered without auth middleware", nil)
		return
	}

	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "merchantID is required and amount must be a whole number of at least 1",
		})
		return
	}

	sess, err := h.svc.Create(c.Request.Context(), userID, CreateInput{
		MerchantID:  req.MerchantID,
		Amount:      req.Amount,
		Description: req.Description,
	})
	switch {
	case errors.Is(err, ErrInvalidAmount):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidAmount.Error()})
		return
	case errors.Is(err, ErrInvalidMerchant):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidMerchant.Error()})
		return
	case err != nil:
		respondInternalError(c, "create payment", err)
		return
	}

	c.JSON(http.StatusCreated, createResponse{
		TransactionID: sess.ID,
		Status:        sess.Status,
		MerchantID:    sess.MerchantID,
		Amount:        sess.Amount,
		ExpiresAt:     sess.ExpiresAt.UTC(),
		Message:       "Payment session created. Confirm before it expires.",
	})
}

type confirmURI struct {
	ID string `uri:"id" binding:"required,uuid"`
}

type confirmResponse struct {
	TransactionID string `json:"transactionID"`
	Message       string `json:"message"`
	NewBalance    int64  `json:"newBalance"`
}

func (h *Handler) confirm(c *gin.Context) {
	userID, ok := auth.UserID(c)
	if !ok {
		respondInternalError(c, "confirm payment: route registered without auth middleware", nil)
		return
	}

	var uri confirmURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment id"})
		return
	}

	res, err := h.svc.Confirm(c.Request.Context(), userID, uri.ID)
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
		return
	case errors.Is(err, ErrExpired):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrExpired.Error()})
		return
	case errors.Is(err, ErrNotPending):
		c.JSON(http.StatusConflict, gin.H{"error": ErrNotPending.Error()})
		return
	case errors.Is(err, ErrInsufficientFunds):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInsufficientFunds.Error()})
		return
	case errors.Is(err, ErrWalletNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": ErrWalletNotFound.Error()})
		return
	case err != nil:
		respondInternalError(c, "confirm payment", err)
		return
	}

	c.JSON(http.StatusOK, confirmResponse{
		TransactionID: res.TransactionID,
		Message:       "Payment successful",
		NewBalance:    res.NewBalance,
	})
}

func respondInternalError(c *gin.Context, msg string, err error) {
	slog.ErrorContext(c.Request.Context(), msg, "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
