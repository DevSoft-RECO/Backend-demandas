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
	api.Post("/agencias/sync", middleware.AuthRequired, handlers.SyncAgenciasHandler)

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
	api.Post("/demandas/import", middleware.AuthRequired, handlers.ImportDemandasHandler)
	api.Get("/demandas/export", middleware.AuthRequired, handlers.ExportDemandasCSVHandler)
	api.Get("/demandas", middleware.AuthRequired, handlers.ListDemandasHandler)
	api.Get("/demandas/:id", middleware.AuthRequired, handlers.GetDemandaHandler)
	api.Post("/demandas", middleware.AuthRequired, handlers.CreateDemandaHandler)
	api.Put("/demandas/:id", middleware.AuthRequired, handlers.UpdateDemandaHandler)
	api.Delete("/demandas/:id", middleware.AuthRequired, handlers.DeleteDemandaHandler)

	// Seguimientos
	api.Get("/seguimientos", middleware.AuthRequired, handlers.ListSeguimientosHandler)
	api.Get("/seguimientos/export", middleware.AuthRequired, handlers.ExportSeguimientosCSVHandler)
	api.Post("/seguimientos", middleware.AuthRequired, handlers.CreateInitialTrackingHandler)

	// Seguimiento Abogados (Módulo 2)
	api.Get("/seguimientos/abogado", middleware.AuthRequired, handlers.GetSeguimientosByAbogadoHandler)
	api.Post("/seguimientos/:id/comentario", middleware.AuthRequired, handlers.AddComentarioHandler)
	api.Patch("/seguimientos/:id/avanzar", middleware.AuthRequired, handlers.AvanzarEtapaHandler)
	api.Patch("/seguimientos/:id/desistir", middleware.AuthRequired, handlers.DesistirSeguimientoHandler)
	api.Get("/evidencias/signed", middleware.AuthRequired, handlers.GetSignedURLHandler)

	// Pagos (Módulo 3)
	api.Get("/pagos/resumen-abogados", middleware.AuthRequired, handlers.GetPagosResumenAbogadosHandler)
	api.Get("/pagos/seguimiento/:id", middleware.AuthRequired, handlers.GetPagoDetalleSeguimientoHandler)
	api.Patch("/pagos/registrar-desembolso", middleware.AuthRequired, handlers.RegistrarDesembolsoHandler)

	// Dashboard
	api.Get("/dashboard/stats", middleware.AuthRequired, handlers.GetDashboardStats)


	// Health check
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"app":    "Yaman Kutx Ecosistema API (Go)",
		})
	})

	// === BACKUP SYSTEM ===
	// Rutas internas de respaldo llamadas por la APP_MADRE (Firmadas con HMAC)
	api.Post("/internal/backup", handlers.GenerateBackupHandler)
	api.Delete("/internal/backup", handlers.DeleteBackupHandler)
	api.Get("/internal/download-backup", handlers.DownloadBackupHandler)

	// 🕷️ 8. Arquitectura Anti-JSON (Capa de redirección)
	app.Get("/login", func(c *fiber.Ctx) error {
		return c.Redirect(config.Envs.FrontendURL + "/login?session_expired=true")
	})
}
