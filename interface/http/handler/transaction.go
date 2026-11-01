package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	logger "refina-web-bff/config/log"
	grpcClient "refina-web-bff/interface/grpc/client"
	"refina-web-bff/interface/grpc/interceptor"
	"refina-web-bff/internal/cache"
	"refina-web-bff/internal/types/dto"
	"refina-web-bff/internal/utils"
	"refina-web-bff/internal/utils/data"

	tpb "github.com/MuhammadMiftaa/Refina-Protobuf/transaction"
	"github.com/gofiber/fiber/v2"
)

type transactionHandler struct {
	transaction grpcClient.TransactionClient
	wallet      grpcClient.WalletClient
	cache       cache.Cache
}

func NewTransactionHandler(tc grpcClient.TransactionClient, wc grpcClient.WalletClient, c cache.Cache) *transactionHandler {
	return &transactionHandler{
		transaction: tc,
		wallet:      wc,
		cache:       c,
	}
}

// ── Transaction Handlers ──

// GetUserTransactions — GET /transactions
func (h *transactionHandler) GetUserTransactions(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	// Build gRPC request from query params
	pageSize, _ := strconv.Atoi(c.Query("page_size", "10"))
	cursorAmount, _ := strconv.ParseFloat(c.Query("cursor_amount", "0"), 64)

	req := &tpb.GetUserTransactionsRequest{
		PageSize:     int32(pageSize),
		SortBy:       c.Query("sort_by"),
		SortOrder:    c.Query("sort_order"),
		Search:       c.Query("search"),
		WalletId:     c.Query("wallet_id"),
		CategoryId:   c.Query("category_id"),
		CategoryType: c.Query("category_type"),
		DateFrom:     c.Query("date_from"),
		DateTo:       c.Query("date_to"),
		Cursor:       c.Query("cursor"),
		CursorAmount: cursorAmount,
		CursorDate:   c.Query("cursor_date"),
	}

	// Build cache key from query params
	paramsHash := cache.HashParams(fmt.Sprintf("%+v", req))
	cacheKey := cache.TransactionList(userData.ID, paramsHash)

	if cached, err := h.cache.Get(c.UserContext(), cacheKey); err == nil && cached != nil {
		logger.Debug(data.LogCacheHit, map[string]any{"service": data.CacheService, "key": cacheKey})
		c.Set(data.ContentTypeHeader, data.ContentTypeJSON)
		return c.Send(cached)
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	userWallets, err := h.wallet.GetUserWallets(ctx, userData.ID)
	if err != nil {
		logger.Error(data.LogGetWalletsFailed, map[string]any{
			"service":    data.WalletService,
			"request_id": requestID,
			"user_id":    userData.ID,
			"error":      err.Error(),
		})
		grpcErr := utils.MapGRPCError(err)
		return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: grpcErr.HTTPStatus,
			Message:    grpcErr.Message,
		})
	}

	walletIDs := make([]string, 0, len(userWallets.GetWallets()))
	walletMap := make(map[string]string)
	for _, wallet := range userWallets.GetWallets() {
		walletIDs = append(walletIDs, wallet.GetId())
		walletMap[wallet.GetId()] = wallet.GetName()
	}

	req.WalletIds = walletIDs

	result, err := h.transaction.GetUserTransactions(ctx, req)
	if err != nil {
		logger.Error(data.LogGetTransactionsFailed, map[string]any{
			"service":    data.TransactionService,
			"request_id": requestID,
			"user_id":    userData.ID,
			"error":      err.Error(),
		})
		grpcErr := utils.MapGRPCError(err)
		return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: grpcErr.HTTPStatus,
			Message:    grpcErr.Message,
		})
	}

	if result.GetTransactions() != nil && len(result.GetTransactions()) >= 0 {
		for _, tx := range result.GetTransactions() {
			tx.WalletName = walletMap[tx.GetWalletId()]
		}
	}

	resp := dto.APIResponse{
		Status:     true,
		StatusCode: 200,
		Message:    "Transactions retrieved successfully",
		Data:       result,
	}

	if b, err := json.Marshal(resp); err == nil {
		if err := h.cache.Set(c.UserContext(), cacheKey, b, cache.TTLShort); err != nil {
			logger.Warn(data.LogCacheSetFailed, map[string]any{"service": data.CacheService, "key": cacheKey, "error": err.Error()})
		}
	}

	return c.JSON(resp)
}

