package zaplogger

import (
	"go.k6.io/k6/js/modules"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

// init is called by the Go runtime at application startup.
func init() {
	modules.Register("k6/x/zaplogger", new(RootModule))
}

type RootModule struct{}
type ZapLogger struct {
	vu modules.VU
}

var (
	_ modules.Module   = &RootModule{}
	_ modules.Instance = &ZapLogger{}
)

type DynamicObject map[string]interface{}

func (d DynamicObject) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for k, v := range d {
		switch v := v.(type) {
		case int:
			enc.AddInt(k, v)
		case float64:
			enc.AddFloat64(k, v)
		case string:
			enc.AddString(k, v)
		default:
			enc.AddReflected(k, v)
		}
	}
	return nil
}
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &ZapLogger{vu: vu}
}

func (zaplogger *ZapLogger) Exports() modules.Exports {
	return modules.Exports{Default: zaplogger}
}

// InitLogger initializes the JSON logger to log only to the console
func (z *ZapLogger) InitLogger() *zap.SugaredLogger {
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

// CreateDynamicObject creates a dynamic log object with key-value pairs
func (z *ZapLogger) CreateDynamicObject(args ...interface{}) DynamicObject {
	obj := make(DynamicObject)
	for i := 0; i < len(args); i += 2 {
		key, _ := args[i].(string)
		obj[key] = args[i+1]
	}
	return obj
}

// ZapObject creates a zapcore.Field for structured logging
func (z *ZapLogger) ZapObject(key string, args ...interface{}) zapcore.Field {
	obj := make(DynamicObject)
	for i := 0; i < len(args); i += 2 {
		key, _ := args[i].(string)
		obj[key] = args[i+1]
	}
	return zap.Object(key, obj)
}
