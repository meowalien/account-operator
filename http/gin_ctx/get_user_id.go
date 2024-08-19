package gin_ctx

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func GetUserID(c *gin.Context) (string, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", fmt.Errorf("user_id not found")
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return "", fmt.Errorf("user_id is not a string")
	}
	return userIDStr, nil
}
