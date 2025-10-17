package cmd

import (
	"fmt"

	"github.com/zondax/golem/pkg/logger"
	"go.uber.org/zap"
)

func initLogger() *zap.Logger {
	logger, err := zap.NewDevelopment(zap.AddStacktrace(zap.PanicLevel))
	if err != nil {
		panic(fmt.Errorf("failed to create logger: %v", err))
	}
	return logger
}

func getParserLogger() *logger.Logger {
	return logger.NewLogger(logger.Config{Level: "panic"})
}
