package user

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type userService interface {
	Register(ctx context.Context, input RegisterInput) (User, error)
}

type Handler struct {
	userService userService
}

func NewHandler(userService userService) *Handler {
	return &Handler{userService: userService}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/users/register", h.register)
}

type registerRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type registerResponse struct {
	UserID  string `json:"user_id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

func (h *Handler) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "name, valid email, and password (min. 8 characters) are required",
		})
		return
	}

	u, err := h.userService.Register(c.Request.Context(), RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	switch {
	case errors.Is(err, ErrEmailTaken):
		c.JSON(http.StatusConflict, gin.H{"error": ErrEmailTaken.Error()})
		return
	case errors.Is(err, ErrPasswordTooLong):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrPasswordTooLong.Error()})
		return
	case err != nil:
		slog.ErrorContext(c.Request.Context(), "register user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, registerResponse{
		UserID:  u.ID,
		Name:    u.Name,
		Email:   u.Email,
		Message: "User registered successfully",
	})
}
