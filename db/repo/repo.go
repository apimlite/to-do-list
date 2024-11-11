package repo

import (
	"aws-markertplace-integration/db/models"
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/marketplacemetering"
	"gorm.io/gorm"
)

// Repository errors
var (
	ErrCustomerNotFound = errors.New("customer not found")
	ErrProductNotFound  = errors.New("product not found")
	ErrInvalidValue     = errors.New("invalid entitlement value")
)

// CustomerBasicInfo represents the initial customer data
// type CustomerBasicInfo struct {
// 	// *marketplacemetering.ResolveCustomerOutput
// 	CustomerAWSAccountId string `json:"CustomerAWSAccountId"`
// 	CustomerIdentifier   string `json:"CustomerIdentifier"`
// 	ProductCode          string `json:"ProductCode"`
// }

type CustomerBasicInfo = marketplacemetering.ResolveCustomerOutput

// EntitlementValue represents the value structure in API response
type EntitlementValueAWS struct {
	BooleanValue *bool    `json:"BooleanValue,omitempty"`
	DoubleValue  *float64 `json:"DoubleValue,omitempty"`
	IntegerValue *int     `json:"IntegerValue,omitempty"`
	StringValue  *string  `json:"StringValue,omitempty"`
}

// EntitlementInfo represents each entitlement in API response
type EntitlementInfo struct {
	CustomerIdentifier string              `json:"CustomerIdentifier"`
	Dimension          string              `json:"Dimension"`
	ExpirationDate     int64               `json:"ExpirationDate"`
	ProductCode        string              `json:"ProductCode"`
	Value              EntitlementValueAWS `json:"Value"`
}

// EntitlementResponse represents the API response
type EntitlementResponse = GetEntitlementsResponse

type GetEntitlementsRequest struct {
	CustomerIdentifier string  `json:"customerIdentifier"`
	ProductCode        string  `json:"productCode"`
	MaxResults         *int64  `json:"maxResults,omitempty"`
	NextToken          *string `json:"nextToken,omitempty"`
}

// EntitlementValue represents the value of an entitlement.
type EntitlementValue struct {
	BooleanValue *bool    `json:"booleanValue,omitempty"`
	DoubleValue  *float64 `json:"doubleValue,omitempty"`
	IntegerValue *int64   `json:"integerValue,omitempty"`
	StringValue  *string  `json:"stringValue,omitempty"`
}

// Entitlement represents a single entitlement response
type Entitlement struct {
	CustomerIdentifier string           `json:"customerIdentifier"`
	Dimension          string           `json:"dimension"`
	ExpirationDate     *int64           `json:"expirationDate,omitempty"`
	ProductCode        string           `json:"productCode"`
	Value              EntitlementValue `json:"value"`
}

// GetEntitlementsResponse represents the response from the GetEntitlements API.
type GetEntitlementsResponse struct {
	Entitlements []Entitlement `json:"entitlements"`
	NextToken    *string       `json:"nextToken,omitempty"`
}

type CustomerAdditionalInfo struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	JobRole string `json:"job_role"`
	Company string `json:"company"`
	Country string `json:"country"`
}

// CustomerDetailsResponse represents the response structure
type CustomerDetailsResponse struct {
	Message string `json:"message"`
	Status  bool   `json:"status"`
}

// CustomerDetailsRequest represents the expected request payload
type CustomerDetailsRequest struct {
	CustomerIdentifier string `json:"customer_identifier" binding:"required"`
	Name               string `json:"name" binding:"required"`
	Email              string `json:"email" binding:"required,email"`
	Phone              string `json:"phone" binding:"required"`
	JobRole            string `json:"job_role" binding:"required"`
	Company            string `json:"company" binding:"required"`
	Country            string `json:"country" binding:"required"`
}

// Repository interface defines the contract for database operations
type Repository interface {
	UpdateCustomerBasicInfo(ctx context.Context, info *marketplacemetering.ResolveCustomerOutput) error
	UpdateEntitlements(ctx context.Context, response EntitlementResponse) error
	UpdateCustomerAdditionalInfo(ctx context.Context, customerID string, info CustomerAdditionalInfo) error
	CheckCustomerRegistration(ctx context.Context, customerIdentifier string) (*CustomerRegistrationStatus, error)
}

// repository implements the Repository interface
type repository struct {
	db *gorm.DB
}

