package main

import (
	_ "flows/docs"
	"flows/internal/handler"
	"flows/internal/infrastructure/db"
	"flows/internal/service"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

//	@title			Extended Flow Management System API
//	@version		1.0
//	@description	API for managing connectors and executing flows step-by-step.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:8080
//	@BasePath	/api/v1

func main() {
	// 1. Intentar cargar .env desde el directorio actual (donde se ejecuta el comando)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env found in current directory, checking executable directory...")

		// 2. Si falla, intentar cargar desde el directorio donde está el binario
		ex, err := os.Executable()
		if err == nil {
			exPath := filepath.Dir(ex)
			envPath := filepath.Join(exPath, ".env")
			if err := godotenv.Load(envPath); err != nil {
				log.Printf("No .env found at %s either. Using environment variables.\n", envPath)
			} else {
				log.Printf("Loaded .env from executable directory: %s\n", envPath)
			}
		}
	} else {
		log.Println("Loaded .env from current directory")
	}

	database, err := db.NewDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize Services
	connHandler := handler.NewConnectorHandler(database)

	stepValidator := service.NewStepValidator()
	subscriptionClient := service.NewSubscriptionClientFromEnv()
	connectorService := service.NewConnectorService(database)
	flowManager := service.NewFlowManager(database, stepValidator, subscriptionClient, connectorService)

	// AI Service
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("Warning: OPENAI_API_KEY not set. AI features will fail.")
	}
	aiService := service.NewAIService(apiKey)

	flowHandler := handler.NewFlowHandler(flowManager)
	aiHandler := handler.NewAIHandler(aiService)

	r := gin.Default()

	// CORS for all domains and headers
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	api := r.Group("/api/v1")
	{
		// Connector Routes
		api.POST("/connectors", connHandler.CreateConnector)
		api.GET("/connectors", connHandler.ListConnectors)
		api.POST("/connectors/config", connHandler.CreateConfig)

		// Flow Management
		api.POST("/flows", flowHandler.CreateFlow)
		api.GET("/flows", flowHandler.ListFlows)

		// Flow Execution Routes
		api.POST("/flows/:id/start", flowHandler.StartFlow)
		api.GET("/executions/:uuid/step", flowHandler.GetCurrentStep)
		api.POST("/executions/:uuid/step", flowHandler.SubmitStep)

		// New Execution Management Routes
		api.GET("/executions/:uuid", flowHandler.GetExecution)
		api.POST("/executions/:uuid/retry", flowHandler.RetryExecution)

		// AI Generation
		api.POST("/ai/generate", aiHandler.GenerateFlow)
	}

	// Configurar Swagger para que no use delimitadores personalizados si causan conflictos
	// O simplemente pasar opciones personalizadas si es necesario.
	// Por defecto, ginSwagger usa la info registrada en docs.go
	// Si docs.go tiene delimitadores, ginSwagger intentará usarlos.
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
