package zaplogger

import (
	"fmt"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/metrics"
	"go.k6.io/k6/output"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

// DynamicObject is used for key-value pairs in JSON-like logs.
type DynamicObject map[string]interface{}

// RootModule is the entry point for the JS module.
type RootModule struct{}

// ZapLogger is the logger implementation.
type ZapLogger struct {
	vu      modules.VU         // Virtual User context
	logger  *zap.SugaredLogger // Zap Logger instance
	out     output.Params      // K6 Output params
	metrics string             // String for metric information
}

var (
	_ modules.Module   = &RootModule{}
	_ modules.Instance = &ZapLogger{}
	_ output.Output    = &ZapLogger{}
)

// Register module and output
func init() {
	modules.Register("k6/x/zaplogger", new(RootModule))
	output.RegisterExtension("zaplogger", NewZapLogger)
}

// Method for modules.Module implementation
func (r *RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &ZapLogger{
		vu:     vu,
		logger: InitLogger(),
	}
}

// Method for modules.Instance implementation
func (z *ZapLogger) Exports() modules.Exports {
	return modules.Exports{
		Default: z,
		Named: map[string]interface{}{
			"initLogger": InitLogger,
		},
	}
}

// Method for output.Output implementation
func (z *ZapLogger) Description() string {
	return "ZapLogger: A custom logger using Uber Zap for K6."
}

// NewZapLogger initializes the logger output.
func NewZapLogger(params output.Params) (output.Output, error) {
	zapLogger := &ZapLogger{
		logger: InitLogger(),
		out:    params,
	}
	return zapLogger, nil
}

// InitLogger sets up the Zap logger with JSON encoding.
func InitLogger() *zap.SugaredLogger {
	consoleSyncer := zapcore.AddSync(os.Stdout)
	encoder := getJSONEncoder()
	consoleCore := zapcore.NewCore(encoder, consoleSyncer, zapcore.DebugLevel)
	logger := zap.New(consoleCore)
	return logger.Sugar()
}

// getJSONEncoder returns a JSON encoder for the logs.
func getJSONEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

// Start initializes the logger output.
func (z *ZapLogger) Start() error {
	z.logger.Info("Zap Logger for K6 metrics started")
	return nil
}

// Stop finalizes the output.
func (z *ZapLogger) Stop() error {
	z.logger.Sync()
	z.logger.Info("Zap Logger stopped")
	return nil
}

// AddMetricSamples logs metric samples.
func (z *ZapLogger) AddMetricSamples(samples []metrics.SampleContainer) {
	for _, sample := range samples {
		all := sample.GetSamples()
		logData := z.dynamicObjectFromSamples(all)
		z.logger.Infow("Metric Sample",
			"timestamp", all[0].GetTime().Format(time.RFC3339Nano),
			"metricKeyValues", logData,
		)
	}
}

func (z *ZapLogger) dynamicObjectFromSamples(samples []metrics.Sample) DynamicObject {
	data := make(DynamicObject)
	for _, sample := range samples {
		data[sample.Metric.Name] = fmt.Sprintf("%v", sample.Value) // Store sample metric value

		// Include time if present
		if sample.Time != (time.Time{}) {
			data["time"] = sample.Time.Format(time.RFC3339Nano)
		}

		// Process tags using Map()
		if sample.Tags != nil {
			data["tags"] = sample.Tags.Map() // Convert tags to a map
		}
	}
	return data
}
