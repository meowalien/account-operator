package middleware

import (
	"account-operator/code"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

func ParseUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		jet_claim, exist := c.Get("jwt_claims")
		if !exist {
			code.GinResponse(c, code.InternalError, "jwt_claims not found")
			c.Abort()
			return
		}

		claims, ok := jet_claim.(jwt.MapClaims)
		if !ok {
			code.GinResponse(c, code.InternalError, "jwt_claims is invalid")
			c.Abort()
			return
		}
		userString := claims["user_id"]
		if userString == nil {
			code.GinResponse(c, code.InvalidToken, "user_id not found in jwt_claims")
			c.Abort()
			return
		}

		userID, ok := userString.(string)
		if !ok {
			code.GinResponse(c, code.InvalidToken, "user_id in claims is not a string")
			c.Abort()
		}
		c.Set("user_id", userID)
		c.Next()
	}
}
