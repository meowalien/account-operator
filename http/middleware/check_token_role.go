package middleware

import (
	"account-operator/code"
	"account-operator/role"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func CheckTokenRole(expectedRole role.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exist := c.Get("jwt_claims")
		if !exist {
			code.GinResponse(c, code.InternalError)
			logrus.Error("jwt_claims not found")
			c.Abort()
			return
		}
		mapClaims, ok := claims.(jwt.MapClaims)
		if !ok {
			code.GinResponse(c, code.InternalError)
			logrus.Error("jwt_claims is invalid")
			c.Abort()
			return
		}

		roleInClaims, exist := mapClaims["roles"]
		if !exist {
			code.GinResponse(c, code.InvalidToken, "role not found in jwt_claims")
			c.Abort()
			return
		}

		if roleArray, okRoleArray := roleInClaims.([]string); okRoleArray {
			logrus.Debugf("roleArray: %v", roleArray)
			for _, r := range roleArray {
				if r == expectedRole {
					c.Next()
					return
				}
			}
			code.GinResponse(c, code.InvalidToken, "role is invalid")
			c.Abort()
			return
		}
		c.Next()
	}
}
