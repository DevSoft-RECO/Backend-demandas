package handlers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
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

type BitacoraEntry struct {
	Fecha      string `json:"fecha"`
	Comentario string `json:"comentario"`
	Documento  string `json:"documento,omitempty"`
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

	// Verificar si ya existe un seguimiento para esta demanda
	var seguimiento models.Seguimiento
	err := tx.Where("id_demanda = ?", req.IDDemanda).First(&seguimiento).Error
	exists := err == nil

	// 2. Obtener Preset (Solo obligatorio si es nuevo o si se envía un IDPreset > 0)
	var preset models.Preset
	if req.IDPreset > 0 {
		if err := tx.First(&preset, req.IDPreset).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Preset no encontrado"})
		}
	} else if !exists {
		// Si es nuevo seguimiento, el preset es OBLIGATORIO
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Se requiere una plantilla de pagos (Preset) para nuevas asignaciones"})
	}

	// 3. Obtener Abogado (Bufete)
	var abogado models.Bufete
	if err := tx.First(&abogado, req.IDAbogado).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Abogado no encontrado"})
	}

	// Cálculos de Pagos Sugeridos (Solo si hay un preset nuevo)
	if req.IDPreset > 0 {
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

		seguimiento.PorcentajeDemanda = porcentajeComision
		seguimiento.PagoSugerido1 = calcPago(preset.PorcentajeEtapa1)
		seguimiento.PagoSugerido2 = calcPago(preset.PorcentajeEtapa2)
		seguimiento.PagoSugerido3 = calcPago(preset.PorcentajeEtapa3)
		seguimiento.PagoSugerido4 = calcPago(preset.PorcentajeEtapa4)
	}

	// Mapear campos comunes
	seguimiento.IDDemanda = req.IDDemanda
	seguimiento.IDAbogado = req.IDAbogado
	seguimiento.PagoUnico = req.PagoUnico
	seguimiento.MontoDesestimacion = req.MontoDesestimacion

	// Actualizar Pactados
	seguimiento.PagoPactado1 = req.PagoPactado1
	seguimiento.PagoPactado2 = req.PagoPactado2
	seguimiento.PagoPactado3 = req.PagoPactado3
	seguimiento.PagoPactado4 = req.PagoPactado4

	// Inicializar JSONs si es nuevo
	if !exists {
		seguimiento.Etapa1JSON = datatypes.JSON([]byte("[]"))
		seguimiento.Etapa2JSON = datatypes.JSON([]byte("[]"))
		seguimiento.Etapa3JSON = datatypes.JSON([]byte("[]"))
		seguimiento.Etapa4JSON = datatypes.JSON([]byte("[]"))
		
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
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("pageSize", 10)
	abogadoID := c.QueryInt("abogadoId", 0)
	status := c.Query("status", "")

	query := db.DB.Model(&models.Seguimiento{}).Preload("Demanda").Preload("Abogado")

	if abogadoID > 0 {
		query = query.Where("id_abogado = ?", abogadoID)
	}
	if status != "" && status != "Todos" {
		query = query.Where("estado_legal_demanda = ?", status)
	}

	var total int64
	query.Count(&total)

	var seguimientos []models.Seguimiento
	offset := (page - 1) * pageSize
	if err := query.Order("id desc").Offset(offset).Limit(pageSize).Find(&seguimientos).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error listando seguimientos", "error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":       seguimientos,
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func ExportSeguimientosCSVHandler(c *fiber.Ctx) error {
	abogadoID := c.QueryInt("abogadoId", 0)
	status := c.Query("status", "")

	query := db.DB.Model(&models.Seguimiento{}).Preload("Demanda").Preload("Abogado")

	if abogadoID > 0 {
		query = query.Where("id_abogado = ?", abogadoID)
	}
	if status != "" && status != "Todos" {
		query = query.Where("estado_legal_demanda = ?", status)
	}

	var seguimientos []models.Seguimiento
	if err := query.Order("id desc").Find(&seguimientos).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", "attachment; filename=reporte_seguimiento_legal.csv")

	b := &bytes.Buffer{}
	b.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(b)

	// Header enfocado en Seguimiento Procesal
	w.Write([]string{
		"No. Credito", 
		"No. Juicio", 
		"Deudor", 
		"Bufete Asignado", 
		"Etapa Actual", 
		"Estado Proceso", 
		"Ultima Actividad",
		"Bitacora E1 (Presentacion)", 
		"Bitacora E2 (Admision)", 
		"Bitacora E3 (Notificacion)", 
		"Bitacora E4 (Ejecucion)",
	})

	for _, s := range seguimientos {
		noCredito := ""
		if s.Demanda != nil {
			if s.Demanda.NoCredito != nil {
				noCredito = *s.Demanda.NoCredito
			} else if s.Demanda.NoCreditoT24 != nil {
				noCredito = *s.Demanda.NoCreditoT24
			}
		}
		
		noJuicio := "S/N Juicio"
		if s.Demanda != nil && s.Demanda.NoJuicio != nil {
			noJuicio = *s.Demanda.NoJuicio
		}

		deudor := "N/A"
		if s.Demanda != nil && s.Demanda.Deudor != nil {
			deudor = *s.Demanda.Deudor
		}

		bufete := "No Asignado"
		if s.Abogado != nil && s.Abogado.Nombre != nil {
			bufete = *s.Abogado.Nombre
		}

		etapaLabel := getEtapaLabel(s.EstadoSeguimiento)
		if s.EstadoLegalDemanda == "Desistido" {
			etapaLabel = "Finalizacion Anticipada"
		}

		w.Write([]string{
			noCredito,
			noJuicio,
			deudor,
			bufete,
			etapaLabel,
			s.EstadoLegalDemanda,
			s.FechaEstadoSeguimiento.Format("02/01/2006"),
			formatBitacora(s.Etapa1JSON),
			formatBitacora(s.Etapa2JSON),
			formatBitacora(s.Etapa3JSON),
			formatBitacora(s.Etapa4JSON),
		})
	}
	w.Flush()
	return c.Send(b.Bytes())
}

func getEtapaLabel(id int) string {
	stages := map[int]string{
		1: "Presentacion",
		2: "Admision",
		3: "Notificacion",
		4: "Ejecucion",
		5: "Finalizado",
	}
	if label, ok := stages[id]; ok {
		return label
	}
	return "Desconocido"
}

func formatBitacora(data datatypes.JSON) string {
	var entries []BitacoraEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return ""
	}
	
	var lines []string
	for _, e := range entries {
		// Limpiar comas para no romper el CSV si no se usan comillas (aunque csv.Writer las pone)
		lines = append(lines, fmt.Sprintf("[%s] %s", e.Fecha, e.Comentario))
	}
	return strings.Join(lines, " | ")
}

// Helper para nil float64
func nz(f *float64) float64 {
	if f == nil { return 0.0 }
	return *f
}
