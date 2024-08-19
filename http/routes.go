package http

import (
	"account-operator/account"
	"account-operator/http/handlers"
	"account-operator/http/middleware"
	"account-operator/role"
	"github.com/gin-gonic/gin"
	"net/http"
)

func SetupRouter(operator account.Operator) (*gin.Engine, error) {
	r := gin.Default()
	err := r.SetTrustedProxies(nil)
	if err != nil {
		return nil, err
	}

	// Configure CORS
	r.Use(middleware.Cors())

	r.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	accountGroup := r.Group("/account")
	{
		accountGroup.POST("/new", middleware.ParseToken(), middleware.CheckTokenRole(role.Admin), middleware.ParseUserID(), handlers.NewAccount(operator))
		accountGroup.GET("/list", middleware.ParseToken(), middleware.CheckTokenRole(role.Admin), middleware.ParseUserID(), handlers.ListAccount(operator))
	}

	tradeGroup := r.Group("/trade")
	{
		tradeGroup.POST("/withdraw", middleware.ParseToken(), middleware.CheckTokenRole(role.Admin), middleware.ParseUserID(), handlers.Withdraw(operator))
		tradeGroup.POST("/deposit", middleware.ParseToken(), middleware.CheckTokenRole(role.Admin), middleware.ParseUserID(), handlers.Deposit(operator))
		tradeGroup.POST("/delete", middleware.ParseToken(), middleware.CheckTokenRole(role.Admin), middleware.ParseUserID(), handlers.Delete(operator))
		tradeGroup.POST("/order", middleware.ParseToken(), middleware.CheckTokenRole(role.Admin), middleware.ParseUserID(), handlers.TradeOrder(operator))
	}

	return r, nil
}
