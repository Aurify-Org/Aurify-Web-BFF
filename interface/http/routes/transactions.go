package routes

import (
	grpcClient "refina-web-bff/interface/grpc/client"
	"refina-web-bff/interface/http/handler"
	"refina-web-bff/interface/http/middleware"
	"refina-web-bff/internal/cache"

	"github.com/gofiber/fiber/v2"
)

func TransactionRoutes(app *fiber.App, tc grpcClient.TransactionClient, wc grpcClient.WalletClient, c cache.Cache) {
	h := handler.NewTransactionHandler(tc, wc, c)

	// Transaction CRUD routes
	transactions := app.Group("/transactions")
	transactions.Use(middleware.AuthMiddleware())

	transactions.Get("/", h.GetUserTransactions)
	transactions.Post("/", h.CreateTransaction)
	transactions.Post("/transfer", h.CreateFundTransfer)
	transactions.Put("/transfer/:transferId", h.UpdateFundTransfer)    // both legs at once
	transactions.Delete("/transfer/:transferId", h.DeleteFundTransfer) // both legs at once
	transactions.Get("/:id", h.GetTransactionByID)
	transactions.Put("/:id", h.UpdateTransaction)
	transactions.Delete("/:id", h.DeleteTransaction)

	// Category routes
	categories := app.Group("/categories")
	categories.Use(middleware.AuthMiddleware())

	categories.Get("/", h.GetCategories)

	// Attachment routes
	attachments := app.Group("/attachments")
	attachments.Use(middleware.AuthMiddleware())

	attachments.Get("/transaction/:transactionId", h.GetAttachmentsByTransaction)
	attachments.Post("/", h.CreateAttachment)
	attachments.Delete("/:id", h.DeleteAttachment)
}
