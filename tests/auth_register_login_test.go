package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/BariVakhidov/sso/internal/grpc/auth"
	"github.com/BariVakhidov/sso/tests/suite"
	ssov1 "github.com/BariVakhidov/ssoprotos/gen/go/sso"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	emptyAppId     = 0
	appID          = 1
	appSecret      = "test_secret"
	appName        = "test"
	passDefaultLen = 10
)

func TestRegister_HappyPath(t *testing.T) {
	ctx, suite := suite.New(t)

	var (
		email    = gofakeit.Email()
		password = generatePassword()
	)

	registerResp, err := suite.AuthClient.Register(ctx, &ssov1.RegisterRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	assert.NotNil(t, registerResp.GetUserId())
}

func TestRegister_Duplicate(t *testing.T) {
	ctx, suite := suite.New(t)

	var (
		email    = gofakeit.Email()
		password = generatePassword()
	)

	registerResp, err := suite.AuthClient.Register(ctx, &ssov1.RegisterRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	assert.NotNil(t, registerResp.GetUserId())

	resp, err := suite.AuthClient.Register(ctx, &ssov1.RegisterRequest{
		Email:    email,
		Password: password,
	})
	assertErrCode(t, err, codes.AlreadyExists, auth.ErrUserExists)
	assert.Empty(t, resp.GetUserId())
}

func TestRegister_UnHappyPath(t *testing.T) {
	ctx, suite := suite.New(t)

	tests := []struct {
		name         string
		email        string
		password     string
		expectedCode codes.Code
		expectedMsg  string
	}{
		{
			name:         "Register with invalid email",
			email:        gofakeit.Letter(),
			password:     generatePassword(),
			expectedCode: codes.InvalidArgument,
			expectedMsg:  auth.ErrInvalidEmail,
		},
		{
			name:         "Register with empty email",
			email:        "",
			password:     generatePassword(),
			expectedCode: codes.InvalidArgument,
			expectedMsg:  auth.ErrEmailRequired,
		},
		{
			name:         "Register with empty password",
			email:        gofakeit.Email(),
			password:     "",
			expectedCode: codes.InvalidArgument,
			expectedMsg:  auth.ErrPasswordRequired,
		},
		{
			name:         "Register with empty both",
			email:        "",
			password:     "",
			expectedCode: codes.InvalidArgument,
			expectedMsg:  auth.ErrEmailRequired,
		},
	}

	wg := &sync.WaitGroup{}
	for _, test := range tests {
		wg.Add(1)
		go func() {
			t.Run(test.name, func(t *testing.T) {
				resp, err := suite.AuthClient.Register(ctx, &ssov1.RegisterRequest{
					Email:    test.email,
					Password: test.password,
				})
				assert.Empty(t, resp.GetUserId())
				assertErrCode(t, err, test.expectedCode, test.expectedMsg)
				wg.Done()
			})
		}()
	}
	wg.Wait()
}

func TestRegisterLogin_Login_HappyPath(t *testing.T) {
	ctx, suite := suite.New(t)

	var (
		email    = gofakeit.Email()
		password = generatePassword()
	)

	registerResp, err := suite.AuthClient.Register(ctx, &ssov1.RegisterRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	assert.NotNil(t, registerResp.GetUserId())

	loginResp, err := suite.AuthClient.Login(ctx, &ssov1.LoginRequest{
		Email:    email,
		Password: password,
		AppId:    appID,
	})
	require.NoError(t, err)
	loginTime := time.Now()

	token := loginResp.GetToken()
	assert.NotNil(t, token)

	tokenParsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(appSecret), nil
	})
	require.NoError(t, err)

	claims, ok := tokenParsed.Claims.(jwt.MapClaims)
	assert.True(t, ok)

	assert.Equal(t, registerResp.GetUserId(), int64(claims["uid"].(float64)))
	assert.Equal(t, email, claims["email"].(string))
	assert.Equal(t, appID, int(claims["app_id"].(float64)))

	const deltaSeconds = 1
	assert.InDelta(t, loginTime.Add(suite.Cfg.TokenTTL).Unix(), claims["exp"].(float64), deltaSeconds)
}

func TestRegisterLogin_Login_UnHappyPath(t *testing.T) {
	ctx, suite := suite.New(t)
	var (
		email    = gofakeit.Email()
		password = generatePassword()
	)

	tests := []struct {
		name         string
		email        string
		password     string
		appID        int32
		expectedCode codes.Code
		expectedMsg  string
	}{
		{
			name:         "Login wrong email",
			email:        gofakeit.Email(),
			password:     password,
			expectedCode: codes.InvalidArgument,
			expectedMsg:  auth.ErrInvalidCredentials,
			appID:        appID,
		},
		{
			name:         "Login wrong password",
			email:        email,
			password:     generatePassword(),
			expectedCode: codes.InvalidArgument,
			expectedMsg:  auth.ErrInvalidCredentials,
			appID:        appID,
		},
		{
			name:         "Login wrong both",
			email:        gofakeit.Email(),
			password:     generatePassword(),
			expectedCode: codes.InvalidArgument,
			expectedMsg:  auth.ErrInvalidCredentials,
			appID:        appID,
		},
		{
			name:         "Login with invalid email",
			email:        gofakeit.Letter(),
			password:     generatePassword(),
			expectedCode: codes.InvalidArgument,
			expectedMsg:  auth.ErrInvalidEmail,
			appID:        appID,
		},
		{
			name:         "Login with empty email",
			email:        "",
			password:     generatePassword(),
			expectedCode: codes.InvalidArgument,
			expectedMsg:  auth.ErrEmailRequired,
			appID:        appID,
		},
		{
			name:         "Login with empty password",
			email:        gofakeit.Email(),
			password:     "",
			expectedCode: codes.InvalidArgument,
			expectedMsg:  auth.ErrPasswordRequired,
			appID:        appID,
		},
		{
			name:         "Login with empty both",
			email:        "",
			password:     "",
			expectedCode: codes.InvalidArgument,
			expectedMsg:  auth.ErrEmailRequired,
			appID:        appID,
		},
		{
			name:         "Login with wrong appID",
			email:        email,
			password:     password,
			expectedCode: codes.Internal,
			expectedMsg:  auth.ErrInternal,
			appID:        gofakeit.Int32(),
		},
		{
			name:         "Login with empty appID",
			email:        gofakeit.Email(),
			password:     generatePassword(),
			expectedCode: codes.InvalidArgument,
			expectedMsg:  auth.ErrAppIDRequired,
			appID:        emptyAppId,
		},
	}

	registerResp, err := suite.AuthClient.Register(ctx, &ssov1.RegisterRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	assert.NotNil(t, registerResp.GetUserId())

	wg := &sync.WaitGroup{}
	for _, test := range tests {
		wg.Add(1)
		go func() {
			t.Run(test.name, func(t *testing.T) {
				resp, err := suite.AuthClient.Login(ctx, &ssov1.LoginRequest{
					Email:    test.email,
					Password: test.password,
					AppId:    test.appID,
				})
				assert.Empty(t, resp.GetToken())
				assertErrCode(t, err, test.expectedCode, test.expectedMsg)
				wg.Done()
			})
		}()
	}
	wg.Wait()
}

func generatePassword() string {
	return gofakeit.Password(true, false, true, true, true, passDefaultLen)
}

func assertErrCode(t *testing.T, err error, targetCode codes.Code, msg string) {
	t.Helper()
	require.Error(t, err)
	code, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, targetCode, code.Code())
	assert.Equal(t, msg, code.Message())
}