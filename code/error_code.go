package code

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

type errorCode struct {
	HTTPCode int
	Message  string
}

func (e errorCode) Error() string {
	return e.Message
}

var (
	InternalError    = errorCode{HTTPCode: http.StatusInternalServerError, Message: "internal error"}
	CurrencyNotFound = errorCode{HTTPCode: http.StatusNotFound, Message: "Currency not found"}
	UserIDInvalid    = errorCode{HTTPCode: http.StatusBadRequest, Message: "user_id is invalid"}
	UserIDNotfound   = errorCode{HTTPCode: http.StatusNotFound, Message: "user_id not found"}
	InvalidRequest   = errorCode{HTTPCode: http.StatusBadRequest, Message: "invalid request"}
	InvalidToken     = errorCode{HTTPCode: http.StatusUnauthorized, Message: "invalid token"}
	TokenNotfound    = errorCode{HTTPCode: http.StatusUnauthorized, Message: "token not found"}
	AccountDeleted   = errorCode{HTTPCode: http.StatusNotFound, Message: "account deleted"}
)

func GinResponse(c *gin.Context, err error, additionalMessage ...string) {
	var e errorCode
	if errors.As(err, &e) {
		if e.HTTPCode == http.StatusInternalServerError {
			logrus.Errorf("Internal error: %v", err)
			c.JSON(e.HTTPCode, gin.H{"error": e.Message})
			return
		}
		c.JSON(e.HTTPCode, gin.H{"error": fmt.Sprintf("%s %s", Message(err), strings.Join(additionalMessage, " "))})
		return
	}

	c.JSON(HTTPCode(err), gin.H{"error": fmt.Sprintf("%s %s", Message(err), strings.Join(additionalMessage, " "))})
}

func HTTPCode(err error) int {
	if err == nil {
		return http.StatusOK
	}
	var e errorCode
	if errors.As(err, &e) {
		return e.HTTPCode
	}
	return http.StatusInternalServerError
}

func Message(err error) string {
	if err == nil {
		return ""
	}
	var e errorCode
	if errors.As(err, &e) {
		return e.Message
	}
	return err.Error()
}
