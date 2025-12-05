package main

import (
	"MIA_VDIC2S2025/backend/pkg/handlers"
	"MIA_VDIC2S2025/backend/pkg/services"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Cargar discos existentes al iniciar
	log.Println("🔍 Cargando discos existentes...")
	services.LoadExistingDisks()
	diskCount := services.GetDiskCount()
	if diskCount > 0 {
		log.Printf("📀 Se encontraron %d disco(s) existente(s)\n", diskCount)
	}

	// Configurar Gin
	router := gin.Default()

	// Configuración de CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Ruta de prueba
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
			"status":  "Backend funcionando correctamente ✅",
			"time":    time.Now().Format("2006-01-02 15:04:05"),
			"disks":   services.GetDiskCount(),
		})
	})

	// Ruta principal para ejecutar comandos
	router.POST("/api/execute", handleExecuteCommands)

	// Ruta para listar particiones montadas
	router.GET("/api/mounted", handleGetMounted)

	// Ruta para listar discos
	router.GET("/api/disks", handleGetDisks)

	log.Println("╔════════════════════════════════════════════╗")
	log.Println("║   🚀 Servidor Backend Iniciado            ║")
	log.Println("║   📡 http://localhost:8080                ║")
	log.Println("║   📁 Sistema de Archivos EXT2             ║")
	log.Println("╚════════════════════════════════════════════╝")

	if err := router.Run(":8080"); err != nil {
		log.Fatal("Error al iniciar el servidor:", err)
	}
}

// Handler para ejecutar comandos
func handleExecuteCommands(c *gin.Context) {
	var request struct {
		Commands string `json:"commands" binding:"required"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Formato de solicitud inválido",
			"success": false,
		})
		return
	}

	// Ejecutar comandos usando el handler
	output := handlers.ExecuteCommands(request.Commands)

	c.JSON(http.StatusOK, gin.H{
		"output":  output,
		"success": true,
	})
}

// Handler para obtener particiones montadas
func handleGetMounted(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"mounted": []string{},
		"message": "No hay particiones montadas actualmente",
	})
}

// Handler para obtener lista de discos
func handleGetDisks(c *gin.Context) {
	disks := services.GetCreatedDisks()
	c.JSON(http.StatusOK, gin.H{
		"disks": disks,
		"count": len(disks),
	})
}
