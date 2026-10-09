package handler

import "github.com/gin-gonic/gin"

// RegisterRoutes wires HTTP routes for the service.
func RegisterRoutes(router *gin.Engine, deps Deps) {
	router.GET("/health", Health(deps))
	if deps.TransferService != nil {
		router.POST("/transfers", CreateTransfer(deps.TransferService))
		router.GET("/transfers/:id", GetTransfer(deps.TransferService))
	}
}
