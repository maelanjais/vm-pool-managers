package configpool

import (
	"context"

	"control_center/frontcontrolpb"
	"control_center/models"
	"control_center/pb"

	"gorm.io/gorm"
)

type Service struct {
	frontcontrolpb.UnimplementedConfigServiceServer
	pm pb.PoolManagerClient
	DB *gorm.DB
}

func New(pm pb.PoolManagerClient, db *gorm.DB) *Service {
	return &Service{
		pm: pm,
		DB: db,
	}
}

func (s *Service) GetConfig(
	ctx context.Context,
	req *frontcontrolpb.GetConfigRequest,
) (*frontcontrolpb.GetConfigResponse, error) {
	var conf models.ConfigPool
	if err := s.DB.Where(
		"user_id = ? AND name = ?", req.GetUser(), req.GetKey(),
	).First(&conf).Error; err != nil {
		return nil, err
	}

	return &frontcontrolpb.GetConfigResponse{
		Value: conf.Data,
		Key:   conf.Name,
	}, nil
}

func (s *Service) CreateConfig(
	ctx context.Context,
	req *frontcontrolpb.CreateConfigRequest,
) (*frontcontrolpb.CreateConfigResponse, error) {
	conf := models.ConfigPool{
		UserID: req.GetUser(),
		Name:   req.GetKey(),
		Data:   req.GetValue(),
	}

	ress, err := s.pm.SendRessources(
		context.Background(),
		&pb.RessourceRequest{
			User:   req.GetUser(),
			Data:   conf.ToMap(),
			Status: pb.Status_CREATE,
			Type:   pb.Type_CONFIG,
		},
	)

	if err != nil || ress.GetSuccess() == false {
		return &frontcontrolpb.CreateConfigResponse{
			Success: false,
		}, err
	}

	return &frontcontrolpb.CreateConfigResponse{
		Success: true,
	}, nil
}

func (s *Service) UpdateConfig(
	ctx context.Context,
	req *frontcontrolpb.UpdateConfigRequest,
) (*frontcontrolpb.UpdateConfigResponse, error) {
	var conf models.ConfigPool
	if err := s.DB.Where(
		"user_id = ? AND name = ?", req.GetUser(), req.GetKey(),
	).First(&conf).Error; err != nil {
		return &frontcontrolpb.UpdateConfigResponse{
			Success: false,
		}, err
	}

	conf.Data = req.GetValue()

	ress, err := s.pm.SendRessources(
		context.Background(),
		&pb.RessourceRequest{
			User:   req.GetUser(),
			Data:   conf.ToMap(),
			Status: pb.Status_UPDATE,
			Type:   pb.Type_CONFIG,
		},
	)

	if err != nil || ress.GetSuccess() == false {
		return &frontcontrolpb.UpdateConfigResponse{
			Success: false,
		}, err
	}

	return &frontcontrolpb.UpdateConfigResponse{
		Success: true,
	}, nil
}

func (s *Service) DeleteConfig(
	ctx context.Context,
	req *frontcontrolpb.DeleteConfigRequest,
) (*frontcontrolpb.DeleteConfigResponse, error) {
	var conf models.ConfigPool
	if err := s.DB.Where(
		"user_id = ? AND name = ?", req.GetUser(), req.GetKey(),
	).First(&conf).Error; err != nil {
		return &frontcontrolpb.DeleteConfigResponse{
			Success: false,
		}, err
	}

	ress, err := s.pm.SendRessources(
		context.Background(),
		&pb.RessourceRequest{
			User:   req.GetUser(),
			Data:   conf.ToMap(),
			Status: pb.Status_DELETE,
			Type:   pb.Type_CONFIG,
		},
	)

	if err != nil || ress.GetSuccess() == false {
		return &frontcontrolpb.DeleteConfigResponse{
			Success: false,
		}, err
	}

	return &frontcontrolpb.DeleteConfigResponse{
		Success: true,
	}, nil
}
