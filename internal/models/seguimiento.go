package models

import (
	"time"
	"gorm.io/datatypes"
)

type Seguimiento struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	IDDemanda          uint           `gorm:"not null;index" json:"id_demanda"`
	Demanda            *Demanda       `gorm:"foreignKey:IDDemanda" json:"demanda,omitempty"`
	IDAbogado          uint           `gorm:"not null;index" json:"id_abogado"`
	Abogado            *Bufete        `gorm:"foreignKey:IDAbogado" json:"abogado,omitempty"`
	
	// Estados
	EstadoSeguimiento  int            `gorm:"default:1" json:"estado_seguimiento"` // 1: Presentación, 2: Admisión, 3: Notificación, 4: Ejecución
	EstadoLegalDemanda string         `gorm:"size:50;default:'Vigente'" json:"estado_legal_demanda"`
	
	// Snapshot Financiero Global
	PorcentajeDemanda  float64        `json:"porcentaje_demanda"`
	PagoUnico          float64        `json:"pago_unico"`
	MontoDesestimacion float64        `json:"monto_desestimacion"`

	// Etapa 1: Presentación
	Etapa1JSON         datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"etapa_1_json"`
	PagoSugerido1      float64        `json:"pago_sugerido_1"`
	PagoPactado1       float64        `json:"pago_pactado_1"`
	IsPagado1          bool           `gorm:"default:false" json:"is_pagado_1"`

	// Etapa 2: Admisión
	Etapa2JSON         datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"etapa_2_json"`
	PagoSugerido2      float64        `json:"pago_sugerido_2"`
	PagoPactado2       float64        `json:"pago_pactado_2"`
	IsPagado2          bool           `gorm:"default:false" json:"is_pagado_2"`

	// Etapa 3: Notificación
	Etapa3JSON         datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"etapa_3_json"`
	PagoSugerido3      float64        `json:"pago_sugerido_3"`
	PagoPactado3       float64        `json:"pago_pactado_3"`
	IsPagado3          bool           `gorm:"default:false" json:"is_pagado_3"`

	// Etapa 4: Ejecución
	Etapa4JSON         datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"etapa_4_json"`
	PagoSugerido4      float64        `json:"pago_sugerido_4"`
	PagoPactado4       float64        `json:"pago_pactado_4"`
	IsPagado4          bool           `gorm:"default:false" json:"is_pagado_4"`

	// Auditoría
	FechaEstadoSeguimiento time.Time  `json:"fecha_estado_seguimiento"`
	FechaEstadoLegal       time.Time  `json:"fecha_estado_legal"`
	FechaPago1             *time.Time `json:"fecha_pago_1"`
	FechaPago2             *time.Time `json:"fecha_pago_2"`
	FechaPago3             *time.Time `json:"fecha_pago_3"`
	FechaPago4             *time.Time `json:"fecha_pago_4"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

func (Seguimiento) TableName() string {
	return "seguimientos"
}
