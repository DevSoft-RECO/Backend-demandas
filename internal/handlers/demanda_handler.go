package handlers

import (
	"strconv"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/db"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/models"
	"github.com/gofiber/fiber/v2"
)

func ListDemandasHandler(c *fiber.Ctx) error {
	var demandas []models.Demanda
	var total int64

	query := db.DB.Model(&models.Demanda{}).Preload("Agencia").Preload("Seguimiento").Order("id desc")

	// Filtros
	search := c.Query("search")
	if search != "" {
		s := "%" + search + "%"
		query = query.Where("no_credito LIKE ? OR cif LIKE ? OR deudor LIKE ? OR no_juicio LIKE ?", s, s, s, s)
	}

	estado := c.Query("estado")
	if estado != "" {
		query = query.Where("LOWER(estado_legal) = LOWER(?)", estado)
	}

	// Contar total antes de paginar
	query.Count(&total)

	// Paginación
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if page < 1 { page = 1 }
	if limit < 1 { limit = 10 }
	offset := (page - 1) * limit

	if err := query.Limit(limit).Offset(offset).Find(&demandas).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error listando demandas", "error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"total": total,
		"page":  page,
		"limit": limit,
		"data":  demandas,
	})
}

func GetDemandaHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var demanda models.Demanda
	if err := db.DB.First(&demanda, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Demanda no encontrada", "error": err.Error()})
	}
	return c.JSON(demanda)
}

func CreateDemandaHandler(c *fiber.Ctx) error {
	var demanda models.Demanda
	if err := c.BodyParser(&demanda); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Body inválido", "error": err.Error()})
	}

	if err := db.DB.Create(&demanda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error creando demanda", "error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(demanda)
}

func UpdateDemandaHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var existingDemanda models.Demanda
	if err := db.DB.First(&existingDemanda, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Demanda no encontrada", "error": err.Error()})
	}

	if err := c.BodyParser(&existingDemanda); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Body inválido", "error": err.Error()})
	}

	if err := db.DB.Save(&existingDemanda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error actualizando demanda", "error": err.Error()})
	}

	return c.JSON(existingDemanda)
}

func DeleteDemandaHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := db.DB.Delete(&models.Demanda{}, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error eliminando demanda", "error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
