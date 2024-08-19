package handlers

import (
	"account-operator/account"
	"account-operator/code"
	"github.com/gin-gonic/gin"
	"net/http"
)

type DeleteRequest struct {
	AccountID string `json:"account_id" binding:"required"`
}

func Delete(operator account.Operator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DeleteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			code.GinResponse(c, code.InvalidRequest, err.Error())
			return
		}

		err := operator.DeleteAccount(req.AccountID)
		if err != nil {
			code.GinResponse(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Deposit successful"})
	}
}
