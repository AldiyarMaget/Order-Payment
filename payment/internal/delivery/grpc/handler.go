package grpc_delivery

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"order/payment/internal/domain"
	"order/payment/internal/usecase"

	contract "github.com/AldiyarMaget/aitu-go-sdk/contract"
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

func (h *PaymentHandler) mapStatusToPb(s domain.PaymentStatus) contract.PaymentStatus {
	switch s {
	case domain.PaymentStatusAuthorized:
		return contract.PaymentStatus_PAYMENT_STATUS_AUTHORIZED
	case domain.PaymentStatusDeclined:
		return contract.PaymentStatus_PAYMENT_STATUS_DECLINED
	default:
		return contract.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
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

func (h *PaymentHandler) ListPayments(ctx context.Context, req *contract.ListPaymentsRequest) (*contract.ListPaymentsResponse, error) {

	payments, err := h.useCase.ListPayments(ctx, req.GetStatus())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get payments: %v", err)
	}

	var pbPayments []*contract.PaymentResponse
	for _, p := range payments {
		pbPayments = append(pbPayments, &contract.PaymentResponse{
			Id:            p.ID,
			OrderId:       p.OrderID,
			TransactionId: p.TransactionID,
			Amount:        p.Amount,
			Status:        h.mapStatusToPb(p.Status),
		})
	}

	return &contract.ListPaymentsResponse{Payments: pbPayments}, nil
}
