package zaplogger

import (
	"fmt"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/metrics"
	"go.k6.io/k6/output"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"strings"
	"time"
)

// init is called by the Go runtime at application startup.
func init() {
	// Register the JS module for custom logging
	modules.Register("k6/x/zaplogger", new(RootModule))

	// Register the logger as a K6 output type
	output.RegisterExtension("zaplogger", NewZapLogger)
}

type RootModule struct{}
type ZapLogger struct {
	vu      modules.VU         // Virtual User context, used for JS-based export
	logger  *zap.SugaredLogger // Zap logger instance
	out     output.Params      // K6 Output params
	metrics string             // String to hold metric log information for testing
}

var (
	_ modules.Module   = &RootModule{}
	_ modules.Instance = &ZapLogger{}
	_ output.Output    = &ZapLogger{} // To integrate ZapLogger as a K6 output
)

// NewZapLogger creates a new logger instance as a K6 output
func NewZapLogger(params output.Params) (output.Output, error) {
	// Initialize a new ZapLogger instance
	zapLogger := &ZapLogger{
		logger: InitLogger(),
		out:    params,
	}
	return zapLogger, nil
}

// InitLogger creates a console JSON logger (supports stdout)
func InitLogger() *zap.SugaredLogger {
	// Create a console write syncer for stdout
	consoleSyncer := zapcore.AddSync(os.Stdout)

	// JSON Encoder
	encoder := getJSONEncoder()

	// Core for console logging
	consoleCore := zapcore.NewCore(encoder, consoleSyncer, zapcore.DebugLevel)

	// Build the logger
	logger := zap.New(consoleCore)
	return logger.Sugar()
}

// getJSONEncoder returns a JSON encoder for formatting logs
func getJSONEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	// Format timestamp in ISO8601 standard for readability
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

// Start initializes any state needed for the output
func (z *ZapLogger) Start() error {
	z.logger.Info("Zap Logger for K6 metrics started")
	return nil
}

// Stop finalizes any tasks in progress, flushes logs, etc.
func (z *ZapLogger) Stop() error {
	// Flush any remaining logs
	z.logger.Sync()
	z.logger.Info("Zap Logger stopped")
	return nil
}

// AddMetricSamples processes and logs K6 metric samples
func (z *ZapLogger) AddMetricSamples(samples []metrics.SampleContainer) {
	for _, sample := range samples {
		all := sample.GetSamples() // Retrieve all individual samples

		// Convert samples to structured log data
		logData := z.dynamicObjectFromSamples(all)

		// Log metrics with Zap
		z.logger.Infow("Metric Sample",
			"timestamp", all[0].GetTime().Format(time.RFC3339Nano),
			"metricKeyValues", logData,
		)
	}
}

// dynamicObjectFromSamples converts metrics samples into structured JSON log data
func (z *ZapLogger) dynamicObjectFromSamples(samples []metrics.Sample) DynamicObject {
	data := make(DynamicObject)
	for _, sample := range samples {
		data[sample.Metric.Name] = fmt.Sprintf("%v", sample.Value)
		if sample.Time != (time.Time{}) {
			data["time"] = sample.Time.Format(time.RFC3339Nano)
		}
		if len(sample.Tags) > 0 {
			data["tags"] = sample.Tags.String()
		}
	}
	return data
}

// CreateDynamicObject creates a dynamic log object with key-value pairs
func (z *ZapLogger) CreateDynamicObject(args ...interface{}) DynamicObject {
	obj := make(DynamicObject)
	for i := 0; i < len(args); i += 2 {
		key, _ := args[i].(string)
		obj[key] = args[i+1]
	}
	return obj
}
