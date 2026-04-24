package models

import (
	"time"
)

type Demanda struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	IDAgencia           *int       `json:"id_agencia"` // Mapeado de "Central", etc.
	Agencia             *Agencia   `gorm:"foreignKey:IDAgencia" json:"agencia,omitempty"`
	NoCredito           *string    `gorm:"size:50;index" json:"no_credito"`
	CIF                 *string    `gorm:"size:20;index" json:"cif"`
	CodigoCliente       *string    `gorm:"size:50" json:"codigo_cliente"` // Antes IDExterno
	NoCreditoT24        *string    `gorm:"size:50" json:"no_credito_t24"`
	Deudor              *string    `gorm:"size:255;index" json:"deudor"`
	Fiadores            *string    `gorm:"type:text" json:"fiadores"`
	SalarioEmbargadoA   *string    `gorm:"size:255" json:"salario_embargado_a"`
	NoJuicio            *string    `gorm:"size:100" json:"no_juicio"`
	FechaIngresoDemanda *string    `gorm:"size:50" json:"fecha_ingreso_demanda"` // Texto para evitar errores de parseo inicial
	AbogadoNombreExcel  *string    `gorm:"size:255" json:"abogado_nombre_excel"` // El nombre tal cual viene en el Excel
	MontoDemanda        *float64   `json:"monto_demanda"`
	Situacion           *string    `gorm:"size:100" json:"situacion"`
	FormaResolucion     *string    `gorm:"size:100" json:"forma_resolucion"`
	CostasJudiciales    *float64   `json:"costas_judiciales"`
	CostasRecuperadas   *string    `gorm:"size:10" json:"costas_recuperadas"` // "SI" o "NO"
	Observacion1        *string    `gorm:"type:text" json:"observacion_1"`
	EstadoLegal         *string    `gorm:"size:50" json:"estado_legal"` // Vigente, Desistido, etc.
	SeguimientoLegacy   *string    `gorm:"type:text" json:"seguimiento_legacy"` // Columna "SEGUIMIENTO" de Excel
	Observacion2        *string    `gorm:"type:text" json:"observacion_2"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (Demanda) TableName() string {
	return "demandas"
}
