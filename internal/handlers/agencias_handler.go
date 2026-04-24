package handlers

import (
	"github.com/DevSoft-RECO/backend-creditos-go/internal/db"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/models"
	"github.com/gofiber/fiber/v2"
)

func ListAgenciasHandler(c *fiber.Ctx) error {
	var agencias []models.Agencia
	if err := db.DB.Order("nombre asc").Find(&agencias).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error listando agencias", "error": err.Error()})
	}
	return c.JSON(agencias)
}
