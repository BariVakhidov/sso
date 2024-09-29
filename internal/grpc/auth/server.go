package auth

import (
	"context"
	"errors"

	"github.com/BariVakhidov/sso/internal/services/auth"
	ssov1 "github.com/BariVakhidov/ssoprotos/gen/go/sso"
	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	emptyValue = 0
)

const (
	ErrInvalidEmail       = "invalid email format"
	ErrPasswordRequired   = "password is required"
	ErrEmailRequired      = "email is required"
	ErrUserIDRequired     = "userID is required"
	ErrUserNotFound       = "user not found"
	ErrUserExists         = "user already exists"
	ErrAppIDRequired      = "app_id is required"
	ErrInternal           = "internal error"
	ErrInvalidCredentials = "invalid credentials"
)

type Auth interface {
	Login(ctx context.Context, email string, password string, appID int32) (token string, err error)
	RegisterNewUser(ctx context.Context, email string, password string) (userID int64, err error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
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

	return &ssov1.RegisterResponse{UserId: userID}, nil
}

func (s *serverAPI) Login(ctx context.Context, req *ssov1.LoginRequest) (*ssov1.LoginResponse, error) {
	if err := s.validateLoginReq(req); err != nil {
		return nil, err
	}

	token, err := s.auth.Login(ctx, req.GetEmail(), req.GetPassword(), req.GetAppId())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, ErrInvalidCredentials)
		}

		return nil, status.Error(codes.Internal, ErrInternal)
	}

	return &ssov1.LoginResponse{Token: token}, nil
}

func (s *serverAPI) IsAdmin(ctx context.Context, req *ssov1.IsAdminRequest) (*ssov1.IsAdminResponse, error) {
	if err := s.validateIsAdminReq(req); err != nil {
		return nil, err
	}

	isAdmin, err := s.auth.IsAdmin(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, ErrUserNotFound)
		}

		return nil, status.Error(codes.Internal, ErrInternal)
	}

	return &ssov1.IsAdminResponse{IsAdmin: isAdmin}, nil
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
