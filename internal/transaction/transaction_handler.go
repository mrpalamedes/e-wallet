package transaction

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
	TopUp(ctx context.Context, userID string, amount int64) (TopUpResult, error)
	Transfer(ctx context.Context, fromID, toID string, amount int64) (string, error)
	History(ctx context.Context, userID string, page, limit int) (Page, error)
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/transactions/topup", h.topUp)
	rg.POST("/transactions/transfer", h.transfer)
	rg.GET("/transactions/history", h.history)
}

type topUpRequest struct {
	Amount int64 `json:"amount" binding:"required,min=1"`
}

type topUpResponse struct {
	TransactionID string `json:"transactionID"`
	Message       string `json:"message"`
	NewBalance    int64  `json:"newBalance"`
}

func (h *Handler) topUp(c *gin.Context) {
	userID, ok := auth.UserID(c)
	if !ok {
		respondInternalError(c, "top up: route registered without auth middleware", nil)
		return
	}

	var req topUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be a whole number of at least 1"})
		return
	}

	res, err := h.svc.TopUp(c.Request.Context(), userID, req.Amount)
	switch {
	case errors.Is(err, ErrInvalidAmount):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidAmount.Error()})
		return
	case errors.Is(err, ErrWalletNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": ErrWalletNotFound.Error()})
		return
	case err != nil:
		respondInternalError(c, "top up", err)
		return
	}

	c.JSON(http.StatusOK, topUpResponse{
		TransactionID: res.TransactionID,
		Message:       "Top-up successful",
		NewBalance:    res.NewBalance,
	})
}

type transferRequest struct {
	ToUserID string `json:"toUserID" binding:"required,uuid"`
	Amount   int64  `json:"amount" binding:"required,min=1"`
}

type transferResponse struct {
	TransactionID string `json:"transactionID"`
	Status        Status `json:"status"`
	Message       string `json:"message"`
}

func (h *Handler) transfer(c *gin.Context) {
	userID, ok := auth.UserID(c)
	if !ok {
		respondInternalError(c, "transfer: route registered without auth middleware", nil)
		return
	}

	var req transferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "toUserID must be a valid UUID and amount a whole number of at least 1",
		})
		return
	}

	transactionID, err := h.svc.Transfer(c.Request.Context(), userID, req.ToUserID, req.Amount)
	switch {
	case errors.Is(err, ErrInvalidAmount):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidAmount.Error()})
		return
	case errors.Is(err, ErrSelfTransfer):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrSelfTransfer.Error()})
		return
	case errors.Is(err, ErrInsufficientFunds):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInsufficientFunds.Error()})
		return
	case errors.Is(err, ErrRecipientNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": ErrRecipientNotFound.Error()})
		return
	case errors.Is(err, ErrWalletNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": ErrWalletNotFound.Error()})
		return
	case err != nil:
		respondInternalError(c, "transfer", err)
		return
	}

	c.JSON(http.StatusAccepted, transferResponse{
		TransactionID: transactionID,
		Status:        StatusCompleted,
		Message:       "Transfer completed successfully.",
	})
}

type historyQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

type historyItem struct {
	TransactionID string    `json:"transactionID"`
	Type          Type      `json:"type"`
	Amount        int64     `json:"amount"`
	From          string    `json:"from"`
	To            string    `json:"to"`
	Timestamp     time.Time `json:"timestamp"`
	Status        Status    `json:"status"`
}

type pagination struct {
	CurrentPage int `json:"currentPage"`
	TotalPages  int `json:"totalPages"`
	TotalItems  int `json:"totalItems"`
}

type historyResponse struct {
	Transactions []historyItem `json:"transactions"`
	Pagination   pagination    `json:"pagination"`
}

func (h *Handler) history(c *gin.Context) {
	userID, ok := auth.UserID(c)
	if !ok {
		respondInternalError(c, "history: route registered without auth middleware", nil)
		return
	}

	var q historyQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page must be >= 1 and limit must be between 1 and 100"})
		return
	}

	p, err := h.svc.History(c.Request.Context(), userID, q.Page, q.Limit)
	if err != nil {
		respondInternalError(c, "history", err)
		return
	}

	items := make([]historyItem, 0, len(p.Records))
	for _, rec := range p.Records {
		items = append(items, historyItem{
			TransactionID: rec.ID,
			Type:          rec.Type,
			Amount:        rec.Amount,
			From:          rec.From,
			To:            rec.To,
			Timestamp:     rec.CreatedAt.UTC(),
			Status:        rec.Status,
		})
	}

	c.JSON(http.StatusOK, historyResponse{
		Transactions: items,
		Pagination: pagination{
			CurrentPage: p.CurrentPage,
			TotalPages:  p.TotalPages,
			TotalItems:  p.TotalItems,
		},
	})
}

func respondInternalError(c *gin.Context, msg string, err error) {
	slog.ErrorContext(c.Request.Context(), msg, "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
