package models

import (
	"time"
)

type Demanda struct {
	ID                  uint         `gorm:"primaryKey" json:"id"`
	IDAgencia           *int         `json:"id_agencia"`
	Agencia             *Agencia     `gorm:"foreignKey:IDAgencia" json:"agencia,omitempty"`
	NoCredito           *string      `gorm:"size:100;index" json:"no_credito"`
	CIF                 *string      `gorm:"size:50;index" json:"cif"`
	CodigoCliente       *string      `gorm:"size:100" json:"codigo_cliente"`
	NoCreditoT24        *string      `gorm:"size:100" json:"no_credito_t24"`
	Deudor              *string      `gorm:"size:255;index" json:"deudor"`
	Fiadores            *string      `gorm:"type:text" json:"fiadores"`
	SalarioEmbargadoA   *string      `gorm:"type:text" json:"salario_embargado_a"`
	NoJuicio            *string      `gorm:"size:255" json:"no_juicio"`
	FechaIngresoDemanda *string      `gorm:"size:100" json:"fecha_ingreso_demanda"`
	AbogadoNombreExcel  *string      `gorm:"size:255" json:"abogado_nombre_excel"`
	MontoDemanda        *float64     `json:"monto_demanda"`
	Situacion           *string      `gorm:"size:255" json:"situacion"`
	FormaResolucion     *string      `gorm:"size:255" json:"forma_resolucion"`
	CostasJudiciales    *float64     `json:"costas_judiciales"`
	CostasRecuperadas   *string      `gorm:"size:255" json:"costas_recuperadas"`
	Observacion1        *string      `gorm:"type:text" json:"observacion_1"`
	EstadoLegal         *string      `gorm:"size:100" json:"estado_legal"`
	SeguimientoLegacy   *string      `gorm:"type:text" json:"seguimiento_legacy"`
	Observacion2        *string      `gorm:"type:text" json:"observacion_2"`
	
	// Relación uno-a-uno con Seguimiento (el proceso legal centralizado)
	Seguimiento         *Seguimiento `gorm:"foreignKey:IDDemanda" json:"seguimiento,omitempty"`
	
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
}

func (Demanda) TableName() string {
	return "demandas"
}
