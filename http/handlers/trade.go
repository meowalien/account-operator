package handlers

import (
	"account-operator/account"
	"account-operator/code"
	"github.com/gin-gonic/gin"
)

func TradeOrder(operator account.Operator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req account.TradeOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			code.GinResponse(c, code.InvalidRequest, err.Error())
			return
		}
		err := operator.MarketOrder(req)
		if err != nil {
			code.GinResponse(c, err)
			return
		}
	}
}
