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

	ipb "github.com/MuhammadMiftaa/Refina-Protobuf/investment"
	"github.com/gofiber/fiber/v2"
)

type investmentHandler struct {
	investment grpcClient.InvestmentClient
	wallet     grpcClient.WalletClient
	cache      cache.Cache
}

func NewInvestmentHandler(ic grpcClient.InvestmentClient, wc grpcClient.WalletClient, c cache.Cache) *investmentHandler {
	return &investmentHandler{investment: ic, wallet: wc, cache: c}
}

// GetUserInvestments — GET /investments
func (h *investmentHandler) GetUserInvestments(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "10"))

	req := &ipb.GetUserInvestmentListRequest{
		UserId:    userData.ID,
		Page:      int32(page),
		PageSize:  int32(pageSize),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
		Search:    c.Query("search"),
		Code:      c.Query("code"),
	}

	paramsHash := cache.HashParams(fmt.Sprintf("%+v", req))
	cacheKey := cache.InvestmentList(userData.ID, paramsHash)

	if cached, err := h.cache.Get(c.UserContext(), cacheKey); err == nil && cached != nil {
		logger.Debug(data.LogCacheHit, map[string]any{"service": data.CacheService, "key": cacheKey})
		c.Set(data.ContentTypeHeader, data.ContentTypeJSON)
		return c.Send(cached)
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	result, err := h.investment.GetUserInvestmentList(ctx, req)
	if err != nil {
		logger.Error(data.LogGetInvestmentsFailed, map[string]any{
			"service":    data.InvestmentService,
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

	resp := dto.APIResponse{
		Status:     true,
		StatusCode: 200,
		Message:    "Investments retrieved successfully",
		Data:       result,
	}

	if b, err := json.Marshal(resp); err == nil {
		if err := h.cache.Set(c.UserContext(), cacheKey, b, 3*cache.TTLShort/2); err != nil { // ~3 min
			logger.Warn(data.LogCacheSetFailed, map[string]any{"service": data.CacheService, "key": cacheKey, "error": err.Error()})
		}
	}

	return c.JSON(resp)
}

// GetInvestmentDetail — GET /investments/:id
func (h *investmentHandler) GetInvestmentDetail(c *fiber.Ctx) error {
	investmentID := c.Params("id")
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	if investmentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    data.ErrInvestmentRequired,
		})
	}

	cacheKey := cache.InvestmentDetail(investmentID)

	if cached, err := h.cache.Get(c.UserContext(), cacheKey); err == nil && cached != nil {
		logger.Debug(data.LogCacheHit, map[string]any{"service": data.CacheService, "key": cacheKey})
		c.Set(data.ContentTypeHeader, data.ContentTypeJSON)
		return c.Send(cached)
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	result, err := h.investment.GetInvestmentDetail(ctx, &ipb.GetInvestmentDetailRequest{
		UserId:       userData.ID,
		InvestmentId: investmentID,
	})
	if err != nil {
		logger.Error(data.LogGetInvestmentDetailFailed, map[string]any{
			"service":       data.InvestmentService,
			"request_id":    requestID,
			"investment_id": investmentID,
			"error":         err.Error(),
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
		Message:    "Investment detail retrieved successfully",
		Data:       result,
	}

	if b, err := json.Marshal(resp); err == nil {
		if err := h.cache.Set(c.UserContext(), cacheKey, b, cache.TTLMedium); err != nil {
			logger.Warn(data.LogCacheSetFailed, map[string]any{"service": data.CacheService, "key": cacheKey, "error": err.Error()})
		}
	}

	return c.JSON(resp)
}

// CreateInvestment — POST /investments
func (h *investmentHandler) CreateInvestment(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	var body dto.CreateInvestmentRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    data.ErrInvalidRequestBody,
		})
	}

	// ── Wallet ownership & balance validation ──
	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	wallet, err := h.wallet.GetWalletByID(ctx, body.WalletId)
	if err != nil {
		logger.Error(data.LogGetWalletByIDFailed, map[string]any{
			"service":    data.WalletService,
			"request_id": requestID,
			"user_id":    userData.ID,
			"wallet_id":  body.WalletId,
			"error":      err.Error(),
		})
		return c.Status(fiber.StatusNotFound).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 404,
			Message:    "Wallet not found",
		})
	}

	if wallet.GetUserId() != userData.ID {
		return c.Status(fiber.StatusForbidden).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 403,
			Message:    "You do not have access to this wallet",
		})
	}

	if body.Amount > 0 && wallet.GetBalance() < body.Amount {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    fmt.Sprintf("Insufficient wallet balance. Available: %.2f, Required: %.2f", wallet.GetBalance(), body.Amount),
		})
	}

	result, err := h.investment.CreateInvestment(ctx, &ipb.CreateInvestmentRequest{
		UserId:           userData.ID,
		Code:             body.Code,
		Quantity:         body.Quantity,
		Amount:           body.Amount,
		InitialValuation: body.InitialValuation,
		Date:             body.Date,
		Description:      body.Description,
		WalletId:         body.WalletId,
	})
	if err != nil {
		logger.Error(data.LogCreateInvestmentFailed, map[string]any{
			"service":    data.InvestmentService,
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

	// Invalidate investment and dashboard caches
	h.invalidateInvestmentCaches(c.UserContext(), userData.ID)

	return c.Status(fiber.StatusCreated).JSON(dto.APIResponse{
		Status:     true,
		StatusCode: 201,
		Message:    "Investment created successfully",
		Data:       result,
	})
}

// SellInvestment — POST /investments/sell
func (h *investmentHandler) SellInvestment(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	var body dto.SellInvestmentRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.APIResponse{
			Status:     false,
			StatusCode: 400,
			Message:    data.ErrInvalidRequestBody,
		})
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	result, err := h.investment.SellInvestment(ctx, &ipb.SellInvestmentRequest{
		UserId:      userData.ID,
		AssetCode:   body.AssetCode,
		Quantity:    body.Quantity,
		Amount:      body.Amount,
		Date:        body.Date,
		Description: body.Description,
		WalletId:    body.WalletId,
	})
	if err != nil {
		logger.Error(data.LogSellInvestmentFailed, map[string]any{
			"service":    data.InvestmentService,
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

	// Invalidate investment and dashboard caches
	h.invalidateInvestmentCaches(c.UserContext(), userData.ID)

	return c.JSON(dto.APIResponse{
		Status:     true,
		StatusCode: 200,
		Message:    "Investment sold successfully",
		Data:       result,
	})
}

// GetInvestmentSummary — GET /investments/summary
func (h *investmentHandler) GetInvestmentSummary(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	cacheKey := cache.InvestmentSummary(userData.ID)

	if cached, err := h.cache.Get(c.UserContext(), cacheKey); err == nil && cached != nil {
		logger.Debug(data.LogCacheHit, map[string]any{"service": data.CacheService, "key": cacheKey})
		c.Set(data.ContentTypeHeader, data.ContentTypeJSON)
		return c.Send(cached)
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	result, err := h.investment.GetInvestmentSummary(ctx, userData.ID)
	if err != nil {
		logger.Error(data.LogGetInvestmentSummaryFailed, map[string]any{
			"service":    data.InvestmentService,
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

	resp := dto.APIResponse{
		Status:     true,
		StatusCode: 200,
		Message:    "Investment summary retrieved successfully",
		Data:       result,
	}

	if b, err := json.Marshal(resp); err == nil {
		if err := h.cache.Set(c.UserContext(), cacheKey, b, 3*cache.TTLShort/2); err != nil { // ~3 min
			logger.Warn(data.LogCacheSetFailed, map[string]any{"service": data.CacheService, "key": cacheKey, "error": err.Error()})
		}
	}

	return c.JSON(resp)
}

// GetAssetCodes — GET /investments/asset-codes
func (h *investmentHandler) GetAssetCodes(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(dto.UserData)
	requestID, _ := c.Locals(data.REQUEST_ID_LOCAL_KEY).(string)

	cacheKey := cache.InvestmentAssetCodes()

	if cached, err := h.cache.Get(c.UserContext(), cacheKey); err == nil && cached != nil {
		logger.Debug(data.LogCacheHit, map[string]any{"service": data.CacheService, "key": cacheKey})
		c.Set(data.ContentTypeHeader, data.ContentTypeJSON)
		return c.Send(cached)
	}

	ctx := interceptor.ContextWithUserData(c.UserContext(), userData)

	result, err := h.investment.GetAssetCodes(ctx)
	if err != nil {
		logger.Error(data.LogGetAssetCodesFailed, map[string]any{
			"service":    data.InvestmentService,
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

	resp := dto.APIResponse{
		Status:     true,
		StatusCode: 200,
		Message:    "Asset codes retrieved successfully",
		Data:       result,
	}

	if b, err := json.Marshal(resp); err == nil {
		if err := h.cache.Set(c.UserContext(), cacheKey, b, cache.TTLAsset); err != nil {
			logger.Warn(data.LogCacheSetFailed, map[string]any{"service": data.CacheService, "key": cacheKey, "error": err.Error()})
		}
	}

	return c.JSON(resp)
}

// invalidateInvestmentCaches clears investment and dashboard net-worth caches for a user.
func (h *investmentHandler) invalidateInvestmentCaches(ctx context.Context, userID string) {
	// Pattern-based: inv:{user_id}:*
	if err := h.cache.DeleteByPattern(ctx, cache.InvestmentAllPattern(userID)); err != nil {
		logger.Warn(data.LogCacheInvalidateFail, map[string]any{"service": data.CacheService, "pattern": cache.InvestmentAllPattern(userID), "error": err.Error()})
	}
	// Exact key: dashboard net-worth
	if err := h.cache.Delete(ctx, cache.DashboardNetWorth(userID)); err != nil {
		logger.Warn(data.LogCacheInvalidateFail, map[string]any{"service": data.CacheService, "key": cache.DashboardNetWorth(userID), "error": err.Error()})
	}
}
