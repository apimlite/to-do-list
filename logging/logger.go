
package logging

import (
	"os"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(name string) *zap.SugaredLogger {
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		CallerKey:      "C",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), os.Stdout, zap.DebugLevel)
	logger := zap.New(core).WithOptions(
		zap.AddCaller(),
		zap.AddStacktrace(zap.DPanicLevel),
		zap.Development(),
	)
	return logger.Sugar().Named(name)
}
