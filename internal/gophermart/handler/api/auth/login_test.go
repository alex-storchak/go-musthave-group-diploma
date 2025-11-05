package handlerauth

import (
	"context"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/api/auth/mocks"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type handleLoginTestCase struct {
	name           string
	requestBody    string
	mockSetup      func(*mocks.MockUserLoginer)
	expectedStatus int
	expectCookie   bool
	cookieValue    string
}

func TestHandleLogin(t *testing.T) {
	cfg := &config.Config{
		AuthConfig: config.AuthConfig{
			AuthCookieName:     config.DefaultAuthCookieName,
			AuthSecretKey:      config.DefaultAuthSecretKey,
			AuthExpireDuration: config.DefaultAuthExpireDuration,
		},
	}

	logger := zap.NewNop()

	tests := []handleLoginTestCase{
		{
			name:        "successful login",
			requestBody: `{"login": "testuser", "password": "testpass"}`,
			mockSetup: func(m *mocks.MockUserLoginer) {
				m.EXPECT().
					Login(mock.Anything, "testuser", "testpass").
					Return(&models.User{ID: 1, Login: "testuser"}, "login-token", nil).
					Once()
			},
			expectedStatus: http.StatusOK,
			expectCookie:   true,
			cookieValue:    "login-token",
		},
		{
			name:        "invalid credentials",
			requestBody: `{"login": "wronguser", "password": "wrongpass"}`,
			mockSetup: func(m *mocks.MockUserLoginer) {
				m.EXPECT().
					Login(mock.Anything, "wronguser", "wrongpass").
					Return(nil, "", service.ErrInvalidCredentials).
					Once()
			},
			expectedStatus: http.StatusUnauthorized,
			expectCookie:   false,
		},
		{
			name:        "internal server error",
			requestBody: `{"login": "testuser", "password": "testpass"}`,
			mockSetup: func(m *mocks.MockUserLoginer) {
				m.EXPECT().
					Login(mock.Anything, "testuser", "testpass").
					Return(nil, "", assert.AnError).
					Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectCookie:   false,
		},
		{
			name:           "invalid JSON",
			requestBody:    `{"login": "testuser", "password": }`,
			mockSetup:      func(m *mocks.MockUserLoginer) {},
			expectedStatus: http.StatusBadRequest,
			expectCookie:   false,
		},
		{
			name:           "empty login",
			requestBody:    `{"login": "", "password": "testpass"}`,
			mockSetup:      func(m *mocks.MockUserLoginer) {},
			expectedStatus: http.StatusBadRequest,
			expectCookie:   false,
		},
		{
			name:           "empty password",
			requestBody:    `{"login": "testuser", "password": ""}`,
			mockSetup:      func(m *mocks.MockUserLoginer) {},
			expectedStatus: http.StatusBadRequest,
			expectCookie:   false,
		},
		{
			name:           "missing login field",
			requestBody:    `{"password": "testpass"}`,
			mockSetup:      func(m *mocks.MockUserLoginer) {},
			expectedStatus: http.StatusBadRequest,
			expectCookie:   false,
		},
		{
			name:           "missing password field",
			requestBody:    `{"login": "testuser"}`,
			mockSetup:      func(m *mocks.MockUserLoginer) {},
			expectedStatus: http.StatusBadRequest,
			expectCookie:   false,
		},
		{
			name:        "context cancellation error",
			requestBody: `{"login": "testuser", "password": "testpass"}`,
			mockSetup: func(m *mocks.MockUserLoginer) {
				m.EXPECT().
					Login(mock.Anything, "testuser", "testpass").
					Return(nil, "", context.Canceled)
			},
			expectedStatus: http.StatusInternalServerError,
			expectCookie:   false,
		},
	}

	//nolint:dupl // test structure possible the same for login and register
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockLoginer := mocks.NewMockUserLoginer(t)
			tt.mockSetup(mockLoginer)

			handler := HandleLogin(cfg, logger, mockLoginer)

			// Execute
			req := httptest.NewRequest("POST", "/login", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Assert
			assertTestHandleLogin(t, &tt, rr, cfg)
		})
	}
}

func assertTestHandleLogin(
	t *testing.T,
	tt *handleLoginTestCase,
	rr *httptest.ResponseRecorder,
	cfg *config.Config,
) {
	assert.Equal(t, tt.expectedStatus, rr.Code)
	if tt.expectCookie {
		cookies := rr.Result().Cookies()
		assert.Greater(t, len(cookies), 0, "Should have cookies")

		authCookieFound := false
		for _, cookie := range cookies {
			if cookie.Name == cfg.AuthCookieName {
				authCookieFound = true
				assert.Equal(t, tt.cookieValue, cookie.Value)
				break
			}
		}
		assert.True(t, authCookieFound)
	} else {
		cookies := rr.Result().Cookies()
		authCookieFound := false
		for _, cookie := range cookies {
			if cookie.Name == cfg.AuthCookieName {
				authCookieFound = true
				break
			}
		}
		assert.False(t, authCookieFound)
	}
}
