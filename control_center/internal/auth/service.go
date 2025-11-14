package auth

import (
	"context"
	"control_center/frontcontrolpb"
	"control_center/models"
	"control_center/pb"
	"fmt"

	"gorm.io/gorm"
)

type Service struct {
	frontcontrolpb.UnimplementedAuthServiceServer
	DB *gorm.DB
	pm pb.PoolManagerClient
}

func New(db *gorm.DB, pm pb.PoolManagerClient) *Service {
	return &Service{DB: db, pm: pm}
}

func (s *Service) CreateUser(ctx context.Context, req *frontcontrolpb.CreateUserRequest) (*frontcontrolpb.CreateUserResponse, error) {
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return &frontcontrolpb.CreateUserResponse{
			Success: false,
			UserId:  "",
		}, fmt.Errorf("Missing required fields")
	}
	u := models.User{
		Name:     req.Username,
		Email:    req.Email,
		Password: req.Password,
	}
	if err := s.DB.Create(&u).Error; err != nil {
		return &frontcontrolpb.CreateUserResponse{
			Success: false,
			UserId:  "",
		}, fmt.Errorf("Failed to create user: %v", err)
	}
	rep, err := s.pm.SendRessources(context.Background(), &pb.RessourceRequest{
		User: u.Email,
		Data: map[string]string{
			"name":     u.Name,
			"email":    u.Email,
			"password": u.Password,
		},
		Status: pb.Status_CREATE,
		Type:   pb.Type_USER,
	})

	if rep.GetSuccess() == false || err != nil {
		return &frontcontrolpb.CreateUserResponse{
			Success: false,
			UserId:  "",
		}, fmt.Errorf("Failed to notify PoolManager: %v", err)
	}

	return &frontcontrolpb.CreateUserResponse{
		Success: true,
		UserId:  fmt.Sprintf("%d", u.ID),
	}, nil
}

func (s *Service) AuthenticateUser(ctx context.Context, req *frontcontrolpb.AuthenticateUserRequest) (*frontcontrolpb.AuthenticateUserResponse, error) {
	var user models.User
	if err := s.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &frontcontrolpb.AuthenticateUserResponse{
				Success: false,
				Token:   "",
			}, fmt.Errorf("User not found")
		}
		return &frontcontrolpb.AuthenticateUserResponse{
			Success: false,
			Token:   "",
		}, fmt.Errorf("Database error: %v", err)
	}

	if user.Password != req.Password {
		return &frontcontrolpb.AuthenticateUserResponse{
			Success: false,
			Token:   "",
		}, fmt.Errorf("Invalid password")
	}

	// Here you would normally generate a JWT or session token
	token := "dummy-token-for-user-" + fmt.Sprintf("%d", user.ID)

	return &frontcontrolpb.AuthenticateUserResponse{
		Success: true,
		Token:   token,
	}, nil
}
