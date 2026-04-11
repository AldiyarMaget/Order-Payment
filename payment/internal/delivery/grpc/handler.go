package grpc_delivery

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"order/payment/internal/domain"
	"order/payment/internal/usecase"

	contract "github.com/AldiyarMaget/aitu-go-sdk"
)

type PaymentHandler struct {
	contract.UnimplementedPaymentServiceServer
	useCase usecase.PaymentUseCase
}

func NewPaymentHandler(uc usecase.PaymentUseCase) *PaymentHandler {
	return &PaymentHandler{
		useCase: uc,
	}
}

func (h *PaymentHandler) ProcessPayment(ctx context.Context, req *contract.PaymentRequest) (*contract.PaymentResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than zero")
	}

	payment, err := h.useCase.ProcessPayment(ctx, req.GetOrderId(), req.GetAmount())
	if err != nil {
		if err == usecase.ErrPaymentExists {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to process payment: %v", err)
	}

	var pbStatus contract.PaymentStatus
	switch payment.Status {
	case domain.PaymentStatusAuthorized:
		pbStatus = contract.PaymentStatus_PAYMENT_STATUS_AUTHORIZED
	case domain.PaymentStatusDeclined:
		pbStatus = contract.PaymentStatus_PAYMENT_STATUS_DECLINED
	default:
		pbStatus = contract.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}

	return &contract.PaymentResponse{
		Id:            payment.ID,
		OrderId:       payment.OrderID,
		TransactionId: payment.TransactionID,
		Amount:        payment.Amount,
		Status:        pbStatus,
	}, nil
}