// GetTransactionByID — GET /transactions/:id
func (h *transactionHandler) GetTransactionByID(c *fiber.Ctx) error {
	transactionID := c.Params("id")
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	if transactionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    data.ErrTransactionRequired,
		})
	}

	cacheKey := cache.TransactionByID(transactionID)

	if cached, err := h.cache.Get(c.UserContext(), cacheKey); err == nil && cached != nil {
		logger.Debug(data.LogCacheHit, map[string]any{"service": data.CacheService, "key": cacheKey})
		c.Set(data.ContentTypeHeader, data.ContentTypeJSON)
		return c.Send(cached)
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	result, err := h.transaction.GetTransactionByID(ctx, transactionID)
	if err != nil {
		logger.Error(data.LogGetTransactionByIDFailed, map[string]any{
			"service":        data.TransactionService,
			"request_id":     requestID,
			"transaction_id": transactionID,
			"error":          err.Error(),
		})
		grpcErr := utils.MapGRPCError(err)
		return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: grpcErr.HTTPStatus,
			Message:    grpcErr.Message,
		})
	}

	resp := dto.APIResponse{
		Status:     true,
		StatusCode: 200,
		Message:    "Transaction retrieved successfully",
		Data:       result,
	}

	if b, err := json.Marshal(resp); err == nil {
		if err := h.cache.Set(c.UserContext(), cacheKey, b, cache.TTLMedium); err != nil {
			logger.Warn(data.LogCacheSetFailed, map[string]any{"service": data.CacheService, "key": cacheKey, "error": err.Error()})
		}
	}

	return c.JSON(resp)
}

// CreateTransaction — POST /transactions
func (h *transactionHandler) CreateTransaction(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	var req dto.CreateTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    data.ErrInvalidRequestBody,
		})
	}

	if req.WalletID == "" || req.CategoryID == "" || req.TransactionDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    "wallet_id, category_id, and transaction_date are required",
		})
	}

	// ── Wallet ownership & balance validation ──
	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	wallet, err := h.wallet.GetWalletByID(ctx, req.WalletID)
	if err != nil {
		logger.Error(data.LogGetWalletByIDFailed, map[string]any{
			"service":    data.WalletService,
			"request_id": requestID,
			"user_id":    userData.ID,
			"wallet_id":  req.WalletID,
			"error":      err.Error(),
		})
		grpcErr := utils.MapGRPCError(err)
		return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: grpcErr.HTTPStatus,
			Message:    grpcErr.Message,
		})
	}

	if wallet.GetUserId() != userData.ID {
		return c.Status(fiber.StatusForbidden).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 403,
			Message:    "You do not have access to this wallet",
		})
	}

	grpcReq := &tpb.CreateTransactionRequest{
		UserId:             userData.ID,
		WalletId:           req.WalletID,
		CategoryId:         req.CategoryID,
		Amount:             req.Amount,
		TransactionDate:    req.TransactionDate,
		Description:        req.Description,
		Attachments:        req.Attachments,
		IsWalletNotCreated: false,
	}

	result, err := h.transaction.CreateTransaction(ctx, grpcReq)
	if err != nil {
		logger.Error(data.LogCreateTransactionFailed, map[string]any{
			"service":    data.TransactionService,
			"request_id": requestID,
			"user_id":    userData.ID,
			"error":      err.Error(),
		})
		grpcErr := utils.MapGRPCError(err)
		return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: grpcErr.HTTPStatus,
			Message:    grpcErr.Message,
		})
	}

	// Invalidate transaction, wallet, and dashboard caches
	h.invalidateTransactionCaches(c.UserContext(), userData.ID)

	return c.Status(fiber.StatusCreated).JSON(dto.APIResponse{
		Status:     true,
		StatusCode: 201,
		Message:    "Transaction created successfully",
		Data:       result,
	})
}