// NewRepository creates a new instance of the repository
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// UpdateCustomerBasicInfo updates or creates basic customer information
func (r *repository) UpdateCustomerBasicInfo(ctx context.Context, info *marketplacemetering.ResolveCustomerOutput) error {
	if info == nil {
		return errors.New("invalid input: info is nil")
	}

	// Validate required fields
	if info.CustomerIdentifier == nil || info.CustomerAWSAccountId == nil || info.ProductCode == nil {
		return errors.New("invalid input: missing required fields")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Upsert customer
		if err := tx.Exec(`
			INSERT INTO customers (customer_identifier, aws_account_id)
			VALUES (?, ?)
			ON DUPLICATE KEY UPDATE
				aws_account_id = VALUES(aws_account_id)
		`, *info.CustomerIdentifier, *info.CustomerAWSAccountId).Error; err != nil {
			return err
		}

		// Upsert product
		if err := tx.Exec(`
			INSERT INTO products (product_code)
			VALUES (?)
			ON DUPLICATE KEY UPDATE
				product_code = VALUES(product_code)
		`, *info.ProductCode).Error; err != nil {
			return err
		}

		return nil
	})
}

// You might also want to add a helper function to safely access the values
func getResolveCustomerValues(info *marketplacemetering.ResolveCustomerOutput) (customerID, awsAccountID, productCode string, err error) {
	if info == nil {
		return "", "", "", errors.New("invalid input: info is nil")
	}

	if info.CustomerIdentifier == nil {
		return "", "", "", errors.New("missing CustomerIdentifier")
	}
	if info.CustomerAWSAccountId == nil {
		return "", "", "", errors.New("missing CustomerAWSAccountId")
	}
	if info.ProductCode == nil {
		return "", "", "", errors.New("missing ProductCode")
	}

	return *info.CustomerIdentifier, *info.CustomerAWSAccountId, *info.ProductCode, nil
}

// determineValueType determines the type of value and returns appropriate EntitlementValue
func determineValueType(value EntitlementValue) (*models.EntitlementValue, error) {
	ev := &models.EntitlementValue{}

	if value.BooleanValue != nil {
		ev.ValueType = models.ValueTypeBoolean
		ev.BooleanValue = value.BooleanValue
		return ev, nil
	}
	if value.DoubleValue != nil {
		ev.ValueType = models.ValueTypeDouble
		ev.DoubleValue = value.DoubleValue
		return ev, nil
	}
	if value.IntegerValue != nil {
		ev.ValueType = models.ValueTypeInteger
		ev.IntegerValue = value.IntegerValue
		return ev, nil
	}
	if value.StringValue != nil {
		ev.ValueType = models.ValueTypeString
		ev.StringValue = value.StringValue
		return ev, nil
	}

	return nil, ErrInvalidValue
}

// UpdateEntitlements updates or creates entitlements based on API response
func compareEntitlementValues(existing *models.EntitlementValue, new *models.EntitlementValue) bool {
	if existing.ValueType != new.ValueType {
		return false
	}

	switch existing.ValueType {
	case models.ValueTypeBoolean:
		return reflect.DeepEqual(existing.BooleanValue, new.BooleanValue)
	case models.ValueTypeDouble:
		return reflect.DeepEqual(existing.DoubleValue, new.DoubleValue)
	case models.ValueTypeInteger:
		return reflect.DeepEqual(existing.IntegerValue, new.IntegerValue)
	case models.ValueTypeString:
		return reflect.DeepEqual(existing.StringValue, new.StringValue)
	default:
		return false
	}
}

