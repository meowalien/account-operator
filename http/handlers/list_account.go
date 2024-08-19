package handlers

import (
	"account-operator/account"
	"account-operator/code"
	"account-operator/http/gin_ctx"
	"github.com/gin-gonic/gin"
	"net/http"
)

func ListAccount(operator account.Operator) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, err := gin_ctx.GetUserID(c)
		if err != nil {
			code.GinResponse(c, code.UserIDInvalid, err.Error())
			return
		}

		accountInstSlice, err := operator.ListAccount(userIDStr)
		if err != nil {
			code.GinResponse(c, err)
			return
		}
		result := make([]gin.H, len(accountInstSlice))

		for i, i2 := range accountInstSlice {
			result[i] = gin.H{
				"id":       i2.ID(),
				"currency": i2.Currency(),
				"name":     i2.Name(),
			}
		}
		c.JSON(http.StatusOK, result)
	}
}
