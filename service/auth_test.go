package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/calvinchengx/gin-go-pg/apperr"
	"github.com/calvinchengx/gin-go-pg/mock"
	"github.com/calvinchengx/gin-go-pg/mock/mockdb"
	"github.com/calvinchengx/gin-go-pg/model"
	"github.com/calvinchengx/gin-go-pg/repository/auth"
	"github.com/calvinchengx/gin-go-pg/service"
	"github.com/gin-gonic/gin"

	"github.com/stretchr/testify/assert"
)

func TestLogin(t *testing.T) {
	cases := []struct {
		name       string
		req        string
		wantStatus int
		wantResp   *model.AuthToken
		userRepo   *mockdb.User
		jwt        *mock.JWT
	}{
		{
			name:       "Invalid request",
			req:        `{"username":"juzernejm"}`,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "Fail on FindByUsername",
			req:        `{"username":"juzernejm","password":"hunter123"}`,
			wantStatus: http.StatusInternalServerError,
			userRepo: &mockdb.User{
				FindByUsernameFn: func(context.Context, string) (*model.User, error) {
					return nil, apperr.DB
				},
			},
		},
		{
			name:       "Success",
			req:        `{"username":"juzernejm","password":"hunter123"}`,
			wantStatus: http.StatusOK,
			userRepo: &mockdb.User{
				FindByUsernameFn: func(context.Context, string) (*model.User, error) {
					return &model.User{
						Password: auth.HashPassword("hunter123"),
						Active:   true,
					}, nil
				},
				UpdateLoginFn: func(context.Context, *model.User) error {
					return nil
				},
			},
			jwt: &mock.JWT{
				GenerateTokenFn: func(*model.User) (string, string, error) {
					return "jwttokenstring", mock.TestTime(2018).Format(time.RFC3339), nil
				},
			},
			wantResp: &model.AuthToken{Token: "jwttokenstring", Expires: mock.TestTime(2018).Format(time.RFC3339)},
		},
	}
	gin.SetMode(gin.TestMode)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			authService := auth.NewAuthService(tt.userRepo, tt.jwt)
			service.AuthRouter(authService, r)
			ts := httptest.NewServer(r)
			defer ts.Close()
			path := ts.URL + "/login"
			res, err := http.Post(path, "application/json", bytes.NewBufferString(tt.req))
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if tt.wantResp != nil {
				response := new(model.AuthToken)
				if err := json.NewDecoder(res.Body).Decode(response); err != nil {
					t.Fatal(err)
				}
				tt.wantResp.RefreshToken = response.RefreshToken
				assert.Equal(t, tt.wantResp, response)
			}
			assert.Equal(t, tt.wantStatus, res.StatusCode)
		})
	}
}

func TestRefresh(t *testing.T) {
	cases := []struct {
		name       string
		req        string
		wantStatus int
		wantResp   *model.RefreshToken
		userRepo   *mockdb.User
		jwt        *mock.JWT
	}{
		{
			name:       "Fail on FindByToken",
			req:        "refreshtoken",
			wantStatus: http.StatusInternalServerError,
			userRepo: &mockdb.User{
				FindByTokenFn: func(context.Context, string) (*model.User, error) {
					return nil, apperr.DB
				},
			},
		},
		{
			name:       "Success",
			req:        "refreshtoken",
			wantStatus: http.StatusOK,
			userRepo: &mockdb.User{
				FindByTokenFn: func(context.Context, string) (*model.User, error) {
					return &model.User{
						Username: "johndoe",
						Active:   true,
					}, nil
				},
			},
			jwt: &mock.JWT{
				GenerateTokenFn: func(*model.User) (string, string, error) {
					return "jwttokenstring", mock.TestTime(2018).Format(time.RFC3339), nil
				},
			},
			wantResp: &model.RefreshToken{Token: "jwttokenstring", Expires: mock.TestTime(2018).Format(time.RFC3339)},
		},
	}
	gin.SetMode(gin.TestMode)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			authService := auth.NewAuthService(tt.userRepo, tt.jwt)
			service.AuthRouter(authService, r)
			ts := httptest.NewServer(r)
			defer ts.Close()
			path := ts.URL + "/refresh/" + tt.req
			res, err := http.Get(path)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if tt.wantResp != nil {
				response := new(model.RefreshToken)
				if err := json.NewDecoder(res.Body).Decode(response); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, tt.wantResp, response)
			}
			assert.Equal(t, tt.wantStatus, res.StatusCode)
		})
	}
}