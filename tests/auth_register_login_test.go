package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/BariVakhidov/sso/internal/grpc/auth"
	authService "github.com/BariVakhidov/sso/internal/services/auth"
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
	emptyAppId     = ""
	passDefaultLen = 10
)

func TestCreateApp_HappyPath(t *testing.T) {
	ctx, suite := suite.New(t)

	createAppResp, err := suite.AuthClient.CreateApp(ctx, &ssov1.CreateAppRequest{
		Name:   fmt.Sprintf("test_%s", gofakeit.LetterN(10)),
		Secret: gofakeit.LetterN(10),
	})
	require.NoError(t, err)
	assert.NotNil(t, createAppResp.GetAppId())
}

func TestFindApp_HappyPath(t *testing.T) {
	ctx, suite := suite.New(t)
	name := fmt.Sprintf("test_%s", gofakeit.LetterN(10))

	createAppResp, err := suite.AuthClient.CreateApp(ctx, &ssov1.CreateAppRequest{
		Name:   name,
		Secret: gofakeit.LetterN(10),
	})
	require.NoError(t, err)
	assert.NotNil(t, createAppResp.GetAppId())

	findAppResp, err := suite.AuthClient.App(ctx, &ssov1.AppRequest{Name: name})
	require.NoError(t, err)
	assert.Equal(t, findAppResp.GetAppId(), createAppResp.GetAppId())
}

func TestRegister_HappyPath(t *testing.T) {
	ctx, suite := suite.New(t)

	var (
		email    = fmt.Sprintf("test_%s", gofakeit.Email())
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
		email    = fmt.Sprintf("test_%s", gofakeit.Email())
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

func TestBruteforceLogin(t *testing.T) {
	ctx, suite := suite.New(t)
	appID, secret := createApp(t, suite, ctx)

	var (
		email    = fmt.Sprintf("test_%s", gofakeit.Email())
		password = generatePassword()
	)

	registerResp, err := suite.AuthClient.Register(ctx, &ssov1.RegisterRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	assert.NotNil(t, registerResp.GetUserId())

	for i := 0; i < authService.MaxFailedLoginAttempts; i++ {
		_, err := suite.AuthClient.Login(ctx, &ssov1.LoginRequest{
			Email:    email,
			Password: generatePassword(),
			AppId:    appID,
		})
		assertErrCode(t, err, codes.InvalidArgument, auth.ErrInvalidCredentials)
	}

	_, err = suite.AuthClient.Login(ctx, &ssov1.LoginRequest{
		Email:    email,
		Password: generatePassword(),
		AppId:    appID,
	})
	assertErrCode(t, err, codes.InvalidArgument, auth.ErrAccountTemporaryLocked)

	time.Sleep(authService.BaseLockoutDuration)
	assertLogin(t, ctx, email, password, appID, registerResp.GetUserId(), secret, suite)
}

func TestRegisterLogin_Login_HappyPath(t *testing.T) {
	ctx, suite := suite.New(t)
	appID, secret := createApp(t, suite, ctx)

	var (
		email    = fmt.Sprintf("test_%s", gofakeit.Email())
		password = generatePassword()
	)

	registerResp, err := suite.AuthClient.Register(ctx, &ssov1.RegisterRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	assert.NotNil(t, registerResp.GetUserId())

	assertLogin(t, ctx, email, password, appID, registerResp.GetUserId(), secret, suite)
}

func TestRegisterLogin_Login_UnHappyPath(t *testing.T) {
	ctx, suite := suite.New(t)
	appID, _ := createApp(t, suite, ctx)

	var (
		email    = fmt.Sprintf("test_%s", gofakeit.Email())
		password = generatePassword()
	)

	tests := []struct {
		name         string
		email        string
		password     string
		appID        string
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
			expectedCode: codes.InvalidArgument,
			expectedMsg:  auth.ErrInvalidCredentials,
			appID:        gofakeit.LetterN(15),
		},
		{
			name:         "Login with empty appID",
			email:        gofakeit.Email(),
			password:     generatePassword(),
			expectedCode: codes.InvalidArgument,
			expectedMsg:  auth.ErrAppIDRequired,
			appID:        "",
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

func createApp(t *testing.T, suite *suite.Suite, ctx context.Context) (appId string, secret string) {
	t.Helper()
	secret = gofakeit.LetterN(10)
	createAppResp, err := suite.AuthClient.CreateApp(ctx, &ssov1.CreateAppRequest{
		Name:   fmt.Sprintf("test_%s", gofakeit.LetterN(10)),
		Secret: secret,
	})
	require.NoError(t, err)
	assert.NotNil(t, createAppResp.GetAppId())
	return createAppResp.GetAppId(), secret
}

func assertLogin(t *testing.T, ctx context.Context, email, password, appID, userID, appSecret string, suite *suite.Suite) {
	t.Helper()
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

	assert.Equal(t, userID, claims["uid"].(string))
	assert.Equal(t, email, claims["email"].(string))
	assert.Equal(t, appID, claims["app_id"].(string))

	const deltaSeconds = 1
	assert.InDelta(t, loginTime.Add(suite.Cfg.TokenTTL).Unix(), claims["exp"].(float64), deltaSeconds)
}
