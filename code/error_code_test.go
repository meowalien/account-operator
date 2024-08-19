package code

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGinResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name              string
		err               error
		additionalMessage []string
		expectedCode      int
		expectedMessage   string
	}{
		{
			name:            "InternalError",
			err:             InternalError,
			expectedCode:    http.StatusInternalServerError,
			expectedMessage: "internal error",
		},
		{
			name:            "CurrencyNotFound",
			err:             CurrencyNotFound,
			expectedCode:    http.StatusNotFound,
			expectedMessage: "Currency not found",
		},
		{
			name:            "UserIDInvalid",
			err:             UserIDInvalid,
			expectedCode:    http.StatusBadRequest,
			expectedMessage: "user_id is invalid",
		},
		{
			name:            "NoError",
			err:             nil,
			expectedCode:    200,
			expectedMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			GinResponse(c, tt.err, tt.additionalMessage...)

			assert.Equal(t, tt.expectedCode, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedMessage)
		})
	}
}

func TestHTTPCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"InternalError", InternalError, http.StatusInternalServerError},
		{"CurrencyNotFound", CurrencyNotFound, http.StatusNotFound},
		{"UserIDInvalid", UserIDInvalid, http.StatusBadRequest},
		{"NoError", nil, 200},
		{"UnknownError", errors.New("unknown error"), 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, HTTPCode(tt.err))
		})
	}
}

func TestMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"InternalError", InternalError, "internal error"},
		{"CurrencyNotFound", CurrencyNotFound, "Currency not found"},
		{"UserIDInvalid", UserIDInvalid, "user_id is invalid"},
		{"NoError", nil, ""},
		{"UnknownError", errors.New("unknown error"), "unknown error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Message(tt.err))
		})
	}
}
