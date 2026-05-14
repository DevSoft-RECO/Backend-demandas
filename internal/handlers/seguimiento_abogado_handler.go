package handlers

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/DevSoft-RECO/backend-creditos-go/internal/auth"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/db"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/gcs"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
)

// DemandaSimple DTO con campos básicos para el abogado
type DemandaSimple struct {
	ID           uint    `json:"id"`
	NoCredito    *string `json:"no_credito"`
	NoCreditoT24 *string `json:"no_credito_t24"`
	Deudor       *string `json:"deudor"`
	NoJuicio     *string `json:"no_juicio"`
}

// SeguimientoAbogadoResponse DTO para ocultar campos financieros y optimizar data
type SeguimientoAbogadoResponse struct {
	ID                  uint            `json:"id"`
	IDDemanda           uint            `json:"id_demanda"`
	Demanda             DemandaSimple   `json:"demanda"`
	EstadoSeguimiento   int             `json:"estado_seguimiento"`
	EstadoLegalDemanda  string          `json:"estado_legal_demanda"`
	Etapa1JSON          datatypes.JSON  `json:"etapa_1_json"`
	Etapa2JSON          datatypes.JSON  `json:"etapa_2_json"`
	Etapa3JSON          datatypes.JSON  `json:"etapa_3_json"`
	Etapa4JSON          datatypes.JSON  `json:"etapa_4_json"`
	FechaEstadoSeguimiento time.Time    `json:"fecha_estado_seguimiento"`
	FechaEstadoLegal       time.Time    `json:"fecha_estado_legal"`
}

// GetSeguimientosByAbogadoHandler retorna los casos asignados al abogado actual (basado en sesión)
func GetSeguimientosByAbogadoHandler(c *fiber.Ctx) error {
	userID := getUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"detail": "No se pudo identificar al usuario"})
	}

	// Buscar el bufete asociado a este usuario
	var bufete models.Bufete
	if err := db.DB.Where("usuario_id = ?", userID).First(&bufete).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "No se encontró un bufete/abogado asociado a este usuario"})
	}

	var seguimientos []models.Seguimiento
	if err := db.DB.Preload("Demanda").Where("id_abogado = ?", bufete.ID).Find(&seguimientos).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error al buscar seguimientos"})
	}

	// Mapear a DTO para ocultar campos financieros y simplificar la demanda
	response := make([]SeguimientoAbogadoResponse, len(seguimientos))
	for i, s := range seguimientos {
		simple := DemandaSimple{
			ID:           s.IDDemanda,
			NoCredito:    s.Demanda.NoCredito,
			NoCreditoT24: s.Demanda.NoCreditoT24,
			Deudor:       s.Demanda.Deudor,
			NoJuicio:     s.Demanda.NoJuicio,
		}

		response[i] = SeguimientoAbogadoResponse{
			ID:                  s.ID,
			IDDemanda:           s.IDDemanda,
			Demanda:             simple,
			EstadoSeguimiento:   s.EstadoSeguimiento,
			EstadoLegalDemanda:  s.EstadoLegalDemanda,
			Etapa1JSON:          s.Etapa1JSON,
			Etapa2JSON:          s.Etapa2JSON,
			Etapa3JSON:          s.Etapa3JSON,
			Etapa4JSON:          s.Etapa4JSON,
			FechaEstadoSeguimiento: s.FechaEstadoSeguimiento,
			FechaEstadoLegal:       s.FechaEstadoLegal,
		}
	}

	return c.JSON(response)
}

