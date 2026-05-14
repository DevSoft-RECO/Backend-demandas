package handlers

import (
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
			// Pagos por Etapas
			if seg.IsPagado1 {
				totalPagado += seg.PagoPactado1
			} else {
				totalPendiente += seg.PagoPactado1
			}
			if seg.IsPagado2 {
				totalPagado += seg.PagoPactado2
			} else {
				totalPendiente += seg.PagoPactado2
			}
			if seg.IsPagado3 {
				totalPagado += seg.PagoPactado3
			} else {
				totalPendiente += seg.PagoPactado3
			}
			if seg.IsPagado4 {
				totalPagado += seg.PagoPactado4
			} else {
				totalPendiente += seg.PagoPactado4
			}

			// Pagos Especiales
			if seg.IsPagadoUnico {
				totalPagado += seg.PagoUnico
			} else if seg.PagoUnico > 0 {
				totalPendiente += seg.PagoUnico
			}
			if seg.IsPagadoDesestimacion {
				totalPagado += seg.MontoDesestimacion
			} else if seg.MontoDesestimacion > 0 {
				totalPendiente += seg.MontoDesestimacion
			}
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

	// Comisión Total = Suma de todas las pactadas + Casos especiales
	totalComision := s.PagoPactado1 + s.PagoPactado2 + s.PagoPactado3 + s.PagoPactado4 + s.PagoUnico + s.MontoDesestimacion

	pagado := 0.0
	if s.IsPagado1 {
		pagado += s.PagoPactado1
	}
	if s.IsPagado2 {
		pagado += s.PagoPactado2
	}
	if s.IsPagado3 {
		pagado += s.PagoPactado3
	}
	if s.IsPagado4 {
		pagado += s.PagoPactado4
	}
	if s.IsPagadoUnico {
		pagado += s.PagoUnico
	}
	if s.IsPagadoDesestimacion {
		pagado += s.MontoDesestimacion
	}

	return c.JSON(fiber.Map{
		"seguimiento":    s,
		"total_comision": totalComision,
		"total_pagado":   pagado,
		"saldo_restante": totalComision - pagado,
	})
}

// RegistrarDesembolsoHandler registra el pago de una etapa o caso especial
func RegistrarDesembolsoHandler(c *fiber.Ctx) error {
	var req struct {
		IDSeguimiento uint        `json:"id_seguimiento"`
		Etapa         interface{} `json:"etapa"`
		Cancelar      bool        `json:"cancelar"` // Nuevo campo para anular montos
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Cuerpo de solicitud inválido"})
	}

	var s models.Seguimiento
	if err := db.DB.First(&s, req.IDSeguimiento).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Seguimiento no encontrado"})
	}

	now := time.Now()

	// Evaluar el tipo de etapa
	switch v := req.Etapa.(type) {
	case float64: // JSON numbers are float64 in Go interface{}
		etapaInt := int(v)
		// La regla del 25% ya se validó al asignar el abogado.
		// No es necesario volver a validarla aquí, especialmente porque en casos de
		// desistimiento los totales cambian y dispararían este error erróneamente.
		switch etapaInt {
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
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Etapa numérica inválida"})
		}

	case string:
		switch v {
		case "unico":
			s.IsPagadoUnico = true
			s.FechaPagoUnico = &now
		case "desestimacion":
			if req.Cancelar {
				// Opción de "Anular Cobro"
				s.MontoDesestimacion = 0
				s.IsPagadoDesestimacion = false
				s.FechaPagoDesestimacion = nil
			} else {
				// Registro normal
				s.IsPagadoDesestimacion = true
				s.FechaPagoDesestimacion = &now

				// Cleanup: Si se paga desestimación, aseguramos que etapas futuras/actuales sean 0
				// para que no quede saldo pendiente en los reportes
				if s.EstadoSeguimiento <= 4 {
					if s.EstadoSeguimiento <= 4 && !s.IsPagado4 {
						s.PagoPactado4 = 0
					}
					if s.EstadoSeguimiento <= 3 && !s.IsPagado3 {
						s.PagoPactado3 = 0
					}
					if s.EstadoSeguimiento <= 2 && !s.IsPagado2 {
						s.PagoPactado2 = 0
					}
					if s.EstadoSeguimiento <= 1 && !s.IsPagado1 {
						s.PagoPactado1 = 0
					}
				}
			}
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Tipo de pago especial inválido"})
		}

	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Tipo de dato para etapa no soportado"})
	}

	if err := db.DB.Save(&s).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error al registrar el pago"})
	}

	return c.JSON(fiber.Map{"status": "ok", "message": "Operación realizada exitosamente"})
}
