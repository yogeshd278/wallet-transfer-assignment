package handler

import "wallet_transfer_service/internal/service"

// Deps holds optional infrastructure dependencies for HTTP handlers.
type Deps struct {
	DBPing          func() error
	TransferService *service.TransferService
}
