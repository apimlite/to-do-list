package service

import (
	"aws-markertplace-integration/db/repo"
	"context"

	"github.com/aws/aws-sdk-go-v2/service/marketplaceentitlementservice"
	"github.com/aws/aws-sdk-go-v2/service/marketplacemetering"
)

// EntitlementClientInterface defines the methods for interacting with the AWS Marketplace Entitlement Service.
type EntitlementClientInterface interface {
	GetEntitlements(ctx context.Context, params *marketplaceentitlementservice.GetEntitlementsInput, optFns ...func(*marketplaceentitlementservice.Options)) (*marketplaceentitlementservice.GetEntitlementsOutput, error)
}

// ResolveCustomerRequest represents a request to resolve a customer in AWS Marketplace.
type ResolveCustomerRequest struct {
	AwsMarketplaceToken string `json:"awsMarketplaceToken"`
}

// MeteringClientInterface defines the methods for interacting with the AWS Marketplace Metering Service.
type MeteringClientInterface interface {
	BatchMeterUsage(ctx context.Context, params *marketplacemetering.BatchMeterUsageInput, optFns ...func(*marketplacemetering.Options)) (*marketplacemetering.BatchMeterUsageOutput, error)
	ResolveCustomer(ctx context.Context, params *marketplacemetering.ResolveCustomerInput, optFns ...func(*marketplacemetering.Options)) (*marketplacemetering.ResolveCustomerOutput, error)
}

// GetEntitlementsRequest represents a request to get entitlements for a customer.
type GetEntitlementsRequest struct {
	CustomerIdentifier string  `json:"customerIdentifier"`
	ProductCode        string  `json:"productCode"`
	MaxResults         *int32  `json:"maxResults,omitempty"`
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

type CustomerAdditionalInfo = repo.CustomerAdditionalInfo

// CustomerDetailsResponse represents the response structure
type CustomerDetailsResponse struct {
	Message string `json:"message"`
	Status  bool   `json:"status"`
}

// CustomerDetailsRequest represents the expected request payload
type CustomerDetailsRequest struct {
	CustomerIdentifier string `form:"customer_identifier" binding:"required"`
	Name               string `form:"name" binding:"required"`
	Email              string `form:"email" binding:"required,email"`
	Phone              string `form:"phone" binding:"required"`
	JobRole            string `form:"job_role" binding:"required"`
	Company            string `form:"company" binding:"required"`
	Country            string `form:"country" binding:"required"`
}
