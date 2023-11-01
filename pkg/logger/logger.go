package logger

import (
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Log is global logger
	log *zap.Logger

	// timeFormat is custom Time format
	customTimeFormat string

	// onceInit guarantee initialize logger only once
	onceInit sync.Once
)

// customTimeEncoder encode Time to our custom format
// This example how we can customize zap default functionality
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(customTimeFormat))
}

// Init initializes log by input parameters
// lvl - global log level: Debug(-1), Info(0), Warn(1), Error(2), DPanic(3), Panic(4), Fatal(5)
// timeFormat - custom time format for logger of empty string to use default
func Init(lvl string, timeFormat string) (*zap.Logger, error) {
	var err error

	onceInit.Do(func() {
		// First, define our level-handling logic.
		if lvl == "" {
			lvl = "info"
		}
		globalLevel, pErr := zapcore.ParseLevel(lvl)
		if err != nil {
			err = pErr
		}

		// High-priority output should also go to standard error, and low-priority
		// output should also go to standard out.
		// It is useful for Kubernetes deployment.
		// Kubernetes interprets os.Stdout log items as INFO and os.Stderr log items
		// as ERROR by default.
		highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel
		})
		lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= globalLevel && lvl < zapcore.ErrorLevel
		})
		consoleInfos := zapcore.Lock(os.Stdout)
		consoleErrors := zapcore.Lock(os.Stderr)

		// Configure console output.
		var useCustomTimeFormat bool
		ecfg := zap.NewProductionEncoderConfig()
		ecfg.MessageKey = "message" // this is so we have consistent key names with node.js winston logger
		if len(timeFormat) > 0 {
			customTimeFormat = timeFormat
			ecfg.TimeKey = "timestamp"
			ecfg.EncodeTime = customTimeEncoder
			useCustomTimeFormat = true
		}
		consoleEncoder := zapcore.NewJSONEncoder(ecfg)

		// Join the outputs, encoders, and level-handling functions into
		// zapcore.
		core := zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
			zapcore.NewCore(consoleEncoder, consoleInfos, lowPriority),
		)

		// From a zapcore.Core, it's easy to construct a Logger.
		log = zap.New(core)
		zap.ReplaceGlobals(log)
		zap.RedirectStdLog(log)

		if !useCustomTimeFormat {
			log.Warn("time format for logger is not provided - using zap default")
		}
	})

	return log, err
}

func Flush() error {
	return log.Sync()
}
