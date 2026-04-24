package handlers

import (
	"github.com/DevSoft-RECO/backend-creditos-go/internal/db"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/models"
	"github.com/gofiber/fiber/v2"
)

func ListPresetsHandler(c *fiber.Ctx) error {
	var presets []models.Preset
	if err := db.DB.Order("id desc").Find(&presets).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error listando presets", "error": err.Error()})
	}
	return c.JSON(presets)
}

func GetPresetHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var preset models.Preset
	if err := db.DB.First(&preset, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Preset no encontrado", "error": err.Error()})
	}
	return c.JSON(preset)
}

func CreatePresetHandler(c *fiber.Ctx) error {
	var preset models.Preset
	if err := c.BodyParser(&preset); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Body inválido", "error": err.Error()})
	}

	if preset.Nombre == nil || *preset.Nombre == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "El nombre es requerido"})
	}

	if err := db.DB.Create(&preset).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error creando preset", "error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(preset)
}

func UpdatePresetHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var existingPreset models.Preset
	if err := db.DB.First(&existingPreset, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Preset no encontrado", "error": err.Error()})
	}

	if err := c.BodyParser(&existingPreset); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Body inválido", "error": err.Error()})
	}

	if err := db.DB.Save(&existingPreset).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error actualizando preset", "error": err.Error()})
	}

	return c.JSON(existingPreset)
}

func DeletePresetHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := db.DB.Delete(&models.Preset{}, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error eliminando preset", "error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