// AddComentarioHandler agrega un comentario a la bitácora de la etapa actual y un archivo opcional
func AddComentarioHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	
	// Leer valores del formulario en lugar de JSON
	etapaStr := c.FormValue("etapa")
	comentario := c.FormValue("comentario")
	
	if etapaStr == "" || comentario == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Etapa y comentario son requeridos"})
	}

	etapaInt := 0
	fmt.Sscanf(etapaStr, "%d", &etapaInt)

	var seguimiento models.Seguimiento
	if err := db.DB.Preload("Demanda").First(&seguimiento, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Seguimiento no encontrado"})
	}

	// Validaciones de negocio
	if seguimiento.EstadoLegalDemanda != "Vigente" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"detail": "El proceso no está vigente para edición"})
	}

	if etapaInt != seguimiento.EstadoSeguimiento {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"detail": "Solo se pueden agregar comentarios a la etapa activa"})
	}

	// Manejo de archivo opcional con Google Cloud Storage
	documentoPath := ""
	file, err := c.FormFile("archivo")
	if err == nil && file != nil { // El archivo fue enviado
		deudorNombre := "Desconocido"
		if seguimiento.Demanda != nil && seguimiento.Demanda.Deudor != nil {
			deudorNombre = *seguimiento.Demanda.Deudor
		}

		// Crear nombre de carpeta único con nombre sanitizado y ID Demanda
		folderName := fmt.Sprintf("Demandas_Deudores/%s_%d", sanitizeName(deudorNombre), seguimiento.IDDemanda)
		
		timestamp := time.Now().UnixNano()
		filename := fmt.Sprintf("%s/evidencia_%s_%d.pdf", folderName, id, timestamp)
		
		// Subir a GCS
		gcsPath, err := gcs.UploadFile(file, filename)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error al subir documento a la nube", "error": err.Error()})
		}
		
		// Guardar el object path de GCS para la bitácora
		documentoPath = gcsPath
	}

	// Estructura del comentario
	nuevoComentario := map[string]string{
		"fecha":      time.Now().Format("2006-01-02"),
		"comentario": comentario,
	}
	if documentoPath != "" {
		nuevoComentario["documento"] = documentoPath
	}

	// Obtener la bitácora actual
	var bitacoraActual datatypes.JSON
	switch etapaInt {
	case 1: bitacoraActual = seguimiento.Etapa1JSON
	case 2: bitacoraActual = seguimiento.Etapa2JSON
	case 3: bitacoraActual = seguimiento.Etapa3JSON
	case 4: bitacoraActual = seguimiento.Etapa4JSON
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "Etapa inválida"})
	}

	var comentarios []interface{}
	json.Unmarshal(bitacoraActual, &comentarios)
	comentarios = append(comentarios, nuevoComentario)
	
	nuevaBitacora, _ := json.Marshal(comentarios)

	// Actualizar el campo correspondiente en el struct y guardar
	switch etapaInt {
	case 1:
		seguimiento.Etapa1JSON = datatypes.JSON(nuevaBitacora)
	case 2:
		seguimiento.Etapa2JSON = datatypes.JSON(nuevaBitacora)
	case 3:
		seguimiento.Etapa3JSON = datatypes.JSON(nuevaBitacora)
	case 4:
		seguimiento.Etapa4JSON = datatypes.JSON(nuevaBitacora)
	}

	if err := db.DB.Save(&seguimiento).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"detail": "Error al guardar el comentario en la base de datos",
			"error":  err.Error(),
		})
	}

	return c.JSON(fiber.Map{"status": "ok", "comentario": nuevoComentario})
}

// AvanzarEtapaHandler incrementa el estado del seguimiento
func AvanzarEtapaHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var seguimiento models.Seguimiento
	if err := db.DB.First(&seguimiento, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Seguimiento no encontrado"})
	}

	if seguimiento.EstadoLegalDemanda != "Vigente" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"detail": "El proceso no está vigente"})
	}

	if seguimiento.EstadoSeguimiento >= 5 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "El proceso ya ha finalizado"})
	}

	seguimiento.EstadoSeguimiento++
	seguimiento.FechaEstadoSeguimiento = time.Now()

	if seguimiento.EstadoSeguimiento == 5 {
		seguimiento.EstadoLegalDemanda = "Finalizado"
		seguimiento.FechaEstadoLegal = time.Now()
	}

	if err := db.DB.Save(&seguimiento).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error al avanzar etapa"})
	}

	return c.JSON(seguimiento)
}

// DesistirSeguimientoHandler marca el proceso como desistido
func DesistirSeguimientoHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var seguimiento models.Seguimiento
	if err := db.DB.First(&seguimiento, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"detail": "Seguimiento no encontrado"})
	}

	seguimiento.EstadoLegalDemanda = "Desistido"
	seguimiento.FechaEstadoLegal = time.Now()

	// Zero out payments for the current stage and all future stages
	// If the lawyer is in stage N, they haven't "finished" stage N, so we zero it out too.
	// We only keep payments for stages < N (the ones truly completed).
	if seguimiento.EstadoSeguimiento <= 4 {
		if seguimiento.EstadoSeguimiento <= 4 {
			seguimiento.PagoPactado4 = 0
		}
		if seguimiento.EstadoSeguimiento <= 3 {
			seguimiento.PagoPactado3 = 0
		}
		if seguimiento.EstadoSeguimiento <= 2 {
			seguimiento.PagoPactado2 = 0
		}
		if seguimiento.EstadoSeguimiento <= 1 {
			seguimiento.PagoPactado1 = 0
		}
	}

	if err := db.DB.Save(&seguimiento).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error al desistir seguimiento"})
	}

	return c.JSON(seguimiento)
}

// Helper para obtener ID de usuario
func getUserID(c *fiber.Ctx) int {
	rawAuth := c.Get("Authorization")
	if rawAuth == "" { return 0 }
	token := strings.Replace(rawAuth, "Bearer ", "", 1)
	claims, err := auth.VerifyToken(token)
	if err != nil { return 0 }
	sub := fmt.Sprintf("%v", claims["sub"])
	uid, _ := strconv.Atoi(sub)
	return uid
}
// Funciones auxiliares

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "_")
	// Eliminar caracteres que no sean alfanuméricos o guiones bajos
	reg := regexp.MustCompile("[^a-zA-Z0-9_]+")
	return reg.ReplaceAllString(name, "")
}

// GetSignedURLHandler retorna una URL firmada de Google Cloud Storage para acceso temporal
func GetSignedURLHandler(c *fiber.Ctx) error {
	path := c.Query("path")
	if path == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"detail": "La ruta del documento es requerida"})
	}

	url, err := gcs.GenerateSignedURL(path)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"detail": "Error generando enlace seguro", "error": err.Error()})
	}

	return c.JSON(fiber.Map{"url": url})
}
