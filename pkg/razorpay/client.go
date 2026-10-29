package razorpay

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://api.razorpay.com/v1"

type Client struct {
	keyID     string
	keySecret string
	http      *http.Client
}

func NewClient(keyID, keySecret string) *Client {
	return &Client{
		keyID:     keyID,
		keySecret: keySecret,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

// ── Orders ────────────────────────────────────────────────────────────────────

type CreateOrderReq struct {
	Amount   int64             `json:"amount"` // paise
	Currency string            `json:"currency"`
	Receipt  string            `json:"receipt"`
	Notes    map[string]string `json:"notes,omitempty"`
}

type OrderResp struct {
	ID       string `json:"id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Receipt  string `json:"receipt"`
	Status   string `json:"status"`
}

func (c *Client) CreateOrder(req *CreateOrderReq) (*OrderResp, error) {
	var resp OrderResp
	return &resp, c.post("/orders", req, &resp)
}

// ── Payments ──────────────────────────────────────────────────────────────────

type PaymentDetails struct {
	ID      string `json:"id"`
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
	Status  string `json:"status"` // authorized | captured | failed
	Method  string `json:"method"`
}

func (c *Client) FetchPayment(gatewayPaymentID string) (*PaymentDetails, error) {
	var resp PaymentDetails
	return &resp, c.get("/payments/"+gatewayPaymentID, &resp)
}

func (c *Client) CapturePayment(gatewayPaymentID string, amount int64) error {
	return c.post("/payments/"+gatewayPaymentID+"/capture",
		map[string]interface{}{"amount": amount, "currency": "INR"}, nil)
}

// ── Refunds ───────────────────────────────────────────────────────────────────

type RefundReq struct {
	Amount int64             `json:"amount"`
	Speed  string            `json:"speed"` // normal | optimum
	Notes  map[string]string `json:"notes,omitempty"`
}

type RefundResp struct {
	ID        string `json:"id"`
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount"`
	Status    string `json:"status"` // pending | processed | failed
}

func (c *Client) CreateRefund(gatewayPaymentID string, req *RefundReq) (*RefundResp, error) {
	var resp RefundResp
	return &resp, c.post("/payments/"+gatewayPaymentID+"/refund", req, &resp)
}

// ── Contacts & Fund Accounts (Razorpay X payouts) ─────────────────────────────

type CreateContactReq struct {
	Name        string `json:"name"`
	Email       string `json:"email,omitempty"`
	Contact     string `json:"contact,omitempty"`
	Type        string `json:"type"` // customer
	ReferenceID string `json:"reference_id,omitempty"`
}

type ContactResp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) CreateContact(req *CreateContactReq) (*ContactResp, error) {
	var resp ContactResp
	return &resp, c.post("/contacts", req, &resp)
}

type CreateFundAccountReq struct {
	ContactID   string       `json:"contact_id"`
	AccountType string       `json:"account_type"` // bank_account | vpa
	BankAccount *BankAcctReq `json:"bank_account,omitempty"`
	VPA         *VPAReq      `json:"vpa,omitempty"`
}

type BankAcctReq struct {
	Name          string `json:"name"`
	IFSC          string `json:"ifsc"`
	AccountNumber string `json:"account_number"`
}

type VPAReq struct {
	Address string `json:"address"`
}

type FundAccountResp struct {
	ID          string `json:"id"`
	ContactID   string `json:"contact_id"`
	AccountType string `json:"account_type"`
}

func (c *Client) CreateFundAccount(req *CreateFundAccountReq) (*FundAccountResp, error) {
	var resp FundAccountResp
	return &resp, c.post("/fund_accounts", req, &resp)
}

// ── Payouts (Razorpay X) ──────────────────────────────────────────────────────

type CreatePayoutReq struct {
	AccountNumber string            `json:"account_number"` // RazorpayX virtual account
	FundAccountID string            `json:"fund_account_id"`
	Amount        int64             `json:"amount"` // paise
	Currency      string            `json:"currency"`
	Mode          string            `json:"mode"` // NEFT | RTGS | IMPS | UPI
	Purpose       string            `json:"purpose"`
	Queue         bool              `json:"queue_if_low_balance"`
	ReferenceID   string            `json:"reference_id"`
	Narration     string            `json:"narration"`
	Notes         map[string]string `json:"notes,omitempty"`
}

type PayoutResp struct {
	ID            string `json:"id"`
	FundAccountID string `json:"fund_account_id"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"` // queued | pending | processing | processed | cancelled | reversed
	UTR           string `json:"utr"`
	ReferenceID   string `json:"reference_id"`
}

func (c *Client) CreatePayout(req *CreatePayoutReq) (*PayoutResp, error) {
	var resp PayoutResp
	return &resp, c.postWithHeader("/payouts",
		map[string]string{"X-Payout-Idempotency": req.ReferenceID},
		req, &resp)
}

// ── Signature verification ────────────────────────────────────────────────────

// VerifyPaymentSignature verifies Razorpay Checkout signature.
// data = gatewayOrderID + "|" + gatewayPaymentID
func (c *Client) VerifyPaymentSignature(gatewayOrderID, gatewayPaymentID, signature string) bool {
	data := gatewayOrderID + "|" + gatewayPaymentID
	return verifyHMAC([]byte(data), signature, c.keySecret)
}

// VerifyWebhookSignature verifies an inbound Razorpay webhook using the webhook secret.
func VerifyWebhookSignature(rawBody []byte, signature, secret string) bool {
	return verifyHMAC(rawBody, signature, secret)
}

func verifyHMAC(data []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

func (c *Client) post(path string, body, out interface{}) error {
	return c.postWithHeader(path, nil, body, out)
}

func (c *Client) postWithHeader(path string, headers map[string]string, body, out interface{}) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", baseURL+path, bytes.NewBuffer(b))
	req.SetBasicAuth(c.keyID, c.keySecret)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.do(req, out)
}

func (c *Client) get(path string, out interface{}) error {
	req, _ := http.NewRequest("GET", baseURL+path, nil)
	req.SetBasicAuth(c.keyID, c.keySecret)
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out interface{}) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("razorpay %s: %w", req.URL.Path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("razorpay %s (%d): %s", req.URL.Path, resp.StatusCode, raw)
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}
