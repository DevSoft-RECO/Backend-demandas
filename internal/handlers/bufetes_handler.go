package handlers

import (
	"strings"

	"github.com/DevSoft-RECO/backend-creditos-go/internal/db"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/models"
	"github.com/gofiber/fiber/v2"
)

type CreateBufeteRequest struct {
	UsuarioID int `json:"usuario_id"`
}

func ListBufetesHandler(c *fiber.Ctx) error {
	var bufetes []models.Bufete
	if err := db.DB.Find(&bufetes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error listando bufetes", "error": err.Error()})
	}
	return c.JSON(bufetes)
}

func CreateBufeteHandler(c *fiber.Ctx) error {
	var body CreateBufeteRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Body inválido", "error": err.Error()})
	}

	if body.UsuarioID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "El campo 'usuario_id' es requerido"})
	}

	// Copiar datos básicos desde la tabla usuarios
	var usuario models.Usuario
	if err := db.DB.First(&usuario, body.UsuarioID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Usuario no encontrado", "error": err.Error()})
	}

	if usuario.Name == nil || strings.TrimSpace(*usuario.Name) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "El usuario no tiene 'name' para precargar el bufete"})
	}

	// Verificar si el usuario ya está asignado a un bufete
	var existingBufete models.Bufete
	if err := db.DB.Where("usuario_id = ?", body.UsuarioID).First(&existingBufete).Error; err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Este usuario ya está asignado a un bufete registrado"})
	}

	bufete := models.Bufete{
		UsuarioID: body.UsuarioID,
		Nombre:    usuario.Name,
		Telefono:  usuario.Telefono,
	}

	if err := db.DB.Create(&bufete).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error creando bufete", "error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(bufete)
}

