package auth

import (
	"context"
	"errors"

	"github.com/BariVakhidov/sso/internal/domain/models"
	"github.com/BariVakhidov/sso/internal/services/auth"
	ssov1 "github.com/BariVakhidov/ssoprotos/gen/go/sso"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	emptyValue = ""
)

const (
	ErrInvalidEmail           = "invalid email format"
	ErrPasswordRequired       = "password is required"
	ErrEmailRequired          = "email is required"
	ErrUserIDRequired         = "userID is required"
	ErrUserNotFound           = "user not found"
	ErrUserExists             = "user already exists"
	ErrAppExists              = "app already exists"
	ErrAppNotFound            = "app not found"
	ErrAppNameRequired        = "app name required"
	ErrAppSecretRequired      = "app secret required"
	ErrAppIDRequired          = "app_id is required"
	ErrInternal               = "internal error"
	ErrInvalidCredentials     = "invalid credentials"
	ErrAccountTemporaryLocked = "account is temporary locked"
)

type Auth interface {
	Login(ctx context.Context, email string, password string, appID uuid.UUID) (token string, err error)
	RegisterNewUser(ctx context.Context, email string, password string) (userID uuid.UUID, err error)
	IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
	CreateApp(ctx context.Context, name, secret string) (uuid.UUID, error)
	App(ctx context.Context, name string) (models.App, error)
}

type serverAPI struct {
	validator *validator.Validate
	auth      Auth
	ssov1.UnimplementedAuthServer
}

func Register(gRPC *grpc.Server, auth Auth) {
	ssov1.RegisterAuthServer(gRPC, &serverAPI{auth: auth, validator: validator.New()})
}

func (s *serverAPI) Register(ctx context.Context, req *ssov1.RegisterRequest) (*ssov1.RegisterResponse, error) {
	if err := s.validateRegisterReq(req); err != nil {
		return nil, err
	}

	userID, err := s.auth.RegisterNewUser(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		if errors.Is(err, auth.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, ErrUserExists)
		}

		return nil, status.Error(codes.Internal, ErrInternal)
	}

	return &ssov1.RegisterResponse{UserId: userID.String()}, nil
}

func (s *serverAPI) Login(ctx context.Context, req *ssov1.LoginRequest) (*ssov1.LoginResponse, error) {
	if err := s.validateLoginReq(req); err != nil {
		return nil, err
	}

	appId, err := uuid.Parse(req.GetAppId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrInvalidCredentials)
	}

	token, err := s.auth.Login(ctx, req.GetEmail(), req.GetPassword(), appId)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, ErrInvalidCredentials)
		}

		if errors.Is(err, auth.ErrAccountIsLocked) {
			return nil, status.Error(codes.InvalidArgument, ErrAccountTemporaryLocked)
		}

		return nil, status.Error(codes.Internal, ErrInternal)
	}

	return &ssov1.LoginResponse{Token: token}, nil
}

func (s *serverAPI) IsAdmin(ctx context.Context, req *ssov1.IsAdminRequest) (*ssov1.IsAdminResponse, error) {
	if err := s.validateIsAdminReq(req); err != nil {
		return nil, err
	}

	userId, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrInvalidCredentials)
	}

	isAdmin, err := s.auth.IsAdmin(ctx, userId)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, ErrUserNotFound)
		}

		return nil, status.Error(codes.Internal, ErrInternal)
	}

	return &ssov1.IsAdminResponse{IsAdmin: isAdmin}, nil
}

func (s *serverAPI) CreateApp(ctx context.Context, req *ssov1.CreateAppRequest) (*ssov1.CreateAppResponse, error) {
	if err := s.validateCreateAppReq(req); err != nil {
		return nil, err
	}

	appID, err := s.auth.CreateApp(ctx, req.GetName(), req.GetSecret())
	if err != nil {
		if errors.Is(err, auth.ErrAppExists) {
			return nil, status.Error(codes.AlreadyExists, ErrAppExists)
		}

		return nil, status.Error(codes.Internal, ErrInternal)
	}

	return &ssov1.CreateAppResponse{AppId: appID.String()}, nil
}

func (s *serverAPI) App(ctx context.Context, req *ssov1.AppRequest) (*ssov1.AppResponse, error) {
	if err := s.validateAppReq(req); err != nil {
		return nil, err
	}

	app, err := s.auth.App(ctx, req.GetName())
	if err != nil {
		if errors.Is(err, auth.ErrAppNotFound) {
			return nil, status.Error(codes.NotFound, ErrAppNotFound)
		}

		return nil, status.Error(codes.Internal, ErrInternal)
	}

	return &ssov1.AppResponse{AppId: app.ID.String(), Name: app.Name}, nil
}

func (s *serverAPI) validateLoginReq(req *ssov1.LoginRequest) error {
	if req.GetEmail() == "" {
		return status.Error(codes.InvalidArgument, ErrEmailRequired)
	}

	if err := s.validator.Var(req.GetEmail(), "email"); err != nil {
		return status.Error(codes.InvalidArgument, ErrInvalidEmail)
	}

	if req.GetPassword() == "" {
		return status.Error(codes.InvalidArgument, ErrPasswordRequired)
	}

	if req.GetAppId() == emptyValue {
		return status.Error(codes.InvalidArgument, ErrAppIDRequired)
	}

	return nil
}

func (s *serverAPI) validateRegisterReq(req *ssov1.RegisterRequest) error {
	if req.GetEmail() == "" {
		return status.Error(codes.InvalidArgument, ErrEmailRequired)
	}

	if err := s.validator.Var(req.GetEmail(), "email"); err != nil {
		return status.Error(codes.InvalidArgument, ErrInvalidEmail)
	}

	if req.GetPassword() == "" {
		return status.Error(codes.InvalidArgument, ErrPasswordRequired)
	}

	return nil
}

func (s *serverAPI) validateIsAdminReq(req *ssov1.IsAdminRequest) error {
	if req.GetUserId() == emptyValue {
		return status.Error(codes.InvalidArgument, ErrUserIDRequired)
	}

	return nil
}

func (s *serverAPI) validateCreateAppReq(req *ssov1.CreateAppRequest) error {
	if req.GetName() == emptyValue {
		return status.Error(codes.InvalidArgument, ErrAppNameRequired)
	}

	if req.GetSecret() == emptyValue {
		return status.Error(codes.InvalidArgument, ErrAppSecretRequired)
	}

	return nil
}

func (s *serverAPI) validateAppReq(req *ssov1.AppRequest) error {
	if req.GetName() == emptyValue {
		return status.Error(codes.InvalidArgument, ErrAppNameRequired)
	}

	return nil
}
