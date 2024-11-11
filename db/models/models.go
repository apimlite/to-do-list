package models

import (
	"time"

	"gorm.io/gorm"
)

// Customer represents the customers table
type Customer struct {
	CustomerIdentifier string        `gorm:"column:customer_identifier;primaryKey;type:varchar(255)" json:"customer_identifier"`
	AWSAccountID       string        `gorm:"column:aws_account_id;not null;type:varchar(255)" json:"aws_account_id"`
	Name               string        `gorm:"column:name;type:varchar(255)" json:"name"`
	Email              string        `gorm:"column:email;type:varchar(255)" json:"email"`
	Phone              string        `gorm:"column:phone;type:varchar(50)" json:"phone"`
	JobRole            string        `gorm:"column:job_role;type:varchar(100)" json:"job_role"`
	Company            string        `gorm:"column:company;type:varchar(255)" json:"company"`
	Country            string        `gorm:"column:country;type:varchar(100)" json:"country"`
	Entitlements       []Entitlement `gorm:"foreignKey:CustomerIdentifier" json:"entitlements,omitempty"`
}

// TableName specifies the table name for Customer
func (Customer) TableName() string {
	return "customers"
}

// Product represents the products table
type Product struct {
	ProductCode  string        `gorm:"column:product_code;primaryKey;type:varchar(255)" json:"product_code"`
	ProductID    string        `gorm:"column:product_id;type:varchar(255)" json:"product_id"`
	ProductName  string        `gorm:"column:product_name;type:varchar(255)" json:"product_name"`
	Entitlements []Entitlement `gorm:"foreignKey:ProductCode" json:"entitlements,omitempty"`
}

// TableName specifies the table name for Product
func (Product) TableName() string {
	return "products"
}

// ValueType represents the possible types for entitlement values
type ValueType string

const (
	ValueTypeBoolean ValueType = "boolean"
	ValueTypeDouble  ValueType = "double"
	ValueTypeInteger ValueType = "integer"
	ValueTypeString  ValueType = "string"
)

// EntitlementValue represents the entitlement_values table
type EntitlementValue struct {
	ValueID      int64     `gorm:"column:value_id;primaryKey;autoIncrement" json:"value_id"`
	BooleanValue *bool     `gorm:"column:boolean_value;type:boolean" json:"boolean_value,omitempty"`
	DoubleValue  *float64  `gorm:"column:double_value;type:double" json:"double_value,omitempty"`
	IntegerValue *int64    `gorm:"column:integer_value;type:int" json:"integer_value,omitempty"`
	StringValue  *string   `gorm:"column:string_value;type:varchar(255)" json:"string_value,omitempty"`
	ValueType    ValueType `gorm:"column:value_type;type:enum('boolean','double','integer','string')" json:"value_type"`
}

// TableName specifies the table name for EntitlementValue
func (EntitlementValue) TableName() string {
	return "entitlement_values"
}

// Entitlement represents the entitlements table
type Entitlement struct {
	EntitlementID      int64             `gorm:"column:entitlement_id;primaryKey;autoIncrement" json:"entitlement_id"`
	CustomerIdentifier string            `gorm:"column:customer_identifier;not null;type:varchar(255)" json:"customer_identifier"`
	ProductCode        string            `gorm:"column:product_code;not null;type:varchar(255)" json:"product_code"`
	Dimension          string            `gorm:"column:dimension;type:varchar(255)" json:"dimension"`
	ExpirationDate     string            `gorm:"column:expiration_date;type:varchar(255)" json:"expiration_date"`
	ValueID            int64             `gorm:"column:value_id" json:"value_id"`
	CreatedAt          time.Time         `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt          time.Time         `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
	Customer           *Customer         `gorm:"foreignKey:CustomerIdentifier" json:"customer,omitempty"`
	Product            *Product          `gorm:"foreignKey:ProductCode" json:"product,omitempty"`
	Value              *EntitlementValue `gorm:"foreignKey:ValueID" json:"value,omitempty"`
}

// TableName specifies the table name for Entitlement
func (Entitlement) TableName() string {
	return "entitlements"
}

// BeforeCreate hook for Entitlement to set CreatedAt and UpdatedAt
func (e *Entitlement) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now
	return nil
}

// BeforeUpdate hook for Entitlement to update UpdatedAt
func (e *Entitlement) BeforeUpdate(tx *gorm.DB) error {
	e.UpdatedAt = time.Now()
	return nil
}
