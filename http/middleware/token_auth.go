package middleware

import (
	"account-operator/code"
	"account-operator/token"
	"github.com/gin-gonic/gin"
)

func ParseToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("token")
		if err != nil {
			code.GinResponse(c, code.TokenNotfound)
			c.Abort()
			return
		}
		claims, err := token.VerifyToken(tokenString)
		if err != nil {
			code.GinResponse(c, code.InvalidToken, err.Error())
			c.Abort()
			return
		}
		//logrus.Debug("claims: ", claims)
		c.Set("jwt_claims", claims)
		c.Next()
	}
}
