package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"order/payment/internal/usecase"
)

type PaymentHandler struct {
	usecase usecase.PaymentUseCase
}

func NewPaymentHandler(r *gin.Engine, uc usecase.PaymentUseCase) {
	handler := &PaymentHandler{usecase: uc}
	paymentGroup := r.Group("/payments")
	{
		paymentGroup.POST("", handler.ProcessPayment)
		paymentGroup.GET("/:order_id", handler.GetPaymentStatus)
	}
}

type ProcessPaymentRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Amount  int64  `json:"amount" binding:"required,gt=0"`
}

func (h *PaymentHandler) ProcessPayment(c *gin.Context) {
	var req ProcessPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := h.usecase.ProcessPayment(c.Request.Context(), req.OrderID, req.Amount)
	if err != nil {
		if err == usecase.ErrPaymentExists {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, payment)
}

func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_id is required"})
		return
	}

	payment, err := h.usecase.GetPaymentByOrderID(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, payment)
}
