package grpc_delivery

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"order/internal/domain"
	"order/internal/usecase"

	contract "github.com/AldiyarMaget/aitu-go-sdk"
)

type OrderHandler struct {
	contract.UnimplementedOrderTrackingServiceServer
	useCase usecase.OrderUseCase
}

func NewOrderHandler(uc usecase.OrderUseCase) *OrderHandler {
	return &OrderHandler{
		useCase: uc,
	}
}

func (h *OrderHandler) SubscribeToOrderUpdates(req *contract.SubscribeOrderRequest, stream contract.OrderTrackingService_SubscribeToOrderUpdatesServer) error {
	orderID := req.GetOrderId()
	if orderID == "" {
		return status.Error(codes.InvalidArgument, "order_id is required")
	}

	_, err := h.useCase.GetOrder(stream.Context(), orderID)
	if err != nil {
		if err == usecase.ErrOrderNotFound {
			return status.Errorf(codes.NotFound, "order not found: %s", orderID)
		}
		return status.Errorf(codes.Internal, "failed to fetch order: %v", err)
	}

	ch, cleanup := h.useCase.SubscribeToOrder(stream.Context(), orderID)
	defer cleanup()

	for {
		select {
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "client closed the connection")
		case order, ok := <-ch:
			if !ok {
				// The channel was closed (e.g. system shutting down)
				return nil
			}

			var pbStatus contract.OrderStatus
			switch order.Status {
			case domain.StatusPending:
				pbStatus = contract.OrderStatus_ORDER_STATUS_PENDING
			case domain.StatusPaid:
				pbStatus = contract.OrderStatus_ORDER_STATUS_PAID
			case domain.StatusFailed:
				pbStatus = contract.OrderStatus_ORDER_STATUS_FAILED
			case domain.StatusCancelled:
				pbStatus = contract.OrderStatus_ORDER_STATUS_CANCELLED
			default:
				pbStatus = contract.OrderStatus_ORDER_STATUS_UNSPECIFIED
			}

			update := &contract.OrderStatusUpdate{
				Id:             order.ID,
				CustomerId:     order.CustomerID,
				ItemName:       order.ItemName,
				Amount:         order.Amount,
				Status:         pbStatus,
				CreatedAt:      timestamppb.New(order.CreatedAt),
				IdempotencyKey: order.IdempotencyKey,
			}

			if err := stream.Send(update); err != nil {
				return status.Errorf(codes.Internal, "failed to send order update stream: %v", err)
			}

			// If the order reached a terminal state, we should cleanly terminate the stream
			if order.Status == domain.StatusPaid || order.Status == domain.StatusFailed || order.Status == domain.StatusCancelled {
				return nil
			}
		}
	}
}
