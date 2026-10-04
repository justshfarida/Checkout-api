package services

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type XsollaService struct {
	projectID     string
	merchantID    string
	apiKey        string
	client        *http.Client
	webhookSecret string
	// returnURL is where Pay Station sends the user after payment. It is set
	// per environment so local and minikube each redirect back to themselves.
	returnURL string
}

func NewXsollaService(
	projectID string,
	merchantID string,
	apiKey string,
	webhookSecret string,
	returnURL string,
) *XsollaService {
	return &XsollaService{
		projectID:     projectID,
		merchantID:    merchantID,
		apiKey:        apiKey,
		webhookSecret: webhookSecret,
		returnURL:     returnURL,
		client:        &http.Client{},
	}
}

type xsollaPaymentRequest struct {
	Sandbox bool `json:"sandbox"`

	User struct {
		ID struct {
			Value string `json:"value"`
		} `json:"id"`

		Country struct {
			Value       string `json:"value"`
			AllowModify bool   `json:"allow_modify"`
		} `json:"country"`
	} `json:"user"`

	Purchase struct {
		Items []XsollaItem `json:"items"`
	} `json:"purchase"`
	CustomParameters map[string]string `json:"custom_parameters"`

	Settings *xsollaPaymentSettings `json:"settings,omitempty"`
}

type xsollaPaymentSettings struct {
	ReturnURL string `json:"return_url,omitempty"`
}

type XsollaItem struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

type XsollaPaymentResponse struct {
	Token   string `json:"token"`
	OrderID int    `json:"order_id"`
}

func (s *XsollaService) CreatePayment(
	ctx context.Context,
	userID int,
	orderID int,
	items []XsollaItem,
) (*XsollaPaymentResponse, error) {

	var body xsollaPaymentRequest

	body.Sandbox = true
	body.User.ID.Value = strconv.Itoa(userID)
	body.User.Country.Value = "US"
	body.User.Country.AllowModify = false
	body.Purchase.Items = items
	body.CustomParameters = map[string]string{
		"order_id": strconv.Itoa(orderID),
	}

	if s.returnURL != "" {
		body.Settings = &xsollaPaymentSettings{ReturnURL: s.returnURL}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"https://store.xsolla.com/api/v3/project/%s/admin/payment/token",
		s.projectID,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(s.projectID, s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"xsolla returned %s: %s",
			resp.Status,
			string(responseBody),
		)
	}

	var result XsollaPaymentResponse

	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
func (s *XsollaService) VerifyXsollaSignature(rawBody []byte, authHeader string) bool {
	const prefix = "Signature "

	if !strings.HasPrefix(authHeader, prefix) {
		return false
	}
	provided := strings.TrimSpace(
		strings.TrimPrefix(authHeader, prefix),
	)
	h := sha1.New()
	h.Write(rawBody)
	h.Write([]byte(s.webhookSecret))

	expected := hex.EncodeToString(
		h.Sum(nil),
	) //give me the SHA-1 result as bytes

	return subtle.ConstantTimeCompare(
		[]byte(strings.ToLower(provided)),
		[]byte(strings.ToLower(expected)),
	) == 1
}
