package handlers

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/DevSoft-RECO/backend-creditos-go/internal/db"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/models"
	"github.com/gofiber/fiber/v2"
)

type DashboardStats struct {
	TotalDemandas   int64   `json:"total_demandas"`
	TotalRecuperado float64 `json:"total_recuperado"`
	TotalPendiente  float64 `json:"total_pendiente"`
	PagosPendientes int64   `json:"pagos_pendientes"`
	AbogadosActivos int64   `json:"abogados_activos"`
	// Distribución por estado legal
	CasosPendientes  int64 `json:"casos_pendientes"`
	CasosActivos     int64 `json:"casos_activos"`
	CasosCancelados  int64 `json:"casos_cancelados"`
	CasosFinalizados int64 `json:"casos_finalizados"`
	// Montos globales
	CapitalEnRiesgo float64 `json:"capital_en_riesgo"`
	// Secciones
	Stages          []Stage  `json:"stages"`
	RecentMovements []Move   `json:"recent_movements"`
	TopLawyers      []Lawyer `json:"top_lawyers"`
}

type Lawyer struct {
	Name            string  `json:"name"`
	Initials        string  `json:"initials"`
	Performance     string  `json:"performance"`
	CasosAsignados  int64   `json:"casos_asignados"`
	CasosAvanzados  int64   `json:"casos_avanzados"`
	TotalDesembolso float64 `json:"total_desembolso"`
}

