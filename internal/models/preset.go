package models

import "time"

type Preset struct {
	ID                     int       `gorm:"primaryKey" json:"id"`
	Nombre                 *string   `gorm:"size:150;not null" json:"nombre"`
	RangoMin               *float64  `json:"rango_min"`
	RangoMax               *float64  `json:"rango_max"`
	PorcentajeComision     *float64  `json:"porcentaje_comision"` // El % total (ej. 10.0)
	PorcentajeEtapa1       *float64  `json:"p_etapa_1"`           // Presentación
	PorcentajeEtapa2       *float64  `json:"p_etapa_2"`           // Admisión
	PorcentajeEtapa3       *float64  `json:"p_etapa_3"`           // Notificación
	PorcentajeEtapa4       *float64  `json:"p_etapa_4"`           // Ejecución
	MontoPagoUnico         *float64  `json:"monto_pago_unico"`    // Para casos de Seguro o montos fijos
	MontoDesestimacionBase *float64  `json:"monto_desestimacion"`
	Descripcion            *string   `gorm:"type:text" json:"descripcion"`
	Activo                 bool      `gorm:"default:true" json:"activo"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func (Preset) TableName() string {
	return "presets"
}
