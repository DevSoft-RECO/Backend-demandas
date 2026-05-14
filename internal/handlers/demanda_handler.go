package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/DevSoft-RECO/backend-creditos-go/internal/db"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/models"
	"github.com/gofiber/fiber/v2"
)

// ExportDemandasCSVHandler genera un CSV con todos los campos de la tabla demandas
func ExportDemandasCSVHandler(c *fiber.Ctx) error {
	var demandas []models.Demanda
	if err := db.DB.Preload("Agencia").Preload("Seguimiento").Order("id asc").Find(&demandas).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error exportando demandas", "error": err.Error()})
	}

	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=demandas_export_%s.csv", time.Now().Format("20060102_150405")))

	b := &bytes.Buffer{}
	b.Write([]byte{0xEF, 0xBB, 0xBF}) // BOM para UTF-8 en Excel
	w := csv.NewWriter(b)

	// Header con todos los campos de la tabla demandas
	w.Write([]string{
		"ID",
		"Agencia",
		"No. Credito",
		"CIF",
		"Codigo Cliente",
		"No. Credito T24",
		"Deudor",
		"Fiadores",
		"Salario Embargado A",
		"No. Juicio",
		"Fecha Ingreso Demanda",
		"Abogado (Excel)",
		"Monto Demanda",
		"Situacion",
		"Forma Resolucion",
		"Costas Judiciales",
		"Costas Recuperadas",
		"Observacion 1",
		"Estado Legal",
		"Seguimiento Legacy",
		"Observacion 2",
		"Tiene Seguimiento",
		"Estado Seguimiento",
		"Fecha Creacion",
		"Ultima Actualizacion",
	})

	for _, d := range demandas {
		agencia := ""
		if d.Agencia != nil {
			agencia = d.Agencia.Nombre
		}

		tieneSeguimiento := "No"
		estadoSeguimiento := ""
		if d.Seguimiento != nil {
			tieneSeguimiento = "Si"
			estadoSeguimiento = d.Seguimiento.EstadoLegalDemanda
		}

		w.Write([]string{
			strconv.Itoa(int(d.ID)),
			agencia,
			ptrStr(d.NoCredito),
			ptrStr(d.CIF),
			ptrStr(d.CodigoCliente),
			ptrStr(d.NoCreditoT24),
			ptrStr(d.Deudor),
			ptrStr(d.Fiadores),
			ptrStr(d.SalarioEmbargadoA),
			ptrStr(d.NoJuicio),
			ptrStr(d.FechaIngresoDemanda),
			ptrStr(d.AbogadoNombreExcel),
			ptrFloat(d.MontoDemanda),
			ptrStr(d.Situacion),
			ptrStr(d.FormaResolucion),
			ptrFloat(d.CostasJudiciales),
			ptrStr(d.CostasRecuperadas),
			ptrStr(d.Observacion1),
			ptrStr(d.EstadoLegal),
			ptrStr(d.SeguimientoLegacy),
			ptrStr(d.Observacion2),
			tieneSeguimiento,
			estadoSeguimiento,
			d.CreatedAt.Format("02/01/2006 15:04"),
			d.UpdatedAt.Format("02/01/2006 15:04"),
		})
	}
	w.Flush()
	return c.Send(b.Bytes())
}

// Helpers para campos nullable
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrFloat(f *float64) string {
	if f == nil {
		return "0"
	}
	return fmt.Sprintf("%.2f", *f)
}

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
