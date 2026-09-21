package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/DevSoft-RECO/backend-creditos-go/internal/db"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/models"
	"github.com/gofiber/fiber/v2"
)

// Helper para limpiar strings (quita espacios, comillas y convierte "N/A" a nulo)
func cleanStringField(val string) *string {
	val = strings.TrimSpace(val)
	// Quitar comillas dobles externas que Excel a veces deja
	if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
		val = val[1 : len(val)-1]
	}
	val = strings.TrimSpace(val)
	val = strings.ReplaceAll(val, "\x00", "")

	if val == "" || strings.ToUpper(val) == "N/A" || strings.ToUpper(val) == "NULL" {
		return nil
	}
	return &val
}

// Helper para limpiar números (extrae solo dígitos y punto decimal)
func cleanNumericField(val string) *float64 {
	val = strings.TrimSpace(val)
	if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
		val = val[1 : len(val)-1]
	}

	if val == "" || strings.ToUpper(val) == "N/A" || strings.ToUpper(val) == "NULL" {
		return nil
	}

	// Dejar solo números y el punto decimal. Esto limpiará letras, comas, guiones, Q, etc.
	re := regexp.MustCompile(`[^0-9.]`)
	cleanVal := re.ReplaceAllString(val, "")

	if cleanVal == "" {
		return nil
	}

	parsed, err := strconv.ParseFloat(cleanVal, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

// Helper para resolver la agencia por ID o por nombre fuzzy
func resolveAgenciaID(raw string, agencias []models.Agencia) *int {
	clean := strings.TrimSpace(raw)
	if clean == "" || strings.ToUpper(clean) == "N/A" || strings.ToUpper(clean) == "NULL" {
		return nil
	}

	// 1. Si es numérico directo
	if idNum, err := strconv.Atoi(clean); err == nil {
		for _, a := range agencias {
			if a.ID == idNum {
				res := a.ID
				return &res
			}
		}
	}

	lower := strings.ToLower(clean)

	// 2. Caso especial Huehue07 -> Agencia Huehuetenango
	if strings.Contains(lower, "huehue") {
		for _, a := range agencias {
			if strings.Contains(strings.ToLower(a.Nombre), "huehuetenango") {
				res := a.ID
				return &res
			}
		}
	}

	// 3. Coincidencia por subcadena de nombre
	for _, a := range agencias {
		aLower := strings.ToLower(a.Nombre)
		nameWithoutPrefix := strings.TrimPrefix(aLower, "agencia ")
		if strings.Contains(aLower, lower) || strings.Contains(lower, nameWithoutPrefix) {
			res := a.ID
			return &res
		}
	}

	return nil
}

// Helper para normalizar el estado legal clasificando textos largos y preservando detalles
func normalizeEstadoLegal(raw string, obs2 *string) (*string, *string) {
	clean := strings.TrimSpace(raw)
	if clean == "" || strings.ToUpper(clean) == "N/A" || strings.ToUpper(clean) == "NULL" {
		vig := "Vigente"
		return &vig, obs2
	}

	upper := strings.ToUpper(clean)
	var standard string

	if strings.Contains(upper, "DESIST") || strings.Contains(upper, "DISIST") || strings.Contains(upper, "DESIT") {
		standard = "Desistido"
	} else if strings.Contains(upper, "CANCEL") || strings.Contains(upper, "RESUELT") {
		standard = "Cancelado"
	} else if strings.Contains(upper, "SUSPEND") {
		standard = "Suspendido"
	} else if strings.Contains(upper, "VIGENT") {
		standard = "Vigente"
	} else {
		standard = "Vigente"
	}

	// Si el texto contenía detalles o fechas adicionales (ej: "DESISTIDO EN ENERO 2026", "DESISTIDO (PAGO DE COSTAS)"),
	// lo preservamos en Observacion2 para que no se pierda la información original.
	if upper != "DESISTIDO" && upper != "VIGENTE" && upper != "CANCELADO" && upper != "SUSPENDIDO" && upper != "DESISTIMIENTO" {
		tag := fmt.Sprintf("[Estado original: %s]", clean)
		if obs2 == nil || *obs2 == "" {
			obs2 = &tag
		} else if !strings.Contains(*obs2, clean) {
			combined := fmt.Sprintf("%s | %s", *obs2, tag)
			obs2 = &combined
		}
	}

	return &standard, obs2
}

func ImportDemandasHandler(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "No se proporcionó un archivo válido"})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error al abrir el archivo"})
	}
	defer file.Close()

	// Pre-cargar catálogo de agencias
	var agencias []models.Agencia
	db.DB.Find(&agencias)

	// Lector CSV estándar compatible con RFC 4180 (soporta saltos de línea dentro de celdas entre comillas)
	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	var demandas []models.Demanda

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// En caso de alguna anomalía en una fila puntual, continuamos con las demás
			continue
		}

		if len(record) == 0 {
			continue
		}

		// Quitar posible BOM UTF-8 del primer campo
		firstVal := strings.TrimSpace(record[0])
		firstVal = strings.TrimPrefix(firstVal, "\ufeff")

		// Si es la fila de encabezado ("id"), la ignoramos
		if strings.ToLower(firstVal) == "id" {
			continue
		}

		// Rellenar con cadenas vacías si el registro tiene menos de 21 columnas
		for len(record) < 21 {
			record = append(record, "")
		}

		idAgencia := resolveAgenciaID(record[1], agencias)
		obs2 := cleanStringField(record[20])
		estadoLegal, obs2 := normalizeEstadoLegal(record[18], obs2)

		d := models.Demanda{
			IDAgencia:           idAgencia,
			NoCredito:           cleanStringField(record[2]),
			CIF:                 cleanStringField(record[3]),
			CodigoCliente:       cleanStringField(record[4]),
			NoCreditoT24:        cleanStringField(record[5]),
			Deudor:              cleanStringField(record[6]),
			Fiadores:            cleanStringField(record[7]),
			SalarioEmbargadoA:   cleanStringField(record[8]),
			NoJuicio:            cleanStringField(record[9]),
			FechaIngresoDemanda: cleanStringField(record[10]),
			AbogadoNombreExcel:  cleanStringField(record[11]),
			MontoDemanda:        cleanNumericField(record[12]),
			Situacion:           cleanStringField(record[13]),
			FormaResolucion:     cleanStringField(record[14]),
			CostasJudiciales:    cleanNumericField(record[15]),
			CostasRecuperadas:   cleanStringField(record[16]),
			Observacion1:        cleanStringField(record[17]),
			EstadoLegal:         estadoLegal,
			SeguimientoLegacy:   cleanStringField(record[19]),
			Observacion2:        obs2,
		}

		demandas = append(demandas, d)
	}

	if len(demandas) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "No se encontraron registros válidos para importar o el formato no es el esperado (separador ;)"})
	}

	// Insertamos por lotes de 100 para no saturar la memoria/red
	result := db.DB.CreateInBatches(&demandas, 100)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error guardando en la base de datos: " + result.Error.Error()})
	}

	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": fmt.Sprintf("Se importaron %d expedientes exitosamente", len(demandas)),
	})
}
