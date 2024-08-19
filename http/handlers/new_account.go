package handlers

import (
	"account-operator/account"
	"account-operator/code"
	"account-operator/http/gin_ctx"
	"github.com/gin-gonic/gin"
	"net/http"
)

type NewAccountRequest struct {
	Currency string `json:"currency" binding:"required"`
	Name     string `json:"name" binding:"required"`
}

func NewAccount(operator account.Operator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req NewAccountRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			code.GinResponse(c, code.InvalidRequest)
			return
		}

		userIDStr, err := gin_ctx.GetUserID(c)
		if err != nil {
			code.GinResponse(c, code.UserIDInvalid, err.Error())
			return
		}

		accountInst, err := operator.CreateAccount(userIDStr, req.Currency, req.Name)
		if err != nil {
			code.GinResponse(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"account": accountInst.ID()})
	}
}