// CreateFundTransfer — POST /transactions/transfer
func (h *transactionHandler) CreateFundTransfer(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	var req dto.CreateFundTransferRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    data.ErrInvalidRequestBody,
		})
	}

	if req.FromWalletID == "" || req.ToWalletID == "" || req.TransactionDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    "from_wallet_id, to_wallet_id, and transaction_date are required",
		})
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	// Both wallets must be the caller's own, as for a single transaction.
	if resp, ok := h.requireOwnWallets(c, ctx, userData.ID, requestID, req.FromWalletID, req.ToWalletID); !ok {
		return resp
	}

	grpcReq := &tpb.CreateFundTransferRequest{
		UserId:            userData.ID,
		FromWalletId:      req.FromWalletID,
		ToWalletId:        req.ToWalletID,
		Amount:            req.Amount,
		AdminFee:          req.AdminFee,
		CashOutCategoryId: data.CATEGORY_ID_FUND_TRANSFER_CASH_OUT,
		CashInCategoryId:  data.CATEGORY_ID_FUND_TRANSFER_CASH_IN,
		TransactionDate:   req.TransactionDate,
		Description:       req.Description,
	}

	result, err := h.transaction.CreateFundTransfer(ctx, grpcReq)
	if err != nil {
		logger.Error(data.LogCreateFundTransferFailed, map[string]any{
			"service":    data.TransactionService,
			"request_id": requestID,
			"user_id":    userData.ID,
			"error":      err.Error(),
		})
		grpcErr := utils.MapGRPCError(err)
		return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: grpcErr.HTTPStatus,
			Message:    grpcErr.Message,
		})
	}

	// Invalidate transaction, wallet, and dashboard caches (both wallets involved)
	h.invalidateTransactionCaches(c.UserContext(), userData.ID)
	_ = h.cache.Delete(c.UserContext(), cache.WalletByID(req.FromWalletID), cache.WalletByID(req.ToWalletID))

	return c.Status(fiber.StatusCreated).JSON(dto.APIResponse{
		Status:     true,
		StatusCode: 201,
		Message:    "Fund transfer created successfully",
		Data:       result,
	})
}

// UpdateFundTransfer — PUT /transactions/transfer/:transferId
//
// Replaces a whole transfer (both legs, both balances) in one step. The
// transaction service checks that every wallet involved is the caller's.
func (h *transactionHandler) UpdateFundTransfer(c *fiber.Ctx) error {
	transferID := c.Params("transferId")
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	var req dto.UpdateFundTransferRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    data.ErrInvalidRequestBody,
		})
	}

	if transferID == "" || req.FromWalletID == "" || req.ToWalletID == "" || req.TransactionDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    "transfer id, from_wallet_id, to_wallet_id, and transaction_date are required",
		})
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	result, err := h.transaction.UpdateFundTransfer(ctx, &tpb.UpdateFundTransferRequest{
		TransferId:      transferID,
		FromWalletId:    req.FromWalletID,
		ToWalletId:      req.ToWalletID,
		Amount:          req.Amount,
		AdminFee:        req.AdminFee,
		TransactionDate: req.TransactionDate,
		Description:     req.Description,
	})
	if err != nil {
		logger.Error(data.LogUpdateFundTransferFailed, map[string]any{
			"service":     data.TransactionService,
			"request_id":  requestID,
			"user_id":     userData.ID,
			"transfer_id": transferID,
			"error":       err.Error(),
		})
		grpcErr := utils.MapGRPCError(err)
		return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: grpcErr.HTTPStatus,
			Message:    grpcErr.Message,
		})
	}

	h.invalidateTransactionCaches(c.UserContext(), userData.ID)
	_ = h.cache.Delete(c.UserContext(),
		cache.TransactionByID(result.GetCashOutTransactionId()),
		cache.TransactionByID(result.GetCashInTransactionId()),
	)

	return c.JSON(dto.APIResponse{
		Status:     true,
		StatusCode: 200,
		Message:    "Fund transfer updated successfully",
		Data:       result,
	})
}

// DeleteFundTransfer — DELETE /transactions/transfer/:transferId
func (h *transactionHandler) DeleteFundTransfer(c *fiber.Ctx) error {
	transferID := c.Params("transferId")
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	result, err := h.transaction.DeleteFundTransfer(ctx, transferID)
	if err != nil {
		logger.Error(data.LogDeleteFundTransferFailed, map[string]any{
			"service":     data.TransactionService,
			"request_id":  requestID,
			"user_id":     userData.ID,
			"transfer_id": transferID,
			"error":       err.Error(),
		})
		grpcErr := utils.MapGRPCError(err)
		return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: grpcErr.HTTPStatus,
			Message:    grpcErr.Message,
		})
	}

	h.invalidateTransactionCaches(c.UserContext(), userData.ID)
	_ = h.cache.Delete(c.UserContext(),
		cache.TransactionByID(result.GetCashOutTransactionId()),
		cache.TransactionByID(result.GetCashInTransactionId()),
	)

	return c.JSON(dto.APIResponse{
		Status:     true,
		StatusCode: 200,
		Message:    "Fund transfer deleted successfully",
		Data:       result,
	})
}

