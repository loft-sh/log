package logr

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	zaphook "github.com/Sytten/logrus-zap-hook"
	"github.com/go-logr/logr"
	"github.com/loft-sh/log/logr/zapr"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
)

func NewLogger(component string) (logr.Logger, error) {
	path, _ := os.Getwd()
	path = fmt.Sprintf("%s/", path)

	// -- Variables --
	development := os.Getenv("DEVELOPMENT") // true or false
	loftDebug := os.Getenv("LOFTDEBUG")     // true or false

	loftLogEncoding := os.Getenv("LOFT_LOG_ENCODING") // json or console
	if loftLogEncoding == "" {
		loftLogEncoding = "console"
	}
	if loftLogEncoding != "json" && loftLogEncoding != "console" {
		return logr.Logger{}, fmt.Errorf("invalid log encoding: %s", loftLogEncoding)
	}

	logFullCallerPath := os.Getenv("LOFT_LOG_FULL_CALLER_PATH") // true or false
	if logFullCallerPath == "" {
		logFullCallerPath = "false"
	} else if loftDebug == "true" {
		logFullCallerPath = "true"
	}

	logLevel := os.Getenv("LOFT_LOG_LEVEL") // debug, info, warn, error, dpanic, panic, fatal
	if logLevel == "" {
		logLevel = "warn"
	}

	kubernetesVerbosityLevel := os.Getenv("KUBERNETES_VERBOSITY_LEVEL") // numerical values increasing: 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10
	if kubernetesVerbosityLevel == "" {
		kubernetesVerbosityLevel = "0"
	}
	if kubernetesVerbosityLevel != "0" {
		logLevel = "debug"
	}
	if logLevel == "debug" && kubernetesVerbosityLevel == "0" {
		kubernetesVerbosityLevel = "1"
	}

	// -- Config --
	config := zap.NewProductionConfig()

	if development == "true" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// -- Set log encoding --
	config.Encoding = loftLogEncoding

	// -- Set log caller format --
	if logFullCallerPath == "true" || loftDebug == "true" {
		config.EncoderConfig.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(strings.TrimPrefix(caller.String(), path))
		}
	} else {
		config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	// -- Set log level --
	atomicLevel, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		atomicLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	}
	config.Level = atomicLevel

	// -- Build config --
	zapLog, err := config.Build(zap.Fields(zap.String("component", component)))
	if err != nil {
		return logr.Logger{}, fmt.Errorf("failed to build zap logger: %w", err)
	}

	// Zap global logger
	_ = zap.ReplaceGlobals(zapLog)

	// logr
	log := zapr.NewLogger(zapLog)

	// Klog global logger
	klog.ClearLogger()

	klogFlagSet := &flag.FlagSet{}
	klog.InitFlags(klogFlagSet)
	if err := klogFlagSet.Set("v", kubernetesVerbosityLevel); err != nil {
		return logr.Logger{}, fmt.Errorf("failed to set klog verbosity level: %w", err)
	}
	if err := klogFlagSet.Parse([]string{}); err != nil {
		return logr.Logger{}, fmt.Errorf("failed to parse klog flags: %w", err)
	}

	klog.SetLogger(log)

	// Logrus
	logrus.SetReportCaller(true) // So Zap reports the right caller
	logrus.SetOutput(io.Discard) // Prevent logrus from writing its logs

	hook, err := zaphook.NewZapHook(zapLog)
	if err != nil {
		return logr.Logger{}, fmt.Errorf("failed to create logrus hook: %w", err)
	}
	logrus.AddHook(hook)

	return log, nil
}
