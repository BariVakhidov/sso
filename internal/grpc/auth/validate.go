package auth

import (
	ssov1 "github.com/BariVakhidov/ssoprotos/gen/go/sso"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ServerAPI) validateLoginReq(req *ssov1.LoginRequest) error {
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

func (s *ServerAPI) validateRegisterReq(req *ssov1.RegisterRequest) error {
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

func (s *ServerAPI) validateIsAdminReq(req *ssov1.IsAdminRequest) error {
	if req.GetUserId() == emptyValue {
		return status.Error(codes.InvalidArgument, ErrUserIDRequired)
	}

	return nil
}

func (s *ServerAPI) validateCreateAppReq(req *ssov1.CreateAppRequest) error {
	if req.GetName() == emptyValue {
		return status.Error(codes.InvalidArgument, ErrAppNameRequired)
	}

	if req.GetSecret() == emptyValue {
		return status.Error(codes.InvalidArgument, ErrAppSecretRequired)
	}

	return nil
}

func (s *ServerAPI) validateAppReq(req *ssov1.AppRequest) error {
	if req.GetName() == emptyValue {
		return status.Error(codes.InvalidArgument, ErrAppNameRequired)
	}

	return nil
}
