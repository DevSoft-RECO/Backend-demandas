package handlers

import (
	"fmt"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/auth"
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

func SyncAgenciasHandler(c *fiber.Ctx) error {
	rawAuth := c.Get("Authorization")
	if rawAuth == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"detail": "No se proporcionó token"})
	}

	motherAgencias, err := auth.FetchAgenciasFromMother(rawAuth)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"detail": "Error consultando agencias de la App Madre", "error": err.Error()})
	}

	count := 0
	for _, ma := range motherAgencias {
		ag := models.Agencia{
			ID:        ma.ID,
			Nombre:    ma.Nombre,
			Codigo:    ma.Codigo,
			CodigoT24: ma.CodigoT24,
			Direccion: ma.Direccion,
		}
		if err := db.DB.Save(&ag).Error; err == nil {
			count++
		}
	}

	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": fmt.Sprintf("Se sincronizaron %d agencias", count),
	})
}
