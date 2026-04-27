package handlers

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/db"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/models"
)

type DashboardStats struct {
	TotalDemandas     int64   `json:"total_demandas"`
	TotalRecuperado   float64 `json:"total_recuperado"`
	PagosPendientes   int64   `json:"pagos_pendientes"`
	AbogadosActivos   int64   `json:"abogados_activos"`
	Stages            []Stage `json:"stages"`
	RecentMovements   []Move  `json:"recent_movements"`
	TopLawyers        []Lawyer `json:"top_lawyers"`
}

type Lawyer struct {
	Name        string `json:"name"`
	Initials    string `json:"initials"`
	Performance string `json:"performance"`
}

type Stage struct {
	Name    string  `json:"name"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

type Move struct {
	ID     string `json:"id"`
	Stage  string `json:"stage"`
	Amount string `json:"amount"`
	Date   string `json:"date"`
}

func GetDashboardStats(c *fiber.Ctx) error {
	var stats DashboardStats

	// 1. Total Demandas
	db.DB.Model(&models.Demanda{}).Count(&stats.TotalDemandas)

	// 2. Abogados Activos
	db.DB.Model(&models.Seguimiento{}).Distinct("id_abogado").Count(&stats.AbogadosActivos)

	// 3. Cálculos Financieros (Recuperado y Pendientes)
	var seguimientos []models.Seguimiento
	db.DB.Find(&seguimientos)

	var totalRecuperado float64 = 0
	var pagosPendientesCount int64 = 0

	for _, s := range seguimientos {
		// Recuperado
		if s.IsPagadoUnico { totalRecuperado += s.PagoUnico }
		if s.IsPagadoDesestimacion { totalRecuperado += s.MontoDesestimacion }
		if s.IsPagado1 { totalRecuperado += s.PagoPactado1 }
		if s.IsPagado2 { totalRecuperado += s.PagoPactado2 }
		if s.IsPagado3 { totalRecuperado += s.PagoPactado3 }
		if s.IsPagado4 { totalRecuperado += s.PagoPactado4 }

		// Pendientes
		if !s.IsPagado1 && s.PagoPactado1 > 0 { pagosPendientesCount++ }
		if !s.IsPagado2 && s.PagoPactado2 > 0 { pagosPendientesCount++ }
		if !s.IsPagado3 && s.PagoPactado3 > 0 { pagosPendientesCount++ }
		if !s.IsPagado4 && s.PagoPactado4 > 0 { pagosPendientesCount++ }
		if !s.IsPagadoUnico && s.PagoUnico > 0 { pagosPendientesCount++ }
	}

	stats.TotalRecuperado = totalRecuperado
	stats.PagosPendientes = pagosPendientesCount

	// 4. Distribución por Etapas
	stageNames := map[int]string{1: "Presentación", 2: "Admisión", 3: "Notificación", 4: "Ejecución"}
	for i := 1; i <= 4; i++ {
		var count int64
		db.DB.Model(&models.Seguimiento{}).Where("estado_seguimiento = ?", i).Count(&count)
		
		percent := 0.0
		if stats.TotalDemandas > 0 {
			percent = (float64(count) / float64(stats.TotalDemandas)) * 100
		}

		stats.Stages = append(stats.Stages, Stage{
			Name:    stageNames[i],
			Count:   count,
			Percent: percent,
		})
	}

	// 5. Últimos Movimientos (Simulado basado en los seguimientos más recientes con pagos)
	// En un sistema real, esto vendría de una tabla de 'pagos' o 'transacciones'
	var recentSegs []models.Seguimiento
	db.DB.Preload("Demanda").Order("updated_at desc").Limit(5).Find(&recentSegs)

	for _, rs := range recentSegs {
		label := "Gestión"
		amount := 0.0
		if rs.IsPagado4 { 
			label = "Ejecución"; amount = rs.PagoPactado4 
		} else if rs.IsPagado3 { 
			label = "Notificación"; amount = rs.PagoPactado3 
		} else if rs.IsPagado2 { 
			label = "Admisión"; amount = rs.PagoPactado2 
		} else if rs.IsPagado1 { 
			label = "Presentación"; amount = rs.PagoPactado1 
		} else if rs.IsPagadoUnico { 
			label = "Pago Único"; amount = rs.PagoUnico 
		}

		if rs.Demanda != nil {
			expediente := "S/N"
			if rs.Demanda.NoJuicio != nil {
				expediente = *rs.Demanda.NoJuicio
			} else if rs.Demanda.NoCredito != nil {
				expediente = *rs.Demanda.NoCredito
			}

			stats.RecentMovements = append(stats.RecentMovements, Move{
				ID:     expediente,
				Stage:  label,
				Amount: fmt.Sprintf("Q %.2f", amount),
				Date:   rs.UpdatedAt.Format("02/01/2006"),
			})
		}
	}

	// 6. Eficiencia Legal (Top Bufetes)
	var bufetes []models.Bufete
	db.DB.Find(&bufetes)

	for _, b := range bufetes {
		var totalCasos int64
		var casosExitosos int64
		
		db.DB.Model(&models.Seguimiento{}).Where("id_abogado = ?", b.ID).Count(&totalCasos)
		
		if totalCasos > 0 {
			// Definimos éxito como casos en etapa 3 o 4, o con algún pago realizado
			db.DB.Model(&models.Seguimiento{}).Where("id_abogado = ? AND (estado_seguimiento >= 3 OR is_pagado_1 = true)", b.ID).Count(&casosExitosos)
			
			efficiency := (float64(casosExitosos) / float64(totalCasos)) * 100
			
			initials := ""
			nombreStr := "Sin Nombre"
			if b.Nombre != nil && len(*b.Nombre) > 0 {
				nombreStr = *b.Nombre
				initials = string(nombreStr[0])
				for i, char := range nombreStr {
					if char == ' ' && i+1 < len(nombreStr) {
						initials += string(nombreStr[i+1])
						break
					}
				}
			}

			stats.TopLawyers = append(stats.TopLawyers, Lawyer{
				Name:        nombreStr,
				Initials:    initials,
				Performance: fmt.Sprintf("%.0f%%", efficiency),
			})
		}
	}

	return c.JSON(stats)
}
