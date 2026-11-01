package client

import (
	"context"
	"time"

	tpb "github.com/MuhammadMiftaa/Refina-Protobuf/transaction"
)

type TransactionClient interface {
	GetUserTransactions(ctx context.Context, req *tpb.GetUserTransactionsRequest) (*tpb.GetUserTransactionsResponse, error)
	GetTransactionByID(ctx context.Context, transactionID string) (*tpb.TransactionDetail, error)
	CreateTransaction(ctx context.Context, req *tpb.CreateTransactionRequest) (*tpb.TransactionDetail, error)
	CreateFundTransfer(ctx context.Context, req *tpb.CreateFundTransferRequest) (*tpb.FundTransferResponse, error)
	UpdateFundTransfer(ctx context.Context, req *tpb.UpdateFundTransferRequest) (*tpb.FundTransferResponse, error)
	DeleteFundTransfer(ctx context.Context, transferID string) (*tpb.FundTransferResponse, error)
	UpdateTransaction(ctx context.Context, req *tpb.UpdateTransactionRequest) (*tpb.TransactionDetail, error)
	DeleteTransaction(ctx context.Context, transactionID string) (*tpb.TransactionDetail, error)
	GetCategories(ctx context.Context, categoryType string) (*tpb.GetCategoriesResponse, error)
	GetAttachmentsByTransactionID(ctx context.Context, transactionID string) (*tpb.GetAttachmentsResponse, error)
	CreateAttachment(ctx context.Context, req *tpb.CreateAttachmentRequest) (*tpb.Attachment, error)
	DeleteAttachment(ctx context.Context, attachmentID string) (*tpb.Attachment, error)
	// Budget operations
	GetBudgets(ctx context.Context, period string) (*tpb.GetBudgetsResponse, error)
	CreateBudget(ctx context.Context, req *tpb.CreateBudgetRequest) (*tpb.Budget, error)
	UpdateBudget(ctx context.Context, req *tpb.UpdateBudgetRequest) (*tpb.Budget, error)
	DeleteBudget(ctx context.Context, budgetID string) (*tpb.Budget, error)
	ResetBudget(ctx context.Context, budgetID string) (*tpb.Budget, error)
}

type transactionClientImpl struct {
	client tpb.TransactionServiceClient
}

func NewTransactionClient(grpcClient tpb.TransactionServiceClient) TransactionClient {
	return &transactionClientImpl{
		client: grpcClient,
	}
}

func (t *transactionClientImpl) GetUserTransactions(ctx context.Context, req *tpb.GetUserTransactionsRequest) (*tpb.GetUserTransactionsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return t.client.GetUserTransactions(ctx, req)
}

func (t *transactionClientImpl) GetTransactionByID(ctx context.Context, transactionID string) (*tpb.TransactionDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return t.client.GetTransactionByID(ctx, &tpb.TransactionID{Id: transactionID})
}

func (t *transactionClientImpl) CreateTransaction(ctx context.Context, req *tpb.CreateTransactionRequest) (*tpb.TransactionDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return t.client.CreateTransaction(ctx, req)
}

func (t *transactionClientImpl) UpdateFundTransfer(ctx context.Context, req *tpb.UpdateFundTransferRequest) (*tpb.FundTransferResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return t.client.UpdateFundTransfer(ctx, req)
}

func (t *transactionClientImpl) DeleteFundTransfer(ctx context.Context, transferID string) (*tpb.FundTransferResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return t.client.DeleteFundTransfer(ctx, &tpb.TransferID{Id: transferID})
}

func (t *transactionClientImpl) CreateFundTransfer(ctx context.Context, req *tpb.CreateFundTransferRequest) (*tpb.FundTransferResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return t.client.CreateFundTransfer(ctx, req)
}

func (t *transactionClientImpl) UpdateTransaction(ctx context.Context, req *tpb.UpdateTransactionRequest) (*tpb.TransactionDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return t.client.UpdateTransaction(ctx, req)
}

func (t *transactionClientImpl) DeleteTransaction(ctx context.Context, transactionID string) (*tpb.TransactionDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return t.client.DeleteTransaction(ctx, &tpb.TransactionID{Id: transactionID})
}

func (t *transactionClientImpl) GetCategories(ctx context.Context, categoryType string) (*tpb.GetCategoriesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return t.client.GetCategories(ctx, &tpb.GetCategoriesRequest{Type: categoryType})
}

func (t *transactionClientImpl) GetAttachmentsByTransactionID(ctx context.Context, transactionID string) (*tpb.GetAttachmentsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return t.client.GetAttachmentsByTransactionID(ctx, &tpb.TransactionID{Id: transactionID})
}

func (t *transactionClientImpl) CreateAttachment(ctx context.Context, req *tpb.CreateAttachmentRequest) (*tpb.Attachment, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return t.client.CreateAttachment(ctx, req)
}

func (t *transactionClientImpl) DeleteAttachment(ctx context.Context, attachmentID string) (*tpb.Attachment, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return t.client.DeleteAttachment(ctx, &tpb.AttachmentID{Id: attachmentID})
}

// ── Budget operations ──

func (t *transactionClientImpl) GetBudgets(ctx context.Context, period string) (*tpb.GetBudgetsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return t.client.GetBudgets(ctx, &tpb.GetBudgetsRequest{Period: period})
}

func (t *transactionClientImpl) CreateBudget(ctx context.Context, req *tpb.CreateBudgetRequest) (*tpb.Budget, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return t.client.CreateBudget(ctx, req)
}

func (t *transactionClientImpl) UpdateBudget(ctx context.Context, req *tpb.UpdateBudgetRequest) (*tpb.Budget, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return t.client.UpdateBudget(ctx, req)
}

func (t *transactionClientImpl) DeleteBudget(ctx context.Context, budgetID string) (*tpb.Budget, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return t.client.DeleteBudget(ctx, &tpb.BudgetID{Id: budgetID})
}

func (t *transactionClientImpl) ResetBudget(ctx context.Context, budgetID string) (*tpb.Budget, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return t.client.ResetBudget(ctx, &tpb.BudgetID{Id: budgetID})
}