// requireOwnWallets answers 403 unless every wallet belongs to the caller.
// The second result is false when the response has already been written.
func (h *transactionHandler) requireOwnWallets(c *fiber.Ctx, ctx context.Context, userID, requestID string, walletIDs ...string) (error, bool) {
	for _, walletID := range walletIDs {
		wallet, err := h.wallet.GetWalletByID(ctx, walletID)
		if err != nil {
			logger.Error(data.LogGetWalletByIDFailed, map[string]any{
				"service":    data.WalletService,
				"request_id": requestID,
				"wallet_id":  walletID,
				"error":      err.Error(),
			})
			grpcErr := utils.MapGRPCError(err)
			return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
				Status:     false,
				StatusCode: grpcErr.HTTPStatus,
				Message:    grpcErr.Message,
			}), false
		}
		if wallet.GetUserId() != userID {
			return c.Status(fiber.StatusForbidden).JSON(dto.APIResponse{
				Status:     false,
				StatusCode: 403,
				Message:    "You do not have access to this wallet",
			}), false
		}
	}
	return nil, true
}

// UpdateTransaction — PUT /transactions/:id
func (h *transactionHandler) UpdateTransaction(c *fiber.Ctx) error {
	transactionID := c.Params("id")
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	if transactionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    data.ErrTransactionRequired,
		})
	}

	var req dto.UpdateTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    data.ErrInvalidRequestBody,
		})
	}

	// Convert attachment actions to proto
	var attachmentActions []*tpb.UpdateAttachmentAction
	for _, action := range req.AttachmentActions {
		attachmentActions = append(attachmentActions, &tpb.UpdateAttachmentAction{
			Status: action.Status,
			Files:  action.Files,
		})
	}

	grpcReq := &tpb.UpdateTransactionRequest{
		Id:                transactionID,
		WalletId:          req.WalletID,
		CategoryId:        req.CategoryID,
		Amount:            req.Amount,
		TransactionDate:   req.TransactionDate,
		Description:       req.Description,
		AttachmentActions: attachmentActions,
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	result, err := h.transaction.UpdateTransaction(ctx, grpcReq)
	if err != nil {
		logger.Error(data.LogUpdateTransactionFailed, map[string]any{
			"service":        data.TransactionService,
			"request_id":     requestID,
			"transaction_id": transactionID,
			"error":          err.Error(),
		})
		grpcErr := utils.MapGRPCError(err)
		return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: grpcErr.HTTPStatus,
			Message:    grpcErr.Message,
		})
	}

	// Invalidate transaction, wallet, and dashboard caches
	h.invalidateTransactionCaches(c.UserContext(), userData.ID)
	_ = h.cache.Delete(c.UserContext(), cache.TransactionByID(transactionID))

	return c.JSON(dto.APIResponse{
		Status:     true,
		StatusCode: 200,
		Message:    "Transaction updated successfully",
		Data:       result,
	})
}

// DeleteTransaction — DELETE /transactions/:id
func (h *transactionHandler) DeleteTransaction(c *fiber.Ctx) error {
	transactionID := c.Params("id")
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	if transactionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    data.ErrTransactionRequired,
		})
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	_, err := h.transaction.DeleteTransaction(ctx, transactionID)
	if err != nil {
		logger.Error(data.LogDeleteTransactionFailed, map[string]any{
			"service":        data.TransactionService,
			"request_id":     requestID,
			"transaction_id": transactionID,
			"error":          err.Error(),
		})
		grpcErr := utils.MapGRPCError(err)
		return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: grpcErr.HTTPStatus,
			Message:    grpcErr.Message,
		})
	}

	// Invalidate transaction, wallet, and dashboard caches
	h.invalidateTransactionCaches(c.UserContext(), userData.ID)
	_ = h.cache.Delete(c.UserContext(), cache.TransactionByID(transactionID))

	return c.JSON(dto.APIResponse{
		Status:     true,
		StatusCode: 200,
		Message:    "Transaction deleted successfully",
	})
}

// ── Category Handlers ──

