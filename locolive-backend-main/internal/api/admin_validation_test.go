package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const testUserUUID = "cba7f0e2-7253-4957-a417-b69dd33cbff1"

func TestAdminUserActionValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validActions := []string{"ban", "unban", "revoke_sessions", "soft_delete", "promote_admin", "demote_user"}
	invalidActions := []string{"", "BAN", "delete", "promote", "  ban  ", "ban\n"}

	for _, action := range validActions {
		t.Run("valid_"+action, func(t *testing.T) {
			body, _ := json.Marshal(gin.H{"action": action})
			req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			var reqBody adminUserActionRequest
			err := c.ShouldBindJSON(&reqBody)
			require.NoError(t, err, "action=%q should be valid", action)
			require.Equal(t, action, reqBody.Action)
		})
	}

	for _, action := range invalidActions {
		t.Run("invalid_"+action, func(t *testing.T) {
			body, _ := json.Marshal(gin.H{"action": action})
			req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			var reqBody adminUserActionRequest
			err := c.ShouldBindJSON(&reqBody)
			require.Error(t, err, "action=%q should be invalid", action)
		})
	}

	// Test with no Content-Type
	t.Run("no_content_type", func(t *testing.T) {
		body, _ := json.Marshal(gin.H{"action": "ban"})
		req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
		// No Content-Type header
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		var reqBody adminUserActionRequest
		err := c.ShouldBindJSON(&reqBody)
		// Gin v1.11.0 may or may not require Content-Type
		t.Logf("ShouldBindJSON without Content-Type: err=%v", err)
	})
}
