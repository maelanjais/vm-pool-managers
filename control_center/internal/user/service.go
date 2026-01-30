package user

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"control_center/event"
	"control_center/frontcontrolpb"
	"control_center/models"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

type Service struct {
	frontcontrolpb.UnimplementedUserServiceServer
	DB     *gorm.DB
	Broker *event.EventBroker
}

type RessourceEvent struct {
	User      string
	Action    string
	Type      frontcontrolpb.Type
	Ressource any
}

func New(db *gorm.DB, broker *event.EventBroker) *Service {
	return &Service{
		Broker: broker,
		DB:     db,
	}
}

func (s *Service) UpdateDataUser(
	req *frontcontrolpb.UpdateDataUserRequest,
	stream frontcontrolpb.UserService_UpdateDataUserServer,
) error {
	user := req.GetUser()
	ctx := stream.Context()
	sub := s.Broker.Subscribe()
	defer s.Broker.Unsubscribe(sub)

	log.Printf("[User %s] Subscribed to DB events", user)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[User %s] Unsubscribed", user)
			return nil

		case payload := <-sub:
			var evt event.TableEvent
			if err := json.Unmarshal([]byte(payload), &evt); err != nil {
				log.Printf("Invalid event JSON: %v", err)
				continue
			}

			// Table → Type Proto
			var typ frontcontrolpb.Type
			switch evt.Table {
			case "servers":
				typ = frontcontrolpb.Type_SERVER
			case "serverpools":
				typ = frontcontrolpb.Type_SERVERPOOL
			case "config_pools":
				typ = frontcontrolpb.Type_CONFIG
			default:
				continue
			}

			// Action → Status Proto
			var status frontcontrolpb.Status
			switch evt.Action {
			case "create":
				status = frontcontrolpb.Status_CREATE
			case "update":
				status = frontcontrolpb.Status_UPDATE
			case "delete":
				status = frontcontrolpb.Status_DELETE
			default:
				status = frontcontrolpb.Status_STATUS_UNKNOWN
			}

			// Décoder les données JSON
			var data map[string]any
			if err := json.Unmarshal(evt.Data, &data); err != nil {
				log.Printf("Failed to unmarshal data: %v", err)
				continue
			}

			// Vérifier si l’événement concerne ce user
			if uid, ok := data["user_id"].(string); !ok || uid != user {
				continue
			}

			// Convertir data en map<string, string> pour gRPC
			stringData := make(map[string]string)
			for k, v := range data {
				stringData[k] = fmt.Sprintf("%v", v)
			}

			resp := &frontcontrolpb.UpdateDataUserResponse{
				User:   user,
				Status: status,
				Type:   typ,
				Data:   stringData,
			}

			// log.Println("data send : ", resp.GetData())

			if err := stream.Send(resp); err != nil {
				log.Printf("Stream send error: %v", err)
				return err
			}
		}
	}
}

func (s *Service) AddPersonalSSHKey(
	ctx context.Context,
	req *frontcontrolpb.AddPersonalSSHKeyRequest,
) (*frontcontrolpb.AddPersonnalSSHKeyResponse, error) {
	var user models.User
	if err := s.DB.Model(&models.User{}).Where(
		"Email = ?", req.GetUserId()).First(&user).Error; err != nil {
		return &frontcontrolpb.AddPersonnalSSHKeyResponse{Success: false}, err
	}
	_, _, _, _, err := ssh.ParseAuthorizedKey([]byte(req.GetPublicKey()))
	if err != nil {
		return &frontcontrolpb.AddPersonnalSSHKeyResponse{Success: false}, err
	}

	user.Keypubuser = req.GetPublicKey()
	if err := s.DB.Save(&user).Error; err != nil {
		return &frontcontrolpb.AddPersonnalSSHKeyResponse{Success: false}, err
	}

	return &frontcontrolpb.AddPersonnalSSHKeyResponse{Success: true}, nil
}
