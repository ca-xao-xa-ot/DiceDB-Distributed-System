package internal

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"dice-dashboard/handlers"
	"dice-dashboard/services"
)

func SetupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.SetTrustedProxies(nil)
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./static")

	kvStore := services.NewKVStoreFromEnv()
	clusterService := services.NewClusterService(kvStore, 10*time.Second)
	clusterService.StartHeartbeatSimulation()

	pageHandler := handlers.NewPageHandler(clusterService)
	apiHandler := handlers.NewAPIHandler(clusterService)

	router.GET("/", pageHandler.Index)

	api := router.Group("/api")
	{
		api.GET("/overview", apiHandler.GetOverview)
		api.GET("/nodes", apiHandler.GetNodes)
		api.GET("/heartbeat", apiHandler.GetHeartbeat)
		api.GET("/logs", apiHandler.GetLogs)
		api.POST("/logs/clear", apiHandler.ClearLogs)
		api.GET("/replication", apiHandler.GetReplicationLogs)
		api.POST("/kv/set", apiHandler.SetKey)
		api.GET("/kv/:key", apiHandler.GetKey)
		api.DELETE("/kv/:key", apiHandler.DeleteKey)
	}

	return router
}

func Run() {
	router := SetupRouter()
	log.Println("Dice Distributed KV Store Dashboard đang chạy tại http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