// GetCategories — GET /categories
func (h *transactionHandler) GetCategories(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	categoryType := c.Query("type") // optional filter: income, expense, fund_transfer

	cacheKey := cache.TransactionCategories(categoryType)

	if cached, err := h.cache.Get(c.UserContext(), cacheKey); err == nil && cached != nil {
		logger.Debug(data.LogCacheHit, map[string]any{"service": data.CacheService, "key": cacheKey})
		c.Set(data.ContentTypeHeader, data.ContentTypeJSON)
		return c.Send(cached)
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	result, err := h.transaction.GetCategories(ctx, categoryType)
	if err != nil {
		logger.Error(data.LogGetCategoriesFailed, map[string]any{
			"service":    data.TransactionService,
			"request_id": requestID,
			"type":       categoryType,
			"error":      err.Error(),
		})
		grpcErr := utils.MapGRPCError(err)
		return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: grpcErr.HTTPStatus,
			Message:    grpcErr.Message,
		})
	}

	resp := dto.APIResponse{
		Status:     true,
		StatusCode: 200,
		Message:    "Categories retrieved successfully",
		Data:       result.GetCategories(),
	}

	if b, err := json.Marshal(resp); err == nil {
		if err := h.cache.Set(c.UserContext(), cacheKey, b, cache.TTLStatic); err != nil {
			logger.Warn(data.LogCacheSetFailed, map[string]any{"service": data.CacheService, "key": cacheKey, "error": err.Error()})
		}
	}

	return c.JSON(resp)
}

// ── Attachment Handlers ──

// GetAttachmentsByTransaction — GET /attachments/transaction/:transactionId
func (h *transactionHandler) GetAttachmentsByTransaction(c *fiber.Ctx) error {
	transactionID := c.Params("transactionId")
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	if transactionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    data.ErrTransactionRequired,
		})
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	result, err := h.transaction.GetAttachmentsByTransactionID(ctx, transactionID)
	if err != nil {
		logger.Error(data.LogGetAttachmentsFailed, map[string]any{
			"service":        data.TransactionService,
			"request_id":     requestID,
			"transaction_id": transactionID,
			"error":          err.Error(),
		})
		grpcErr := utils.MapGRPCError(err)
		return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: grpcErr.HTTPStatus,
			Message:    grpcErr.Message,
		})
	}

	return c.JSON(dto.APIResponse{
		Status:     true,
		StatusCode: 200,
		Message:    "Attachments retrieved successfully",
		Data:       result.GetAttachments(),
	})
}

// CreateAttachment — POST /attachments
func (h *transactionHandler) CreateAttachment(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	var req dto.CreateAttachmentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    data.ErrInvalidRequestBody,
		})
	}

	if req.TransactionID == "" || req.Image == "" || req.Format == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    "transaction_id, image, and format are required",
		})
	}

	grpcReq := &tpb.CreateAttachmentRequest{
		TransactionId: req.TransactionID,
		Image:         req.Image,
		Format:        req.Format,
		Size:          req.Size,
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	result, err := h.transaction.CreateAttachment(ctx, grpcReq)
	if err != nil {
		logger.Error(data.LogCreateAttachmentFailed, map[string]any{
			"service":        data.TransactionService,
			"request_id":     requestID,
			"transaction_id": req.TransactionID,
			"error":          err.Error(),
		})
		grpcErr := utils.MapGRPCError(err)
		return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: grpcErr.HTTPStatus,
			Message:    grpcErr.Message,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(dto.APIResponse{
		Status:     true,
		StatusCode: 201,
		Message:    "Attachment created successfully",
		Data:       result,
	})
}

// DeleteAttachment — DELETE /attachments/:id
func (h *transactionHandler) DeleteAttachment(c *fiber.Ctx) error {
	attachmentID := c.Params("id")
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	if attachmentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    data.ErrAttachmentRequired,
		})
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	_, err := h.transaction.DeleteAttachment(ctx, attachmentID)
	if err != nil {
		logger.Error(data.LogDeleteAttachmentFailed, map[string]any{
			"service":       data.TransactionService,
			"request_id":    requestID,
			"attachment_id": attachmentID,
			"error":         err.Error(),
		})
		grpcErr := utils.MapGRPCError(err)
		return c.Status(grpcErr.HTTPStatus).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: grpcErr.HTTPStatus,
			Message:    grpcErr.Message,
		})
	}

	return c.JSON(dto.APIResponse{
		Status:     true,
		StatusCode: 200,
		Message:    "Attachment deleted successfully",
	})
}

// invalidateTransactionCaches clears transaction, wallet, and dashboard caches for a user.
func (h *transactionHandler) invalidateTransactionCaches(ctx context.Context, userID string) {
	// Pattern-based invalidation
	patterns := []string{
		cache.TransactionListPattern(userID),
		cache.WalletAllPattern(userID),
		cache.DashboardAllPattern(userID),
		// Budgets show what was spent this month, so any change to a transaction moves them.
		cache.BudgetAllPattern(userID),
	}
	for _, p := range patterns {
		if err := h.cache.DeleteByPattern(ctx, p); err != nil {
			logger.Warn(data.LogCacheInvalidateFail, map[string]any{"service": data.CacheService, "pattern": p, "error": err.Error()})
		}
	}
}
