package handlers

import (
	"time"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/db"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
)

type CreateTrackingRequest struct {
	IDDemanda     uint    `json:"id_demanda"`
	IDAbogado     uint    `json:"id_abogado"`
	IDPreset      uint    `json:"id_preset"`
	PagoUnico     float64 `json:"pago_unico"`
	MontoDesestimacion float64 `json:"monto_desestimacion"`
	PagoPactado1  float64 `json:"pago_pactado_1"`
	PagoPactado2  float64 `json:"pago_pactado_2"`
	PagoPactado3  float64 `json:"pago_pactado_3"`
	PagoPactado4  float64 `json:"pago_pactado_4"`
}

func CreateInitialTrackingHandler(c *fiber.Ctx) error {
	var req CreateTrackingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Body inválido", "error": err.Error()})
	}

	// Iniciar Transacción
	tx := db.DB.Begin()

	// 1. Obtener Demanda
	var demanda models.Demanda
	if err := tx.First(&demanda, req.IDDemanda).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Demanda no encontrada"})
	}

	// 2. Obtener Preset
	var preset models.Preset
	if err := tx.First(&preset, req.IDPreset).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Preset no encontrado"})
	}

	// 3. Obtener Abogado (Bufete)
	var abogado models.Bufete
	if err := tx.First(&abogado, req.IDAbogado).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Abogado no encontrado"})
	}

	// Cálculos de Pagos Sugeridos
	montoDemanda := 0.0
	if demanda.MontoDemanda != nil {
		montoDemanda = *demanda.MontoDemanda
	}

	porcentajeComision := nz(preset.PorcentajeComision)
	totalComision := montoDemanda * (porcentajeComision / 100.0)

	calcPago := func(porcentaje *float64) float64 {
		if porcentaje == nil { return 0.0 }
		return totalComision * (*porcentaje / 100.0)
	}

	// Verificar si ya existe un seguimiento para esta demanda
	var seguimiento models.Seguimiento
	err := tx.Where("id_demanda = ?", req.IDDemanda).First(&seguimiento).Error
	exists := err == nil

	// Mapear campos comunes
	seguimiento.IDDemanda = req.IDDemanda
	seguimiento.IDAbogado = req.IDAbogado
	seguimiento.PorcentajeDemanda = porcentajeComision
	seguimiento.PagoUnico = req.PagoUnico
	seguimiento.MontoDesestimacion = req.MontoDesestimacion

	// Etapa 1
	seguimiento.PagoSugerido1 = calcPago(preset.PorcentajeEtapa1)
	seguimiento.PagoPactado1 = req.PagoPactado1
	if !exists { seguimiento.Etapa1JSON = datatypes.JSON([]byte("[]")) }

	// Etapa 2
	seguimiento.PagoSugerido2 = calcPago(preset.PorcentajeEtapa2)
	seguimiento.PagoPactado2 = req.PagoPactado2
	if !exists { seguimiento.Etapa2JSON = datatypes.JSON([]byte("[]")) }

	// Etapa 3
	seguimiento.PagoSugerido3 = calcPago(preset.PorcentajeEtapa3)
	seguimiento.PagoPactado3 = req.PagoPactado3
	if !exists { seguimiento.Etapa3JSON = datatypes.JSON([]byte("[]")) }

	// Etapa 4
	seguimiento.PagoSugerido4 = calcPago(preset.PorcentajeEtapa4)
	seguimiento.PagoPactado4 = req.PagoPactado4
	if !exists { seguimiento.Etapa4JSON = datatypes.JSON([]byte("[]")) }

	if !exists {
		seguimiento.EstadoSeguimiento = 1
		seguimiento.EstadoLegalDemanda = "Vigente"
		seguimiento.FechaEstadoSeguimiento = time.Now()
		seguimiento.FechaEstadoLegal = time.Now()
		if err := tx.Create(&seguimiento).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error creando seguimiento", "error": err.Error()})
		}
	} else {
		if err := tx.Save(&seguimiento).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error actualizando seguimiento", "error": err.Error()})
		}
	}

	tx.Commit()
	return c.Status(fiber.StatusOK).JSON(seguimiento)
}

func ListSeguimientosHandler(c *fiber.Ctx) error {
	var seguimientos []models.Seguimiento
	if err := db.DB.Preload("Demanda").Preload("Abogado").Order("id desc").Find(&seguimientos).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error listando seguimientos", "error": err.Error()})
	}
	return c.JSON(seguimientos)
}

// Helper para nil float64
func nz(f *float64) float64 {
	if f == nil { return 0.0 }
	return *f
}
