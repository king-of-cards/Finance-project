package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/king-of-cards/finance-project/internal/handlers"
	"github.com/king-of-cards/finance-project/internal/middleware"
)

func New(pool *pgxpool.Pool, jwtSecret string, jwtExpiryHr int) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS())

	userHandler := handlers.NewUserHandler(pool)
	authHandler := handlers.NewAuthHandler(pool, jwtSecret, jwtExpiryHr)
	vendorHandler := handlers.NewVendorHandler(pool)
	poHandler := handlers.NewPurchaseOrderHandler(pool)
	lineItemHandler := handlers.NewLineItemHandler(pool)
	chargeHandler := handlers.NewChargeHandler(pool)

	api := r.Group("/api")
	{
		api.POST("/login", authHandler.Login)

		protected := api.Group("/")
		protected.Use(middleware.RequireAuth(pool, jwtSecret))
		{
			protected.GET("/users", userHandler.GetAllUsers)
			protected.POST("/logout", authHandler.Logout)
			protected.GET("/auth/me", authHandler.Me)
			protected.GET("/finance-users", userHandler.GetFinanceUsers)
			protected.GET("/vendors", vendorHandler.GetVendors)
			protected.GET("/vendors/:id", vendorHandler.GetVendorDetail)
			protected.GET("/vendors/:id/orders", vendorHandler.GetVendorOrders)
			protected.POST("/vendors", vendorHandler.CreateVendor)
			protected.PATCH("/vendors/:id", vendorHandler.UpdateVendor)
			protected.DELETE("/vendors/:id", vendorHandler.DeactivateVendor)
			protected.GET("/purchase-orders", poHandler.GetPurchaseOrders)
			protected.GET("/purchase-orders/:id", poHandler.GetPurchaseOrderDetail)
			protected.POST("/purchase-orders", poHandler.CreatePurchaseOrder)
			protected.GET("/purchase-orders/export", poHandler.ExportPurchaseOrders)
			protected.GET("/purchase-orders/:id/status-history", poHandler.GetPurchaseOrderStatusHistory)
			protected.POST("/purchase-orders/:id/approve", poHandler.ApprovePurchaseOrder)
			protected.POST("/purchase-orders/:id/reject", poHandler.RejectPurchaseOrder)
			protected.POST("/purchase-orders/:id/hold", poHandler.HoldPurchaseOrder)
			protected.POST("/purchase-orders/:id/request-changes", poHandler.RequestChangesPurchaseOrder)
			protected.POST("/purchase-orders/:id/mark-paid", poHandler.MarkPaidPurchaseOrder)
			protected.PATCH("/purchase-orders/:id", poHandler.UpdatePurchaseOrder)
			protected.DELETE("/purchase-orders/:id", poHandler.CancelPurchaseOrder)
			// protected.GET("/purchase-orders/:poId/line-items", lineItemHandler.GetLineItems)
			// protected.POST("/purchase-orders/:poId/line-items", lineItemHandler.AddLineItem)
			// protected.PATCH("/line-items/:id", lineItemHandler.UpdateLineItem)
			// protected.DELETE("/line-items/:id", lineItemHandler.DeleteLineItem)
			// protected.GET("/line-items/:lineItemId/charges", chargeHandler.GetLineItemCharges)
			// protected.PUT("/line-items/:lineItemId/charges", chargeHandler.ReplaceLineItemCharges)
			// protected.DELETE("/line-items/:lineItemId/charges/:chargeId", chargeHandler.DeleteLineItemCharge)
			protected.GET("/purchase-orders/:id/line-items", lineItemHandler.GetLineItems)
			protected.POST("/purchase-orders/:id/line-items", lineItemHandler.AddLineItem)
			protected.PATCH("/line-items/:id", lineItemHandler.UpdateLineItem)
			protected.DELETE("/line-items/:id", lineItemHandler.DeleteLineItem)
			protected.GET("/line-items/:id/charges", chargeHandler.GetLineItemCharges)
			protected.PUT("/line-items/:id/charges", chargeHandler.ReplaceLineItemCharges)
			protected.DELETE("/line-items/:id/charges/:chargeId", chargeHandler.DeleteLineItemCharge)

			protected.GET("/charge-types", chargeHandler.GetChargeTypes)
			protected.POST("/charge-types", chargeHandler.CreateChargeType)
			protected.PATCH("/charge-types/:id", chargeHandler.UpdateChargeType)
			protected.DELETE("/charge-types/:id", chargeHandler.DeleteChargeType)

		}
	}

	return r
}
