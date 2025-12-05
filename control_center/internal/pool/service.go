package pool

import (
	"context"
	"log"
	"strconv"

	"control_center/frontcontrolpb"
	"control_center/models"
	"control_center/pb"

	"gorm.io/gorm"
)

type Service struct {
	frontcontrolpb.UnimplementedPoolServiceServer
	DB *gorm.DB
	pm pb.PoolManagerClient
}

func New(db *gorm.DB, pm pb.PoolManagerClient) *Service {
	return &Service{
		DB: db,
		pm: pm,
	}
}

func (s *Service) CreatePool(
	ctx context.Context,
	req *frontcontrolpb.CreatePoolRequest,
) (*frontcontrolpb.CreatePoolResponse, error) {
	minVM, _ := strconv.Atoi(req.GetMinVm())
	maxVM, _ := strconv.Atoi(req.GetMaxVm())

	pool := models.Serverpool{
		UserID:       req.GetUser(),
		ServerpoolID: req.GetName(),
		ImageRef:     req.GetImage(),
		FlavorRef:    req.GetFlavor(),
		MinVM:        minVM,
		MaxVM:        maxVM,
		Networks:     models.JSONStringSlice{req.GetNetwork()},
		ConfigID:     req.GetConfig(),
	}

	log.Printf("pool to map: %v", pool.ToMap())

	rep, err := s.pm.SendRessources(
		context.Background(),
		&pb.RessourceRequest{
			User:   req.GetUser(),
			Data:   pool.ToMap(),
			Status: pb.Status_CREATE,
			Type:   pb.Type_SERVERPOOL,
		},
	)

	if err != nil || rep.GetSuccess() == false {
		return &frontcontrolpb.CreatePoolResponse{Success: false}, err
	}

	return &frontcontrolpb.CreatePoolResponse{Success: true}, nil
}

func (s *Service) DeletePool(
	ctx context.Context,
	req *frontcontrolpb.DeletePoolRequest,
) (*frontcontrolpb.DeletePoolResponse, error) {
	var pool models.Serverpool
	if err := s.DB.Where(
		"serverpool_id = ? AND user_id = ?", req.GetPoolId(), req.GetUser(),
	).First(&pool).Error; err != nil {
		return &frontcontrolpb.DeletePoolResponse{Success: false}, err
	}

	rep, err := s.pm.SendRessources(
		context.Background(),
		&pb.RessourceRequest{
			User:   req.GetUser(),
			Data:   pool.ToMap(),
			Status: pb.Status_DELETE,
			Type:   pb.Type_SERVERPOOL,
		},
	)

	if err != nil || rep.GetSuccess() == false {
		return &frontcontrolpb.DeletePoolResponse{Success: false}, err
	}

	return &frontcontrolpb.DeletePoolResponse{Success: true}, nil
}

func (s *Service) GetPool(
	ctx context.Context,
	req *frontcontrolpb.GetPoolRequest,
) (*frontcontrolpb.GetPoolResponse, error) {
	var pool models.Serverpool
	if err := s.DB.Where(
		"serverpool_id = ? AND user_id = ?", req.GetPoolId(), req.GetUser(),
	).First(&pool).Error; err != nil {
		return &frontcontrolpb.GetPoolResponse{}, err
	}

	return &frontcontrolpb.GetPoolResponse{
		Name:    pool.ServerpoolID,
		Image:   pool.ImageRef,
		Flavor:  pool.FlavorRef,
		MinVm:   int32(pool.MinVM),
		MaxVm:   int32(pool.MaxVM),
		Network: pool.Networks[0],
		Config:  pool.ConfigID,
	}, nil
}

func (s *Service) RebuildServer(
	ctx context.Context,
	req *frontcontrolpb.RebuildServerRequest,
) (*frontcontrolpb.RebuildServerResponse, error) {
	var server models.Server
	if err := s.DB.Where(
		"name = ? AND user_id = ?", req.GetServerId(), req.GetUser(),
	).First(&server).Error; err != nil {
		return &frontcontrolpb.RebuildServerResponse{Success: false}, err
	}

	data := server.ToMap()
	data["serverpool_id"] = req.GetPoolId()

	rep, err := s.pm.SendRessources(
		context.Background(),
		&pb.RessourceRequest{
			User:   req.GetUser(),
			Data:   data,
			Status: pb.Status_UPDATE,
			Type:   pb.Type_SERVER,
		},
	)

	if err != nil || !rep.GetSuccess() {
		return &frontcontrolpb.RebuildServerResponse{Success: false}, err
	}

	return &frontcontrolpb.RebuildServerResponse{Success: true}, nil
}
