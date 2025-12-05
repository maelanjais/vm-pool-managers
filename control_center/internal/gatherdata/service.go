package gatherdata

import (
	"context"
	"errors"
	"log"

	"control_center/frontcontrolpb"
	"control_center/models"
	"control_center/pb"

	"gorm.io/gorm"
)

type Service struct {
	frontcontrolpb.UnimplementedGatherDataServiceServer
	DB *gorm.DB
	pm pb.PoolManagerClient
}

func New(pm pb.PoolManagerClient, db *gorm.DB) *Service {
	return &Service{
		pm: pm,
		DB: db,
	}
}

func (s *Service) GetAllImages(
	req *frontcontrolpb.UserRequest,
	stream frontcontrolpb.GatherDataService_GetAllImagesServer,
) error {
	rows, err := s.DB.Model(&models.Image{}).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var img models.Image
		if err := s.DB.ScanRows(rows, &img); err != nil {
			return err
		}
		if err := stream.Send(img.ToFrontControlPb()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) GetAllFlavors(
	req *frontcontrolpb.UserRequest,
	stream frontcontrolpb.GatherDataService_GetAllFlavorsServer,
) error {
	rows, err := s.DB.Model(&models.Flavor{}).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var f models.Flavor
		if err := s.DB.ScanRows(rows, &f); err != nil {
			return err
		}
		if err := stream.Send(f.ToFrontControlPb()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) GetAllNetworks(
	req *frontcontrolpb.UserRequest,
	stream frontcontrolpb.GatherDataService_GetAllNetworksServer,
) error {
	rows, err := s.DB.Model(&models.Network{}).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var n models.Network
		if err := s.DB.ScanRows(rows, &n); err != nil {
			return err
		}
		if err := stream.Send(n.ToFrontControlPb()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) GetAllServers(
	req *frontcontrolpb.UserRequest,
	stream frontcontrolpb.GatherDataService_GetAllServersServer,
) error {
	rows, err := s.DB.Model(&models.Server{}).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var n models.Server
		if err := s.DB.ScanRows(rows, &n); err != nil {
			return err
		}
		if n.UserID == req.GetUser() {
			if err := stream.Send(n.ToFrontControlPb()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) GetAllServerPools(
	req *frontcontrolpb.UserRequest,
	stream frontcontrolpb.GatherDataService_GetAllServerPoolsServer,
) error {
	rows, err := s.DB.Model(&models.Serverpool{}).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var n models.Serverpool
		if err := s.DB.ScanRows(rows, &n); err != nil {
			return err
		}
		if n.UserID == req.GetUser() {
			if err := stream.Send(n.ToFrontControlPb()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) GetAllConfigs(
	req *frontcontrolpb.UserRequest,
	stream frontcontrolpb.GatherDataService_GetAllConfigsServer,
) error {
	rows, err := s.DB.Model(&models.ConfigPool{}).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var n models.ConfigPool
		if err := s.DB.ScanRows(rows, &n); err != nil {
			return err
		}
		if n.UserID == req.GetUser() {
			if err := stream.Send(n.ToFrontControlPb()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) ExistServer(
	ctx context.Context,
	req *frontcontrolpb.UserRequest,
) (*frontcontrolpb.ExistData, error) {
	var serv models.Server
	log.Println("coucou server")

	if err := s.DB.Where("user_id = ?", req.GetUser()).First(&serv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &frontcontrolpb.ExistData{Exist: false}, nil
		}
		return nil, err
	}
	return &frontcontrolpb.ExistData{Exist: true}, nil
}

func (s *Service) ExistServerPools(
	ctx context.Context,
	req *frontcontrolpb.UserRequest,
) (*frontcontrolpb.ExistData, error) {
	var pool models.Serverpool
	log.Println("coucou serverpool")

	if err := s.DB.Where("user_id = ?", req.GetUser()).First(&pool).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &frontcontrolpb.ExistData{Exist: false}, nil
		}
		return nil, err
	}
	return &frontcontrolpb.ExistData{Exist: true}, nil
}

func (s *Service) ExistConfigs(
	ctx context.Context,
	req *frontcontrolpb.UserRequest,
) (*frontcontrolpb.ExistData, error) {
	var conf models.ConfigPool
	log.Println("coucou config")

	if err := s.DB.Where("user_id = ?", req.GetUser()).First(&conf).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &frontcontrolpb.ExistData{Exist: false}, nil
		}
		return nil, err
	}
	return &frontcontrolpb.ExistData{Exist: true}, nil
}
