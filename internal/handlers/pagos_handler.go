package handlers

import (
	"fmt"
	"time"

	"github.com/DevSoft-RECO/backend-creditos-go/internal/db"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/models"
	"github.com/gofiber/fiber/v2"
)

type ResumenAbogado struct {
	ID             uint    `json:"id"`
	Nombre         string  `json:"nombre"`
	TotalCasos     int64   `json:"total_casos"`
	TotalPagado    float64 `json:"total_pagado"`
	TotalPendiente float64 `json:"total_pendiente"`
}

// GetPagosResumenAbogadosHandler retorna estadísticas financieras por abogado
func GetPagosResumenAbogadosHandler(c *fiber.Ctx) error {
	var bufetes []models.Bufete
	if err := db.DB.Find(&bufetes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error al listar abogados"})
	}

	var resumen []ResumenAbogado
	for _, b := range bufetes {
		var s []models.Seguimiento
		db.DB.Where("id_abogado = ?", b.ID).Find(&s)

		var totalPagado, totalPendiente float64
		for _, seg := range s {
			// Sumar pagados
			if seg.IsPagado1 { totalPagado += seg.PagoPactado1 }
			if seg.IsPagado2 { totalPagado += seg.PagoPactado2 }
			if seg.IsPagado3 { totalPagado += seg.PagoPactado3 }
			if seg.IsPagado4 { totalPagado += seg.PagoPactado4 }

			// Sumar pendientes
			if !seg.IsPagado1 { totalPendiente += seg.PagoPactado1 }
			if !seg.IsPagado2 { totalPendiente += seg.PagoPactado2 }
			if !seg.IsPagado3 { totalPendiente += seg.PagoPactado3 }
			if !seg.IsPagado4 { totalPendiente += seg.PagoPactado4 }
		}

		nombreAbogado := "Sin Nombre"
		if b.Nombre != nil {
			nombreAbogado = *b.Nombre
		}

		resumen = append(resumen, ResumenAbogado{
			ID:             uint(b.ID),
			Nombre:         nombreAbogado,
			TotalCasos:     int64(len(s)),
			TotalPagado:    totalPagado,
			TotalPendiente: totalPendiente,
		})
	}

	return c.JSON(resumen)
}

// GetPagoDetalleSeguimientoHandler retorna detalle financiero de un caso
func GetPagoDetalleSeguimientoHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var s models.Seguimiento
	if err := db.DB.Preload("Demanda").First(&s, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Seguimiento no encontrado"})
	}

	totalComision := s.PagoPactado1 + s.PagoPactado2 + s.PagoPactado3 + s.PagoPactado4
	pagado := 0.0
	if s.IsPagado1 { pagado += s.PagoPactado1 }
	if s.IsPagado2 { pagado += s.PagoPactado2 }
	if s.IsPagado3 { pagado += s.PagoPactado3 }
	if s.IsPagado4 { pagado += s.PagoPactado4 }

	return c.JSON(fiber.Map{
		"seguimiento":    s,
		"total_comision": totalComision,
		"total_pagado":   pagado,
		"saldo_restante": totalComision - pagado,
	})
}

// RegistrarDesembolsoHandler registra el pago de una etapa
func RegistrarDesembolsoHandler(c *fiber.Ctx) error {
	var req struct {
		IDSeguimiento uint `json:"id_seguimiento"`
		Etapa         int  `json:"etapa"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Cuerpo de solicitud inválido"})
	}

	var s models.Seguimiento
	if err := db.DB.First(&s, req.IDSeguimiento).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Seguimiento no encontrado"})
	}

	// Regla de Etapa 1 (25% Máximo) si no es pago único
	if req.Etapa == 1 && s.PagoUnico == 0 {
		totalComision := s.PagoPactado1 + s.PagoPactado2 + s.PagoPactado3 + s.PagoPactado4
		if s.PagoPactado1 > (totalComision * 0.2501) { // 0.2501 por temas de redondeo
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"detail": fmt.Sprintf("El pago de Etapa 1 (Q%.2f) excede el 25%% de la comisión total (Q%.2f)", s.PagoPactado1, totalComision),
			})
		}
	}

	now := time.Now()
	switch req.Etapa {
	case 1:
		s.IsPagado1 = true
		s.FechaPago1 = &now
	case 2:
		s.IsPagado2 = true
		s.FechaPago2 = &now
	case 3:
		s.IsPagado3 = true
		s.FechaPago3 = &now
	case 4:
		s.IsPagado4 = true
		s.FechaPago4 = &now
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Etapa inválida"})
	}

	if err := db.DB.Save(&s).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error al registrar el pago"})
	}

	return c.JSON(fiber.Map{"status": "ok", "message": "Pago registrado exitosamente"})
}
