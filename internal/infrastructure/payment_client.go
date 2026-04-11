package infrastructure

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"order/internal/domain"

	contract "github.com/AldiyarMaget/aitu-go-sdk"
)

type PaymentClient struct {
	grpcAddr string
}

func NewPaymentClient(grpcAddr string) domain.PaymentClient {
	return &PaymentClient{
		grpcAddr: grpcAddr,
	}
}

func (c *PaymentClient) AuthorizePayment(ctx context.Context, orderID string, amount int64) (string, error) {
	conn, err := grpc.NewClient(c.grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	client := contract.NewPaymentServiceClient(conn)

	req := &contract.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	}

	resp, err := client.ProcessPayment(ctx, req)
	if err != nil {
		return "", err
	}

	switch resp.Status {
	case contract.PaymentStatus_PAYMENT_STATUS_AUTHORIZED:
		return "Authorized", nil
	case contract.PaymentStatus_PAYMENT_STATUS_DECLINED:
		return "Declined", nil
	default:
		return "", errors.New("unknown payment status")
	}
}