type Stage struct {
	Name    string  `json:"name"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

type Move struct {
	ID     string `json:"id"`
	Deudor string `json:"deudor"`
	Stage  string `json:"stage"`
	Amount string `json:"amount"`
	Date   string `json:"date"`
}

func GetDashboardStats(c *fiber.Ctx) error {
	var stats DashboardStats

	// 1. Total Demandas
	db.DB.Model(&models.Demanda{}).Count(&stats.TotalDemandas)

	// 2. Capital en Riesgo (Solo demandas con seguimiento Vigente)
	db.DB.Model(&models.Demanda{}).
		Joins("JOIN seguimientos ON seguimientos.id_demanda = demandas.id").
		Where("seguimientos.estado_legal_demanda = ?", "Vigente").
		Select("COALESCE(SUM(demandas.monto_demanda), 0)").Scan(&stats.CapitalEnRiesgo)

	// 3. Abogados Activos (bufetes distintos con al menos un caso vigente)
	db.DB.Model(&models.Seguimiento{}).
		Where("estado_legal_demanda = ?", "Vigente").
		Distinct("id_abogado").Count(&stats.AbogadosActivos)

	// 4. Obtener todos los seguimientos para cálculos financieros
	var seguimientos []models.Seguimiento
	db.DB.Preload("Demanda").Find(&seguimientos)

	totalSeguimientos := int64(len(seguimientos))

	var totalRecuperado float64
	var totalPendiente float64
	var pagosPendientesCount int64

	for _, s := range seguimientos {
		// Sumar montos recuperados (pagados)
		if s.IsPagadoUnico {
			totalRecuperado += s.PagoUnico
		}
		if s.IsPagadoDesestimacion {
			totalRecuperado += s.MontoDesestimacion
		}
		if s.IsPagado1 {
			totalRecuperado += s.PagoPactado1
		}
		if s.IsPagado2 {
			totalRecuperado += s.PagoPactado2
		}
		if s.IsPagado3 {
			totalRecuperado += s.PagoPactado3
		}
		if s.IsPagado4 {
			totalRecuperado += s.PagoPactado4
		}

		// Sumar montos pendientes (pactados pero no pagados)
		if !s.IsPagado1 && s.PagoPactado1 > 0 {
			totalPendiente += s.PagoPactado1
			pagosPendientesCount++
		}
		if !s.IsPagado2 && s.PagoPactado2 > 0 {
			totalPendiente += s.PagoPactado2
			pagosPendientesCount++
		}
		if !s.IsPagado3 && s.PagoPactado3 > 0 {
			totalPendiente += s.PagoPactado3
			pagosPendientesCount++
		}
		if !s.IsPagado4 && s.PagoPactado4 > 0 {
			totalPendiente += s.PagoPactado4
			pagosPendientesCount++
		}
		if !s.IsPagadoUnico && s.PagoUnico > 0 {
			totalPendiente += s.PagoUnico
			pagosPendientesCount++
		}
		if !s.IsPagadoDesestimacion && s.MontoDesestimacion > 0 {
			totalPendiente += s.MontoDesestimacion
			pagosPendientesCount++
		}
	}

	stats.TotalRecuperado = math.Round(totalRecuperado)
	stats.TotalPendiente = math.Round(totalPendiente)
	stats.PagosPendientes = pagosPendientesCount

	// 5. Distribución por estado legal
	// Pendientes: Demandas que no tienen registro en seguimientos
	db.DB.Model(&models.Demanda{}).
		Joins("LEFT JOIN seguimientos ON seguimientos.id_demanda = demandas.id").
		Where("seguimientos.id IS NULL").
		Count(&stats.CasosPendientes)
		
	// Activos: Seguimientos Vigentes
	db.DB.Model(&models.Seguimiento{}).Where("estado_legal_demanda = ?", "Vigente").Count(&stats.CasosActivos)
	
	// Cancelados: Seguimientos Cancelados o Suspendidos
	db.DB.Model(&models.Seguimiento{}).Where("estado_legal_demanda IN ?", []string{"Cancelado", "Suspendido"}).Count(&stats.CasosCancelados)
	
	// Finalizados: Seguimientos Finalizados y Desistidos (según requerimiento de agruparlos)
	db.DB.Model(&models.Seguimiento{}).Where("estado_legal_demanda IN ?", []string{"Finalizado", "Desistido"}).Count(&stats.CasosFinalizados)

	// 6. Distribución por Etapas (solo seguimientos vigentes, porcentaje sobre total seguimientos)
	stageNames := map[int]string{1: "Presentación", 2: "Admisión", 3: "Notificación", 4: "Ejecución"}
	for i := 1; i <= 4; i++ {
		var count int64
		db.DB.Model(&models.Seguimiento{}).Where("estado_seguimiento = ?", i).Count(&count)

		percent := 0.0
		if totalSeguimientos > 0 {
			percent = math.Round((float64(count)/float64(totalSeguimientos))*10000) / 100
		}

		stats.Stages = append(stats.Stages, Stage{
			Name:    stageNames[i],
			Count:   count,
			Percent: percent,
		})
	}

	// 7. Últimos Movimientos Financieros REALES (solo registros con pagos efectuados)
	type pagoInfo struct {
		expediente string
		deudor     string
		etapa      string
		monto      float64
		fecha      time.Time
	}

	var pagosReales []pagoInfo

	for _, s := range seguimientos {
		expediente := "S/N"
		deudor := "N/A"
		if s.Demanda != nil {
			if s.Demanda.NoJuicio != nil {
				expediente = *s.Demanda.NoJuicio
			} else if s.Demanda.NoCreditoT24 != nil {
				expediente = *s.Demanda.NoCreditoT24
			} else if s.Demanda.NoCredito != nil {
				expediente = *s.Demanda.NoCredito
			}
			if s.Demanda.Deudor != nil {
				deudor = *s.Demanda.Deudor
			}
		}

		if s.IsPagado1 && s.FechaPago1 != nil {
			pagosReales = append(pagosReales, pagoInfo{expediente, deudor, "Presentación", s.PagoPactado1, *s.FechaPago1})
		}
		if s.IsPagado2 && s.FechaPago2 != nil {
			pagosReales = append(pagosReales, pagoInfo{expediente, deudor, "Admisión", s.PagoPactado2, *s.FechaPago2})
		}
		if s.IsPagado3 && s.FechaPago3 != nil {
			pagosReales = append(pagosReales, pagoInfo{expediente, deudor, "Notificación", s.PagoPactado3, *s.FechaPago3})
		}
		if s.IsPagado4 && s.FechaPago4 != nil {
			pagosReales = append(pagosReales, pagoInfo{expediente, deudor, "Ejecución", s.PagoPactado4, *s.FechaPago4})
		}
		if s.IsPagadoUnico && s.FechaPagoUnico != nil {
			pagosReales = append(pagosReales, pagoInfo{expediente, deudor, "Pago Único", s.PagoUnico, *s.FechaPagoUnico})
		}
		if s.IsPagadoDesestimacion && s.FechaPagoDesestimacion != nil {
			pagosReales = append(pagosReales, pagoInfo{expediente, deudor, "Desestimación", s.MontoDesestimacion, *s.FechaPagoDesestimacion})
		}
	}

	// Ordenar por fecha más reciente
	sort.Slice(pagosReales, func(i, j int) bool {
		return pagosReales[i].fecha.After(pagosReales[j].fecha)
	})

	// Tomar los 8 más recientes
	limit := 8
	if len(pagosReales) < limit {
		limit = len(pagosReales)
	}

	for _, p := range pagosReales[:limit] {
		stats.RecentMovements = append(stats.RecentMovements, Move{
			ID:     p.expediente,
			Deudor: p.deudor,
			Stage:  p.etapa,
			Amount: fmt.Sprintf("Q %d", int(math.Round(p.monto))),
			Date:   p.fecha.Format("02/01/2006"),
		})
	}

	// 8. Eficiencia Legal por Bufete (datos reales)
	var bufetes []models.Bufete
	db.DB.Find(&bufetes)

	for _, b := range bufetes {
		var totalCasos int64
		var casosAvanzados int64 // Etapa >= 2 o Finalizado

		db.DB.Model(&models.Seguimiento{}).Where("id_abogado = ?", b.ID).Count(&totalCasos)

		if totalCasos == 0 {
			continue
		}

		// Éxito = casos que avanzaron más allá de la primera etapa O fueron finalizados exitosamente
		db.DB.Model(&models.Seguimiento{}).
			Where("id_abogado = ? AND (estado_seguimiento >= 2 OR estado_legal_demanda = 'Finalizado')", b.ID).
			Count(&casosAvanzados)

		// Calcular total desembolsado para este bufete
		var bufeteSeguimientos []models.Seguimiento
		db.DB.Where("id_abogado = ?", b.ID).Find(&bufeteSeguimientos)

		var totalDesembolso float64
		for _, s := range bufeteSeguimientos {
			if s.IsPagadoUnico {
				totalDesembolso += s.PagoUnico
			}
			if s.IsPagadoDesestimacion {
				totalDesembolso += s.MontoDesestimacion
			}
			if s.IsPagado1 {
				totalDesembolso += s.PagoPactado1
			}
			if s.IsPagado2 {
				totalDesembolso += s.PagoPactado2
			}
			if s.IsPagado3 {
				totalDesembolso += s.PagoPactado3
			}
			if s.IsPagado4 {
				totalDesembolso += s.PagoPactado4
			}
		}

		efficiency := (float64(casosAvanzados) / float64(totalCasos)) * 100

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
			Name:            nombreStr,
			Initials:        initials,
			Performance:     fmt.Sprintf("%.0f%%", efficiency),
			CasosAsignados:  totalCasos,
			CasosAvanzados:  casosAvanzados,
			TotalDesembolso: math.Round(totalDesembolso),
		})
	}

	// Ordenar bufetes por eficiencia descendente
	sort.Slice(stats.TopLawyers, func(i, j int) bool {
		return stats.TopLawyers[i].CasosAvanzados > stats.TopLawyers[j].CasosAvanzados
	})

	return c.JSON(stats)
}
