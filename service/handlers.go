package service

import (
	"aws-markertplace-integration/db/repo"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/marketplaceentitlementservice"
	"github.com/aws/aws-sdk-go-v2/service/marketplacemetering"
	"github.com/aws/smithy-go"
	"github.com/gin-gonic/gin"
)

func (s *Service) getEntitlements(c *gin.Context) (*repo.GetEntitlementsResponse, error) {
	var getEntitlementRequest GetEntitlementsRequest
	if val, exists := c.Get("getEntitlementRequest"); exists {
		getEntitlementRequest = val.(GetEntitlementsRequest)
	} else if err := c.ShouldBindJSON(&getEntitlementRequest); err != nil {
		return nil, fmt.Errorf("invalid request payload: %w", err)
	}
	s.logger.Infow("Processing GetEntitlements request",
		"customerIdentifier", getEntitlementRequest.CustomerIdentifier,
		"productCode", getEntitlementRequest.ProductCode)
	// Build filter map
	filterMap := make(map[string][]string)
	if getEntitlementRequest.CustomerIdentifier != "" {
		filterMap["CUSTOMER_IDENTIFIER"] = []string{getEntitlementRequest.CustomerIdentifier}
	}
	// Create input parameters
	input := &marketplaceentitlementservice.GetEntitlementsInput{
		ProductCode: aws.String(getEntitlementRequest.ProductCode),
		Filter:      filterMap,
	}

	// Add optional parameters if provided
	if getEntitlementRequest.MaxResults != nil {
		input.MaxResults = getEntitlementRequest.MaxResults
	}
	// Add next token if provided
	if getEntitlementRequest.NextToken != nil {
		input.NextToken = getEntitlementRequest.NextToken
	}

	// Call AWS Marketplace Entitlement Service
	result, err := s.EntitlementClient.GetEntitlements(c.Request.Context(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlements: %w", err)
	}

	// Transform AWS response to our response format
	response := &repo.GetEntitlementsResponse{
		NextToken:    result.NextToken,
		Entitlements: make([]repo.Entitlement, 0, len(result.Entitlements)),
	}

	// Process each entitlement
	for _, awsEnt := range result.Entitlements {
		entitlement := repo.Entitlement{
			CustomerIdentifier: *awsEnt.CustomerIdentifier,
			Dimension:          *awsEnt.Dimension,
			ProductCode:        *awsEnt.ProductCode,
		}
		if awsEnt.ExpirationDate != nil {
			unixTime := awsEnt.ExpirationDate.Unix()
			entitlement.ExpirationDate = &unixTime
		}
		// Handle different value types
		var value repo.EntitlementValue
		switch {
		case awsEnt.Value.BooleanValue != nil:
			value.BooleanValue = awsEnt.Value.BooleanValue
		case awsEnt.Value.DoubleValue != nil:
			value.DoubleValue = awsEnt.Value.DoubleValue
		case awsEnt.Value.IntegerValue != nil:
			// Convert to int64 and assign directly
			int64Value := int64(*awsEnt.Value.IntegerValue)
			value.IntegerValue = &int64Value
		case awsEnt.Value.StringValue != nil:
			value.StringValue = awsEnt.Value.StringValue
		default:
			s.logger.Warnw(
				"Unknown entitlement value type",
				"dimension", *awsEnt.Dimension,
				"type", fmt.Sprintf("%T", awsEnt.Value),
			)
		}
		entitlement.Value = value
		response.Entitlements = append(response.Entitlements, entitlement)
	}

	s.logger.Infow("Retrieved entitlements",
		"count", len(response.Entitlements),
		"hasNextToken", response.NextToken != nil)
	return response, nil
}

func (s *Service) handleError(c *gin.Context, err error) {
	var ae smithy.APIError
	if errors.As(err, &ae) {
		s.logger.Errorf("[ERROR] API Error: %s - %s", ae.ErrorCode(), ae.ErrorMessage())
		// Define a map of error codes to HTTP status and error messages
		errorMap := map[string]struct {
			status  int
			message string
		}{
			"InvalidParameterException":          {http.StatusBadRequest, "Invalid parameter in the request"},
			"InvalidProductCodeException":        {http.StatusBadRequest, "The product code is invalid"},
			"InvalidUsageRecordException":        {http.StatusBadRequest, "The usage record is invalid"},
			"InvalidCustomerIdentifierException": {http.StatusBadRequest, "The customer identifier is invalid"},
			"TimestampOutOfBoundsException":      {http.StatusBadRequest, "The timestamp is outside of the allowed range"},
			"ThrottlingException":                {http.StatusTooManyRequests, "Request was throttled, please try again later"},
			"InternalServiceException":           {http.StatusServiceUnavailable, "An internal error occurred"},
			"InvalidTokenException":              {http.StatusBadRequest, "Invalid registration token"},
			"ExpiredTokenException":              {http.StatusBadRequest, "Registration token has expired"},
		}
		// Lookup the error code in the map
		if errInfo, exists := errorMap[ae.ErrorCode()]; exists {
			s.logger.Errorf("[ERROR] %s: %s", errInfo.status, errInfo.message)
		} else {
			s.logger.Errorf("[ERROR] Unexpected API error: %v", ae.ErrorMessage())
		}
		return
	}
	s.logger.Errorf("[ERROR] Unexpected error: %v", err)
}

// handleMarketplaceToken handles POST requests with token in body
func (s *Service) handleMarketplaceToken(c *gin.Context) {

	token := c.PostForm("x-amzn-marketplace-token")

	if token == "" {
		s.logger.Error("No token provided")
		s.handleHTMLResponse(c, "error.tmpl", http.StatusBadRequest, gin.H{"errorTitle": "Invalid Token", "errorMessage": "No token provided."})
		return
	}

	resolvedCustomer, err := s.MeteringClient.ResolveCustomer(c, &marketplacemetering.ResolveCustomerInput{
		RegistrationToken: &token,
	})

	if err != nil {
		s.handleError(c, fmt.Errorf("failed to resolve customer: %w", err))
		s.handleHTMLResponse(c, "error.tmpl", http.StatusInternalServerError, gin.H{"errorTitle": "Resolve Customer Failed", "errorMessage": "Failed to resolve customer."})
		return
	}

	// Get entitlements
	if resolvedCustomer.CustomerIdentifier == nil {
		s.handleError(c, fmt.Errorf("customer identifier is nil"))
		return
	}

	// err = s.repo.UpdateCustomerBasicInfo(c.Request.Context(), resolvedCustomer)

	// if err != nil {
	// 	s.handleError(c, err)
	// 	s.handleHTMLResponse(c, "error.tmpl", http.StatusInternalServerError, gin.H{"errorTitle": "Update Customer Info Failed", "errorMessage": "Failed to update customer info."})
	// 	return
	// }

	getEntitlementReq := GetEntitlementsRequest{
		CustomerIdentifier: *resolvedCustomer.CustomerIdentifier,
		ProductCode:        *resolvedCustomer.ProductCode,
	}
	s.logger.Infow("Getting entitlements",
		"customerIdentifier", getEntitlementReq.CustomerIdentifier,
		"productCode", getEntitlementReq.ProductCode)
	c.Set("getEntitlementRequest", getEntitlementReq)
	entitlements, err := s.getEntitlements(c)

	if err != nil {
		s.handleError(c, fmt.Errorf("failed to get entitlements: %w", err))
		s.handleHTMLResponse(c, "error.tmpl", http.StatusInternalServerError, gin.H{"errorTitle": "Get Entitlements Failed", "errorMessage": "Failed to get entitlements."})
		return
	}

	if len(entitlements.Entitlements) == 0 {
		s.logger.Infow("No entitlements found",
			"customerIdentifier", getEntitlementReq.CustomerIdentifier,
			"productCode", getEntitlementReq.ProductCode)
		s.handleHTMLResponse(c, "error.tmpl", http.StatusNotFound, gin.H{"errorTitle": "No Entitlements Found", "errorMessage": "No entitlements found."})
		return
	}

	// err = s.repo.UpdateEntitlements(c.Request.Context(), *entitlements)

	// if err != nil {
	// 	s.handleError(c, err)
	// 	s.handleHTMLResponse(c, "error.tmpl", http.StatusInternalServerError, gin.H{"errorTitle": "Update Entitlements Failed", "errorMessage": "Failed to update entitlements."})
	// 	return
	// }

	// res, err := s.repo.CheckCustomerRegistration(c.Request.Context(), getEntitlementReq.CustomerIdentifier)

	// if err != nil {
	// 	s.handleError(c, err)
	// 	s.handleHTMLResponse(c, "error.tmpl", http.StatusInternalServerError, gin.H{"errorTitle": "Check Customer Registration Failed", "errorMessage": "Failed to check customer registration"})
	// 	return
	// }

	// if !res.NeedsRegistration {
	// 	s.handleHTMLResponse(c, "success.tmpl", http.StatusOK, gin.H{})
	// 	return
	// }

	basePath := "zvdz/aws-marketplace-integration/v1.0/"
	s.logger.Infow("Redirecting to onboarding", basePath)
	s.logger.Infow("full url", basePath+"onboarding/"+getEntitlementReq.CustomerIdentifier)
	c.Redirect(302, basePath+"onboarding/"+getEntitlementReq.CustomerIdentifier)
}

// Repository errors
var (
	ErrCustomerNotFound = errors.New("customer not found")
)

//respond html with status code

func (s *Service) handleHTMLResponse(
	c *gin.Context,
	templateName string,
	statusCode int,
	messages map[string]any) {
	c.HTML(statusCode, templateName, messages)
}

// handleCustomerDetails processes POST requests to update customer details
func (s *Service) handleCustomerDetails(c *gin.Context) {
	var req CustomerDetailsRequest

	if err := c.ShouldBind(&req); err != nil {
		s.logger.Errorw("Invalid request payload",
			"error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	s.logger.Infow("Processing customer details update",
		"customerIdentifier", req.CustomerIdentifier)

	// s.handleHTMLResponse(c, "success.tmpl", http.StatusOK, gin.H{})
	// rws, err := s.repo.CheckCustomerRegistration(c.Request.Context(), req.CustomerIdentifier)

	// if err != nil {
	// 	s.handleError(c, err)
	// 	return
	// }

	// if !rws.NeedsRegistration {
	// 	s.handleError(c, errors.New("Customer not found or Already registered"))
	// 	return
	// }

	// Convert request to CustomerAdditionalInfo
	customerInfo := CustomerAdditionalInfo{
		Name:    req.Name,
		Email:   req.Email,
		Phone:   req.Phone,
		JobRole: req.JobRole,
		Company: req.Company,
		Country: req.Country,
	}

	// Call repository method to update customer details
	// updateCustomerError := s.repo.UpdateCustomerAdditionalInfo(c.Request.Context(), req.CustomerIdentifier, customerInfo)
	// if updateCustomerError != nil {
	// 	switch err {
	// 	case ErrCustomerNotFound:
	// 		s.logger.Errorw("Customer not found",
	// 			"customerIdentifier", req.CustomerIdentifier)
	// 	default:
	// 		s.logger.Errorw("Failed to update customer details",
	// 			"customerIdentifier", req.CustomerIdentifier,
	// 			"error", err.Error())
	// 	}
	// 	s.handleHTMLResponse(c, "error.tmpl", http.StatusInternalServerError, gin.H{"errorTitle": "Update Customer Info Failed", "errorMessage": "Failed to update customer info."})
	// 	return

	// }

	s.logger.Infow("Customer details updated successfully",
		"customerIdentifier", customerInfo)
	s.handleHTMLResponse(c, "success.tmpl", http.StatusOK, gin.H{})
}

// handlerForm handles GET requests to retrieve customer form
func (s *Service) handlerForm(c *gin.Context) {
	customerIdentifier := c.Param("customerIdentifier")
	s.logger.Infow("Handling form request", "customerIdentifier", customerIdentifier)
	c.HTML(http.StatusOK, "index.tmpl",
		gin.H{
			"productName":        "asdasd",
			"customerIdentifier": customerIdentifier,
		})
	// res, err := s.repo.CheckCustomerRegistration(c.Request.Context(), customerIdentifier)
	// if err != nil {
	// 	s.handleError(c, err)
	// 	return
	// }
	// s.logger.Infow("Customer registration status",
	// 	"customerIdentifier", customerIdentifier,
	// 	"needsRegistration", res.NeedsRegistration)

	// if !res.NeedsRegistration && res.ProductName == "" {
	// 	s.handleHTMLResponse(c, "error.tmpl", http.StatusNotFound, gin.H{"errorTitle": "Customer Not Found", "errorMessage": "Customer not found."})
	// 	return
	// }

	// if !res.NeedsRegistration {
	// 	s.handleHTMLResponse(c, "success.tmpl", http.StatusOK, gin.H{})
	// 	return
	// }

	// c.Header("Content-Type", "text/html")
	// c.HTML(http.StatusOK, "index.tmpl",
	// 	gin.H{
	// 		"productName":        res.ProductName,
	// 		"customerIdentifier": customerIdentifier,
	// 	})
}
