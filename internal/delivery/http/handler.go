package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"order/internal/usecase"
)

type OrderHandler struct {
	useCase usecase.OrderUseCase
}

func NewOrderHandler(router *gin.Engine, uc usecase.OrderUseCase) {
	handler := &OrderHandler{
		useCase: uc,
	}

	router.POST("/orders", handler.CreateOrder)
	router.PATCH("/orders/:id/cancel", handler.CancelOrder)
	router.GET("/orders/:id", handler.GetOrder)
	router.GET("/orders", handler.GetOrders)
}

type createOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name" binding:"required"`
	Amount     int64  `json:"amount" binding:"required,gt=0"`
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Idempotency-Key header is required"})
		return
	}

	order, err := h.useCase.CreateOrder(c.Request.Context(), req.CustomerID, req.ItemName, req.Amount, idempotencyKey)
	if err != nil {
		if errors.Is(err, usecase.ErrPaymentUnavailable) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment service down"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")

	err := h.useCase.CancelOrder(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "cannot cancel a paid order" || err.Error() == "invalid state transition" || err.Error() == "order not found" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Cancelled"})
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.useCase.GetOrder(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, usecase.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) GetOrders(c *gin.Context) {
	minStr := c.Query("min_amount")
	maxStr := c.Query("max_amount")

	if minStr == "" || maxStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "min_amount and max_amount are required"})
		return
	}

	minAmount, err := strconv.ParseInt(minStr, 10, 64)
	if err != nil || minAmount < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid min_amount"})
		return
	}

	maxAmount, err := strconv.ParseInt(maxStr, 10, 64)
	if err != nil || maxAmount < minAmount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid max_amount"})
		return
	}

	orders, err := h.useCase.GetByAmountRange(c.Request.Context(), minAmount, maxAmount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}
