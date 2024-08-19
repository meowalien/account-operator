package handlers

import (
	"account-operator/account"
	"account-operator/code"
	"github.com/gin-gonic/gin"
	"net/http"
)

type WithdrawRequest struct {
	AccountID string `json:"account_id" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
}

func Withdraw(operator account.Operator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req WithdrawRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			code.GinResponse(c, code.InvalidRequest, err.Error())
			return
		}

		err := operator.Withdraw(req.AccountID, req.Amount)
		if err != nil {
			code.GinResponse(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Withdraw successful"})
	}
}
