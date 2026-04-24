package routes

import (
	"github.com/DevSoft-RECO/backend-creditos-go/internal/config"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/handlers"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/middleware"

	"github.com/gofiber/fiber/v2"

)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api")

	// Auth
	api.Get("/me", handlers.MeHandler)

	// Agencias
	api.Get("/agencias", middleware.AuthRequired, handlers.ListAgenciasHandler)

	// Bufetes (Módulo 1)
	api.Get("/bufetes", middleware.AuthRequired, handlers.ListBufetesHandler)
	api.Post("/bufetes", middleware.AuthRequired, handlers.CreateBufeteHandler)
	api.Get("/usuarios/search", middleware.AuthRequired, handlers.SearchUsuariosHandler)

	// Presets (Módulo 1)
	api.Get("/presets", middleware.AuthRequired, handlers.ListPresetsHandler)
	api.Get("/presets/:id", middleware.AuthRequired, handlers.GetPresetHandler)
	api.Post("/presets", middleware.AuthRequired, handlers.CreatePresetHandler)
	api.Put("/presets/:id", middleware.AuthRequired, handlers.UpdatePresetHandler)
	api.Delete("/presets/:id", middleware.AuthRequired, handlers.DeletePresetHandler)

	// Demandas
	api.Get("/demandas", middleware.AuthRequired, handlers.ListDemandasHandler)
	api.Get("/demandas/:id", middleware.AuthRequired, handlers.GetDemandaHandler)
	api.Post("/demandas", middleware.AuthRequired, handlers.CreateDemandaHandler)
	api.Put("/demandas/:id", middleware.AuthRequired, handlers.UpdateDemandaHandler)
	api.Delete("/demandas/:id", middleware.AuthRequired, handlers.DeleteDemandaHandler)


	// Health check
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"app":    "Yaman Kutx Ecosistema API (Go)",
		})
	})

	// 🕷️ 8. Arquitectura Anti-JSON (Capa de redirección)
	app.Get("/login", func(c *fiber.Ctx) error {
		return c.Redirect(config.Envs.FrontendURL + "/login?session_expired=true")
	})
}

