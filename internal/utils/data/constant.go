package data

var (
	DEVELOPMENT_MODE = "development"
	STAGING_MODE     = "staging"
	PRODUCTION_MODE  = "production"

	REQUEST_ID_HEADER    = "X-Request-ID"
	REQUEST_ID_LOCAL_KEY = "request_id"

	CATEGORY_ID_FUND_TRANSFER          = "00000000-0000-0000-0000-000000000010"
	CATEGORY_ID_FUND_TRANSFER_CASH_IN  = "00000000-0000-0000-0000-000000000011"
	CATEGORY_ID_FUND_TRANSFER_CASH_OUT = "00000000-0000-0000-0000-000000000012"

	// Wallet nature, mirroring wallet_types.nature in the wallet service. A
	// liability wallet stores its remaining credit limit in balance.
	WALLET_NATURE_ASSET     = "asset"
	WALLET_NATURE_LIABILITY = "liability"
)

// HTTP header constants (go:S1192 — avoid duplicating string literals)
const (
	ContentTypeHeader = "Content-Type"
	ContentTypeJSON   = "application/json"
)

// Common HTTP error message constants (go:S1192)
const (
	ErrInvalidRequestBody  = "Invalid request body"
	ErrTransactionRequired = "Transaction ID is required"
	ErrWalletRequired      = "Wallet ID is required"
	ErrInvestmentRequired  = "Investment ID is required"
	ErrAttachmentRequired  = "Attachment ID is required"
)
