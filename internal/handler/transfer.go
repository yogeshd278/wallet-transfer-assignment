package handler

import (
	"errors"
	"net/http"

	"wallet_transfer_service/internal/domain"
	"wallet_transfer_service/internal/service"

	"github.com/gin-gonic/gin"
)

type createTransferRequest struct {
	IdempotencyKey string `json:"idempotencyKey"`
	FromWalletID   string `json:"fromWalletId"`
	ToWalletID     string `json:"toWalletId"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
}

const maxIdempotencyKeyLen = 128

// CreateTransfer handles POST /transfers.
func CreateTransfer(svc *service.TransferService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createTransferRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, service.ErrorResponse{Error: "invalid JSON body"})
			return
		}

		key := req.IdempotencyKey
		if headerKey := c.GetHeader("Idempotency-Key"); headerKey != "" {
			key = headerKey
		}
		if key == "" {
			c.JSON(http.StatusBadRequest, service.ErrorResponse{Error: "idempotencyKey is required"})
			return
		}
		if len(key) > maxIdempotencyKeyLen {
			c.JSON(http.StatusBadRequest, service.ErrorResponse{Error: "idempotencyKey too long"})
			return
		}
		if req.FromWalletID == "" || req.ToWalletID == "" || req.Currency == "" || req.Amount == 0 {
			c.JSON(http.StatusBadRequest, service.ErrorResponse{Error: "missing required fields"})
			return
		}

		result, err := svc.CreateTransfer(c.Request.Context(), service.CreateTransferInput{
			IdempotencyKey: key,
			FromWalletID:   req.FromWalletID,
			ToWalletID:     req.ToWalletID,
			Amount:         req.Amount,
			Currency:       req.Currency,
		})
		if err != nil {
			writeServiceError(c, err)
			return
		}

		c.Data(result.StatusCode, "application/json", result.Body)
	}
}

// GetTransfer handles GET /transfers/:id.
func GetTransfer(svc *service.TransferService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		t, err := svc.GetTransfer(c.Request.Context(), id)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		body, status, err := service.MarshalTransferResponse(t)
		if err != nil {
			c.JSON(http.StatusInternalServerError, service.ErrorResponse{Error: "failed to encode response"})
			return
		}
		_ = status
		c.Data(http.StatusOK, "application/json", body)
	}
}

func writeServiceError(c *gin.Context, err error) {
	var ve *domain.ValidationError
	switch {
	case errors.As(err, &ve):
		c.JSON(http.StatusUnprocessableEntity, service.ErrorResponse{Error: ve.Error()})
	case errors.Is(err, domain.ErrSameWallet):
		c.JSON(http.StatusUnprocessableEntity, service.ErrorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrIdempotencyConflict):
		c.JSON(http.StatusConflict, service.ErrorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrTransferNotFound), errors.Is(err, domain.ErrWalletNotFound):
		c.JSON(http.StatusNotFound, service.ErrorResponse{Error: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, service.ErrorResponse{Error: "internal error"})
	}
}
