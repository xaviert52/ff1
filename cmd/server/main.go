package main

import (
	_ "flows/docs"
	"flows/internal/handler"
	"flows/internal/infrastructure/db"
	"flows/internal/service"
	"log"
	"os"
	"path/filepath"

	"flows/internal/domain"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
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

	SeedDatabase(database)

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

func SeedDatabase(database *gorm.DB) {
	var count int64
	database.Model(&domain.Flow{}).Count(&count)

	if count == 0 {
		log.Println("Seeding initial database configuration...")

		// 1. Insertar Conectores
		connectors := []domain.Connector{
			{Name: "OTP Generator", Type: "REST", BaseURL: "http://host.docker.internal:8081/api/v1", AuthType: "NONE"},
			{Name: "WhatsApp Notify", Type: "REST", BaseURL: "http://host.docker.internal:8081/api/v1", AuthType: "NONE"},
			{Name: "OTP Extractor", Type: "REST", BaseURL: "http://host.docker.internal:8081/api/v1", AuthType: "NONE"},
			{Name: "OTP Validator", Type: "REST", BaseURL: "http://host.docker.internal:8081/api/v1", AuthType: "NONE"},
		}
		for _, c := range connectors {
			database.Create(&c)
		}

		// 2. Insertar el Flujo 1 con el JSON corregido (El Poka-Yoke validado)
		definition := `{
			"start_step": "create_otp",
			"steps": {
				"create_otp": {
					"id": "create_otp",
					"type": "ACTION",
					"connector_id": 1,
					"config": {"route": "/otp/generate"},
					"input_mapping": { "account_name": "{{global.phone_number}}", "period_minutes": 5 },
					"next_step": "extract_otp"
				},
				"extract_otp": {
					"id": "extract_otp",
					"type": "ACTION",
					"connector_id": 3,
					"config": {"route": "/otp/code/from_url"},
					"input_mapping": { "url": "{{steps.create_otp.data.0.url}}" },
					"next_step": "send_whatsapp"
				},
				"send_whatsapp": {
					"id": "send_whatsapp",
					"type": "ACTION",
					"connector_id": 2,
					"config": {"route": "/notify"},
					"input_mapping": {
						"type": "whatsapp",
						"recipient": "{{global.phone_number}}",
						"body": "Tu codigo de verificacion es: {{steps.extract_otp.data.0.code}}"
					},
					"next_step": "ask_otp"
				},
				"ask_otp": {
					"id": "ask_otp",
					"type": "FORM",
					"next_step": "validate_otp"
				},
				"validate_otp": {
					"id": "validate_otp",
					"type": "ACTION",
					"connector_id": 4,
					"config": {"route": "/otp/validate"},
					"input_mapping": {
						"code": "{{input.code}}",
						"secret": "{{steps.create_otp.data.0.secret}}",
						"period_minutes": 5
					},
					"next_step": "final_message"
				},
				"final_message": {
					"id": "final_message",
					"type": "ACTION",
					"connector_id": 2,
					"config": {"route": "/notify"},
					"input_mapping": {
						"type": "whatsapp",
						"recipient": "{{global.phone_number}}",
						"body": "✅ ¡Felicidades! Verificación exitosa."
					}
				}
			}
		}`

		// Casteamos el string a json.RawMessage (que es un []byte)
		flow := domain.Flow{
			Name:        "WhatsApp OTP Verification",
			Description: "E2E flow for generating, sending, and validating OTP via WhatsApp",
			Definition:  []byte(definition), // <-- Esta es la corrección clave
		}
		database.Create(&flow)
		log.Println("Database seeding completed.")
	}
}
