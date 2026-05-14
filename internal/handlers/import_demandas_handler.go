package handlers

import (
	"bufio"
	"fmt"
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

// Helper para limpiar enteros (extrae solo dígitos)
func cleanIntField(val string) *int {
	val = strings.TrimSpace(val)
	if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
		val = val[1 : len(val)-1]
	}

	if val == "" || strings.ToUpper(val) == "N/A" || strings.ToUpper(val) == "NULL" {
		return nil
	}

	re := regexp.MustCompile(`[^0-9]`)
	cleanVal := re.ReplaceAllString(val, "")

	if cleanVal == "" {
		return nil
	}

	parsed, err := strconv.Atoi(cleanVal)
	if err != nil {
		return nil
	}
	return &parsed
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

	scanner := bufio.NewScanner(file)
	var demandas []models.Demanda
	var lineBuffer string

	// Pre-cargar IDs válidos de agencias para evitar violaciones de llaves foráneas
	var agencias []models.Agencia
	db.DB.Select("id").Find(&agencias)
	validAgencias := make(map[int]bool)
	for _, a := range agencias {
		validAgencias[int(a.ID)] = true
	}

	// Búfer para ir acumulando las líneas rotas
	for scanner.Scan() {
		line := scanner.Text()

		if lineBuffer == "" {
			lineBuffer = line
		} else {
			// Si estamos concatenando una línea rota, agregamos un espacio para no perder legibilidad
			lineBuffer += " " + line
		}

		// Contamos los puntos y comas. Sabiendo que hay 21 columnas, deben haber al menos 20 separadores.
		semicolonCount := strings.Count(lineBuffer, ";")

		if semicolonCount >= 20 {
			// El registro está completo
			fields := strings.Split(lineBuffer, ";")

			// Si el CSV tiene un encabezado, lo ignoramos verificando que la columna 1 no sea la palabra ID o similar
			if strings.ToLower(strings.TrimSpace(fields[0])) == "id" {
				lineBuffer = ""
				continue
			}

			// Asegurarnos de tener al menos 21 campos (pad en caso de que un registro extraño tenga menos)
			for len(fields) < 21 {
				fields = append(fields, "")
			}

			idAgencia := cleanIntField(fields[1])
			if idAgencia != nil && !validAgencias[*idAgencia] {
				// Si el ID de agencia no existe en la BD, lo ponemos nulo para evitar error de llave foránea
				idAgencia = nil
			}

			d := models.Demanda{
				// ID es la columna 0, la omitimos.
				IDAgencia:           idAgencia,
				NoCredito:           cleanStringField(fields[2]),
				CIF:                 cleanStringField(fields[3]),
				CodigoCliente:       cleanStringField(fields[4]),
				NoCreditoT24:        cleanStringField(fields[5]),
				Deudor:              cleanStringField(fields[6]),
				Fiadores:            cleanStringField(fields[7]),
				SalarioEmbargadoA:   cleanStringField(fields[8]),
				NoJuicio:            cleanStringField(fields[9]),
				FechaIngresoDemanda: cleanStringField(fields[10]),
				AbogadoNombreExcel:  cleanStringField(fields[11]),
				MontoDemanda:        cleanNumericField(fields[12]),
				Situacion:           cleanStringField(fields[13]),
				FormaResolucion:     cleanStringField(fields[14]),
				CostasJudiciales:    cleanNumericField(fields[15]),
				CostasRecuperadas:   cleanStringField(fields[16]),
				Observacion1:        cleanStringField(fields[17]),
				EstadoLegal:         cleanStringField(fields[18]),
				SeguimientoLegacy:   cleanStringField(fields[19]),
				Observacion2:        cleanStringField(fields[20]),
			}

			demandas = append(demandas, d)
			lineBuffer = "" // Limpiamos el buffer para la siguiente fila
		}
	}

	if err := scanner.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error leyendo el contenido del archivo"})
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
