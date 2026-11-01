package data

// Service field logging constants
const (
	MainService        = "main"
	EnvService         = "env"
	HTTPServerService  = "http_server"
	GRPCClientService  = "grpc_client"
	CacheService       = "cache"
	DashboardService   = "dashboard"
	WalletService      = "wallet"
	TransactionService = "transaction"
	InvestmentService  = "investment"
	ProfileService     = "profile"
	BudgetService      = "budget"
)

// Message field logging constants
const (
	// env / startup
	LogEnvVarMissing = "env_var_missing"

	// http server
	LogHTTPServerStarted        = "http_server_started"
	LogHTTPServerStartFailed    = "http_server_start_failed"
	LogHTTPServerShutdownFailed = "http_server_shutdown_failed"

	// grpc client
	LogGRPCClientSetupStarted   = "grpc_client_setup_started"
	LogGRPCClientSetupSuccess   = "grpc_client_setup_success"
	LogGRPCClientSetupFailed    = "grpc_client_setup_failed"
	LogGRPCClientClosed         = "grpc_client_closed"
	LogGRPCClientShutdownFailed = "grpc_client_shutdown_failed"

	// cache / redis
	LogRedisSetupFailed    = "redis_setup_failed"
	LogRedisSetupSuccess   = "redis_setup_success"
	LogRedisCloseFailed    = "redis_close_failed"
	LogCacheHit            = "cache_hit"
	LogCacheMiss           = "cache_miss"
	LogCacheSetFailed      = "cache_set_failed"
	LogCacheGetFailed      = "cache_get_failed"
	LogCacheInvalidated    = "cache_invalidated"
	LogCacheInvalidateFail = "cache_invalidate_failed"

	// shutdown
	LogShutdownSignalReceived      = "shutdown_signal_received"
	LogShutdownCompleted           = "shutdown_completed"
	LogShutdownCompletedWithErrors = "shutdown_completed_with_errors"

	// dashboard handler
	LogGetFinancialSummaryFailed     = "get_financial_summary_failed"
	LogGetUserBalanceFailed          = "get_user_balance_failed"
	LogGetUserTransactionsFailed     = "get_user_transactions_failed"
	LogGetNetWorthCompositionFailed  = "get_net_worth_composition_failed"
	LogGetUserWalletsFailed          = "get_user_wallets_failed"
	LogGetCategoryTransactionsFailed = "get_category_transactions_failed"

	// wallet handler
	LogGetWalletsFailed       = "get_wallets_failed"
	LogGetWalletByIDFailed    = "get_wallet_by_id_failed"
	LogCreateWalletFailed     = "create_wallet_failed"
	LogUpdateWalletFailed     = "update_wallet_failed"
	LogDeleteWalletFailed     = "delete_wallet_failed"
	LogGetWalletTypesFailed   = "get_wallet_types_failed"
	LogGetWalletSummaryFailed = "get_wallet_summary_failed"

	// transaction handler
	LogGetTransactionsFailed    = "get_transactions_failed"
	LogGetTransactionByIDFailed = "get_transaction_by_id_failed"
	LogCreateTransactionFailed  = "create_transaction_failed"
	LogCreateFundTransferFailed = "create_fund_transfer_failed"
	LogUpdateFundTransferFailed = "update_fund_transfer_failed"
	LogDeleteFundTransferFailed = "delete_fund_transfer_failed"
	LogUpdateTransactionFailed  = "update_transaction_failed"
	LogDeleteTransactionFailed  = "delete_transaction_failed"
	LogGetCategoriesFailed      = "get_categories_failed"
	LogGetAttachmentsFailed     = "get_attachments_failed"
	LogCreateAttachmentFailed   = "create_attachment_failed"
	LogDeleteAttachmentFailed   = "delete_attachment_failed"

	// investment handler
	LogGetInvestmentsFailed       = "get_investments_failed"
	LogGetInvestmentDetailFailed  = "get_investment_detail_failed"
	LogCreateInvestmentFailed     = "create_investment_failed"
	LogSellInvestmentFailed       = "sell_investment_failed"
	LogGetInvestmentSummaryFailed = "get_investment_summary_failed"
	LogGetAssetCodesFailed        = "get_asset_codes_failed"

	// profile handler
	LogGetProfileSuccess    = "get_profile_success"
	LogGetProfileFailed     = "get_profile_failed"
	LogUpdateProfileSuccess = "update_profile_success"
	LogUpdateProfileFailed  = "update_profile_failed"
	LogUploadPhotoSuccess   = "upload_photo_success"
	LogUploadPhotoFailed    = "upload_photo_failed"
	LogDeletePhotoSuccess   = "delete_photo_success"
	LogDeletePhotoFailed    = "delete_photo_failed"

	// budget handler
	LogGetBudgetsFailed    = "get_budgets_failed"
	LogGetBudgetsSuccess   = "get_budgets_success"
	LogCreateBudgetFailed  = "create_budget_failed"
	LogCreateBudgetSuccess = "create_budget_success"
	LogUpdateBudgetFailed  = "update_budget_failed"
	LogUpdateBudgetSuccess = "update_budget_success"
	LogDeleteBudgetFailed  = "delete_budget_failed"
	LogDeleteBudgetSuccess = "delete_budget_success"
	LogResetBudgetFailed   = "reset_budget_failed"
	LogResetBudgetSuccess  = "reset_budget_success"
)
