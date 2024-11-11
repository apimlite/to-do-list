package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"aws-markertplace-integration/db/repo"
	"aws-markertplace-integration/logging"
	"aws-markertplace-integration/service"

	"github.com/aws/aws-sdk-go-v2/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	logger := logging.NewLogger("aws-markertplace-integration")
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		logger.Fatalf("DB_DSN environment variable not set")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	repo := repo.NewRepository(db)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	conf, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(os.Getenv("AWS_DEFAULT_REGION")))
	if err != nil {
		logger.Errorf("Failed to initialize AWS client: %v", err)
	}
	s := service.New(conf, 8080, *logger, repo)
	s.SetupRouter()
	s.Run(ctx)
}
