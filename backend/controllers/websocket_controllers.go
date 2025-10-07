package controllers

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // à restreindre plus tard
	},
}

// Hub des connexions
type WebSocketHub struct {
	clients map[string]*websocket.Conn
	mu      sync.Mutex
}

// Instance globale du hub
var Hub = WebSocketHub{
	clients: make(map[string]*websocket.Conn),
}

func HandleWebSocket(c *gin.Context) {
	userID := c.GetString("user_id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println("Erreur upgrade:", err)
		return
	}

	Hub.mu.Lock()
	Hub.clients[userID] = conn
	Hub.mu.Unlock()
	fmt.Printf("🟢 Connexion WebSocket ouverte pour %s\n", userID)

	defer func() {
		conn.Close()
		Hub.mu.Lock()
		delete(Hub.clients, userID)
		Hub.mu.Unlock()
		fmt.Printf("🔴 Connexion fermée pour %s\n", userID)
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Erreur lecture:", err)
			break
		}
		fmt.Printf("📨 Message de %s: %s\n", userID, msg)
	}
}

// Envoi d’un message à un utilisateur
func SendMessageToUser(userID, message string) {
	Hub.mu.Lock()
	defer Hub.mu.Unlock()

	if conn, ok := Hub.clients[userID]; ok {
		conn.WriteMessage(websocket.TextMessage, []byte(message))
	}
}
