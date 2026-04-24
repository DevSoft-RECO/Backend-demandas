package handlers

import (
	"strings"

	"github.com/DevSoft-RECO/backend-creditos-go/internal/db"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/models"
	"github.com/gofiber/fiber/v2"
)

// GET /api/usuarios/search?name=algo
func SearchUsuariosHandler(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("name"))
	if q == "" {
		return c.JSON([]models.Usuario{})
	}

	var usuarios []models.Usuario
	like := "%" + q + "%"
	if err := db.DB.
		Where("name LIKE ?", like).
		Limit(20).
		Find(&usuarios).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error buscando usuarios", "error": err.Error()})
	}

	return c.JSON(usuarios)
}

