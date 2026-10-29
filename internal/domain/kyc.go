package domain

// SubscriberDocument matches actual 'subscriber_documents' table columns exactly.
// DB columns: document_id, subscriber_id, document_type, document_path,
//
//	document_number, uploaded_date, expiry_date,
//	verification_status, verified_by, created_at
type SubscriberDocument struct {
	DocumentID         int    `json:"document_id"`
	SubscriberID       int    `json:"subscriber_id"`
	DocumentType       string `json:"document_type"` // AADHAAR, PAN, PHOTO, SIGNATURE, ADDRESS_PROOF, INCOME_PROOF
	FilePath           string `json:"document_path"` // maps to document_path column
	DocumentNumber     string `json:"document_number"`
	UploadedDate       string `json:"uploaded_date"`       // formatted as YYYY-MM-DD
	ExpiryDate         string `json:"expiry_date"`         // formatted as YYYY-MM-DD, "" if null
	VerificationStatus string `json:"verification_status"` // PENDING, VERIFIED, REJECTED
	VerifiedByText     string `json:"verified_by"`         // stored as text "123" or ""
	CreatedAt          string `json:"created_at"`          // ISO timestamp
}
