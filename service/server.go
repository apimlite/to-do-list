
package service

import (
	"aws-markertplace-integration/db/repo"
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/marketplaceentitlementservice"
	"github.com/aws/aws-sdk-go-v2/service/marketplacemetering"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Service struct {
	port              int
	logger            *zap.SugaredLogger
	MeteringClient    MeteringClientInterface
	EntitlementClient EntitlementClientInterface
	handler           http.Handler
	repo              repo.Repository
}

func New(conf aws.Config, port int, logger zap.SugaredLogger, repo repo.Repository) *Service {
	return &Service{
		port:              port,
		logger:            logger.Named("service"),
		MeteringClient:    marketplacemetering.NewFromConfig(conf),
		EntitlementClient: marketplaceentitlementservice.NewFromConfig(conf),
		repo:              repo,
	}
}

func (s *Service) SetupRouter() {
	router := gin.New()
	router.LoadHTMLGlob("resources/static/*")
	// Error middleware
	router.Use(func(c *gin.Context) {
		c.Next()
	})

	router.POST("/aws-marketplace/webhook", s.handleMarketplaceToken)
	router.POST("/aws-marketplace/onboarding/:customerIdentifier", s.handleCustomerDetails)
	router.GET("/aws-marketplace/onboarding/:customerIdentifier", s.handlerForm)
	router.GET("/health", handleHealthCheck)
	s.handler = router
}

func handleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func (s *Service) Run(ctx context.Context) {
	address := fmt.Sprintf(":%d", s.port)
	srv := http.Server{
		Addr:    address,
		Handler: s.handler,
	}

	go func() {
		s.logger.Infof("Started server in %s", address)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			s.logger.Errorf("ListenAndServe error: %v", err)
		}
	}()
	<-ctx.Done()
	s.logger.Info("Shutdown signal received")
	if err := srv.Shutdown(context.Background()); err != nil {
		s.logger.Errorf("Error during shutdown: %v", err)
	}
	s.logger.Info("Server stopped gracefully")
}
