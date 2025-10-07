package main

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/internal"
	"PoolManagerVM/backend/internal/worker"
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/routes"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {

	// loading .env
	config.LoadEnvConfig()
	models.CreateParams()

	// creating context to stop cleanly
	ctx, cancel := context.WithCancel(context.Background())

	//starting database
	config.Start_DB()
	go config.Sync_DB(ctx)

	//configuring gin server
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	routes.UserRoutes(r)
	routes.ServerpoolRoutes(r)
	routes.LoginRoutes(r)
	routes.WebSocketRoutes(r)

	//preparing workers
	var wg sync.WaitGroup
	worker.LaunchWorkers(5, &wg, ctx)

	// 	//starting goroutines
	go internal.Monitor(ctx)

	//starting server gin in go routine
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()
	log.Println("Server started on port 8080")

	// bloc instruction to shutdown cleanly
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown signal received")
	cancel()
	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTimeout()
	if err := srv.Shutdown(ctxTimeout); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	wg.Wait()

	log.Println("Program exited cleanly")
}
