package auth

import (
	"context"
	"errors"

	"github.com/BariVakhidov/sso/internal/domain/models"
	"github.com/BariVakhidov/sso/internal/services/auth"
	ssov1 "github.com/BariVakhidov/ssoprotos/gen/go/sso"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	emptyValue = ""
)

type AuthService interface {
	Login(ctx context.Context, email string, password string, appID uuid.UUID) (token string, err error)
	RegisterNewUser(ctx context.Context, email string, password string) (userID uuid.UUID, err error)
	IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
	CreateApp(ctx context.Context, name, secret string) (uuid.UUID, error)
	App(ctx context.Context, name string) (models.App, error)
}

type ServerAPI struct {
	validator   *validator.Validate
	authService AuthService
	ssov1.UnimplementedAuthServer
}

func InitializeServerAPI(authService AuthService) *ServerAPI {
	return &ServerAPI{
		authService: authService,
		validator:   validator.New(),
	}
}

func (s *ServerAPI) Register(ctx context.Context, req *ssov1.RegisterRequest) (*ssov1.RegisterResponse, error) {
	if err := s.validateRegisterReq(req); err != nil {
		return nil, err
	}

	userID, err := s.authService.RegisterNewUser(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		if errors.Is(err, auth.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, ErrUserExists)
		}

		return nil, status.Error(codes.Internal, ErrInternal)
	}

	return &ssov1.RegisterResponse{UserId: userID.String()}, nil
}

func (s *ServerAPI) Login(ctx context.Context, req *ssov1.LoginRequest) (*ssov1.LoginResponse, error) {
	if err := s.validateLoginReq(req); err != nil {
		return nil, err
	}

	appId, err := uuid.Parse(req.GetAppId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrInvalidCredentials)
	}

	token, err := s.authService.Login(ctx, req.GetEmail(), req.GetPassword(), appId)
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

func (s *ServerAPI) IsAdmin(ctx context.Context, req *ssov1.IsAdminRequest) (*ssov1.IsAdminResponse, error) {
	if err := s.validateIsAdminReq(req); err != nil {
		return nil, err
	}

	userId, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrInvalidCredentials)
	}

	isAdmin, err := s.authService.IsAdmin(ctx, userId)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, ErrUserNotFound)
		}

		return nil, status.Error(codes.Internal, ErrInternal)
	}

	return &ssov1.IsAdminResponse{IsAdmin: isAdmin}, nil
}

func (s *ServerAPI) CreateApp(ctx context.Context, req *ssov1.CreateAppRequest) (*ssov1.CreateAppResponse, error) {
	if err := s.validateCreateAppReq(req); err != nil {
		return nil, err
	}

	appID, err := s.authService.CreateApp(ctx, req.GetName(), req.GetSecret())
	if err != nil {
		if errors.Is(err, auth.ErrAppExists) {
			return nil, status.Error(codes.AlreadyExists, ErrAppExists)
		}

		return nil, status.Error(codes.Internal, ErrInternal)
	}

	return &ssov1.CreateAppResponse{AppId: appID.String()}, nil
}

func (s *ServerAPI) App(ctx context.Context, req *ssov1.AppRequest) (*ssov1.AppResponse, error) {
	if err := s.validateAppReq(req); err != nil {
		return nil, err
	}

	app, err := s.authService.App(ctx, req.GetName())
	if err != nil {
		if errors.Is(err, auth.ErrAppNotFound) {
			return nil, status.Error(codes.NotFound, ErrAppNotFound)
		}

		return nil, status.Error(codes.Internal, ErrInternal)
	}

	return &ssov1.AppResponse{AppId: app.ID.String(), Name: app.Name}, nil
}