// UpdateEntitlements updates or creates entitlements based on API response
func (r *repository) UpdateEntitlements(ctx context.Context, response EntitlementResponse) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, ent := range response.Entitlements {
			// First check if entitlement exists with the same customer_identifier, product_code, and dimension
			var existing models.Entitlement
			result := tx.Preload("Value").Where(
				"customer_identifier = ? AND product_code = ? AND dimension = ?",
				ent.CustomerIdentifier,
				ent.ProductCode,
				ent.Dimension,
			).Order("created_at DESC").First(&existing)

			// Create new entitlement value for comparison
			newEntValue, err := determineValueType(ent.Value)
			if err != nil {
				return err
			}

			// If no existing entitlement found, create new one
			if result.Error == gorm.ErrRecordNotFound {
				// Create value
				if err := tx.Create(newEntValue).Error; err != nil {
					return err
				}

				// Convert Unix timestamp to string
				expirationDate := time.Unix(*ent.ExpirationDate, 0).Format(time.RFC3339)

				// Create new entitlement
				newEntitlement := models.Entitlement{
					CustomerIdentifier: ent.CustomerIdentifier,
					ProductCode:        ent.ProductCode,
					Dimension:          ent.Dimension,
					ExpirationDate:     expirationDate,
					ValueID:            newEntValue.ValueID,
				}

				if err := tx.Create(&newEntitlement).Error; err != nil {
					return err
				}

				continue
			} else if result.Error != nil {
				return result.Error
			}

			// Compare values only if entitlement exists
			if existing.Value != nil {
				// If values are the same, skip this entitlement
				if compareEntitlementValues(existing.Value, newEntValue) {
					continue
				}

				// Values are different, create new entitlement record
				if err := tx.Create(newEntValue).Error; err != nil {
					return err
				}

				// Convert Unix timestamp to string
				expirationDate := time.Unix(*ent.ExpirationDate, 0).Format(time.RFC3339)

				// Create new entitlement with new value
				newEntitlement := models.Entitlement{
					CustomerIdentifier: ent.CustomerIdentifier,
					ProductCode:        ent.ProductCode,
					Dimension:          ent.Dimension,
					ExpirationDate:     expirationDate,
					ValueID:            newEntValue.ValueID,
				}

				if err := tx.Create(&newEntitlement).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// UpdateCustomerAdditionalInfo updates additional customer information
func (r *repository) UpdateCustomerAdditionalInfo(ctx context.Context, customerID string, info CustomerAdditionalInfo) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE customers 
		SET 
			name = ?,
			email = ?,
			phone = ?,
			job_role = ?,
			company = ?,
			country = ?
		WHERE customer_identifier = ?
	`, info.Name, info.Email, info.Phone, info.JobRole,
		info.Company, info.Country, customerID)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrCustomerNotFound
	}

	return nil
}

// Additional helper functions for common queries

// GetCustomerByID retrieves a customer by their identifier
func (r *repository) GetCustomerByID(ctx context.Context, customerID string) (*models.Customer, error) {
	var customer models.Customer
	if err := r.db.WithContext(ctx).First(&customer, "customer_identifier = ?", customerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}
	return &customer, nil
}

// GetEntitlementsByCustomerID retrieves all entitlements for a customer
func (r *repository) GetEntitlementsByCustomerID(ctx context.Context, customerID string) ([]models.Entitlement, error) {
	var entitlements []models.Entitlement
	if err := r.db.WithContext(ctx).
		Preload("Value").
		Preload("Product").
		Where("customer_identifier = ?", customerID).
		Find(&entitlements).Error; err != nil {
		return nil, err
	}
	return entitlements, nil
}

type customerRegistrationCheck struct {
	CustomerIdentifier string
	Name               *string
	Email              *string
	Phone              *string
	JobRole            *string
	Company            *string
	Country            *string
	ProductName        *string
}

func (r *repository) CheckCustomerRegistration(ctx context.Context, customerIdentifier string) (*CustomerRegistrationStatus, error) {
	var result customerRegistrationCheck

	// Query customer data and latest product
	err := r.db.WithContext(ctx).
		Table("customers").
		Select(`
            customers.customer_identifier,
            customers.name,
            customers.email,
            customers.phone,
            customers.job_role,
            customers.company,
            customers.country,
            products.product_name
        `).
		Joins("LEFT JOIN entitlements ON customers.customer_identifier = entitlements.customer_identifier").
		Joins("LEFT JOIN products ON entitlements.product_code = products.product_code").
		Where("customers.customer_identifier = ?", customerIdentifier).
		Order("entitlements.created_at DESC").
		Limit(1).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	// If no customer found, return false and empty product name
	if result.CustomerIdentifier == "" {
		return &CustomerRegistrationStatus{
			NeedsRegistration: false,
			ProductName:       "",
		}, nil
	}

	// Check if any required fields are empty
	needsRegistration := result.Name == nil ||
		result.Email == nil ||
		result.Phone == nil ||
		result.JobRole == nil ||
		result.Company == nil ||
		result.Country == nil

	productName := ""
	if result.ProductName != nil {
		productName = *result.ProductName
	}

	return &CustomerRegistrationStatus{
		NeedsRegistration: needsRegistration,
		ProductName:       productName,
	}, nil
}

// CustomerRegistrationStatus represents the response for customer registration check
type CustomerRegistrationStatus struct {
	NeedsRegistration bool   `json:"needs_registration"`
	ProductName       string `json:"product_name,omitempty"`
}
