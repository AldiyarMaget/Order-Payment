package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"order/internal/domain"
)

type PaymentClient struct {
	client  *http.Client
	baseURL string
}

func NewPaymentClient(client *http.Client, baseURL string) domain.PaymentClient {
	return &PaymentClient{
		client:  client,
		baseURL: baseURL,
	}
}

type paymentRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type paymentResponse struct {
	Status string `json:"status"`
}

func (c *PaymentClient) AuthorizePayment(ctx context.Context, orderID string, amount int64) (string, error) {
	reqBody, _ := json.Marshal(paymentRequest{
		OrderID: orderID,
		Amount:  amount,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/payments", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable {
		return "", errors.New("503 Service Unavailable")
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", errors.New("payment failed with status: " + resp.Status)
	}

	var pRes paymentResponse
	if err := json.Unmarshal(bodyBytes, &pRes); err != nil {
		return "", err
	}

	return pRes.Status, nil
}
