package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"go.uber.org/fx"
)

// =====================================================
// DEMO: Map Value Groups + Decoration in Fx
// =====================================================

// Logger interface for different log outputs
type Logger interface {
	Log(message string)
	Name() string
}

// FileLogger logs to files
type FileLogger struct{ filename string }

func (l *FileLogger) Log(message string) {
	fmt.Printf("📁 FileLogger[%s]: %s\n", l.filename, message)
}

func (l *FileLogger) Name() string {
	return "file"
}

// ConsoleLogger logs to console
type ConsoleLogger struct{}

func (l *ConsoleLogger) Log(message string) {
	fmt.Printf("🖥️  ConsoleLogger: %s\n", message)
}

func (l *ConsoleLogger) Name() string {
	return "console"
}

// DatabaseLogger logs to database
type DatabaseLogger struct{}

func (l *DatabaseLogger) Log(message string) {
	fmt.Printf("🗄️  DatabaseLogger: %s\n", message)
}

func (l *DatabaseLogger) Name() string {
	return "database"
}

// =====================================================
// PROVIDERS
// =====================================================

func ProvideFileLogger() Logger {
	return &FileLogger{filename: "app.log"}
}

func ProvideConsoleLogger() Logger {
	return &ConsoleLogger{}
}

func ProvideDatabaseLogger() Logger {
	return &DatabaseLogger{}
}

// =====================================================
// MAP DECORATION EXAMPLE
// =====================================================

type LoggerDecorationParams struct {
	fx.In
	// Input: map of loggers by name
	Loggers map[string]Logger `group:"loggers"`
}

type LoggerDecorationResult struct {
	fx.Out
	// Output: enhanced map of loggers
	Loggers map[string]Logger `group:"loggers"`
}

// DecorateLoggers demonstrates map decoration - add prefix to all loggers
func DecorateLoggers(params LoggerDecorationParams) LoggerDecorationResult {
	fmt.Println("\n🎨 Decorating loggers with [ENHANCED] prefix...")

	enhancedLoggers := make(map[string]Logger)

	for name, logger := range params.Loggers {
		// Wrap each logger with enhancement
		enhancedLoggers[name] = &EnhancedLogger{
			wrapped: logger,
			prefix:  "[ENHANCED]",
		}
		fmt.Printf("   ✨ Enhanced logger: %s\n", name)
	}

	return LoggerDecorationResult{Loggers: enhancedLoggers}
}

// EnhancedLogger wraps another logger with a prefix
type EnhancedLogger struct {
	wrapped Logger
	prefix  string
}

func (l *EnhancedLogger) Log(message string) {
	l.wrapped.Log(fmt.Sprintf("%s %s", l.prefix, message))
}

func (l *EnhancedLogger) Name() string {
	return l.wrapped.Name()
}

// =====================================================
// LOGGING SERVICE USING DECORATED MAP
// =====================================================

type LoggingServiceParams struct {
	fx.In

	// 🎯 This map will contain DECORATED loggers!
	LoggerMap map[string]Logger `group:"loggers"`
}

type LoggingService struct {
	loggers map[string]Logger
}

func NewLoggingService(params LoggingServiceParams) *LoggingService {
	fmt.Printf("\n🔧 LoggingService created with %d decorated loggers\n", len(params.LoggerMap))

	// Show what we received
	for name := range params.LoggerMap {
		fmt.Printf("   📄 Logger available: %s\n", name)
	}

	return &LoggingService{loggers: params.LoggerMap}
}

func (s *LoggingService) LogToSpecific(loggerName, message string) error {
	logger, exists := s.loggers[loggerName]
	if !exists {
		return fmt.Errorf("logger %q not found", loggerName)
	}

	logger.Log(message)
	return nil
}

func (s *LoggingService) LogToAll(message string) {
	fmt.Println("\n📢 Broadcasting to all loggers:")
	for _, logger := range s.loggers {
		logger.Log(message)
	}
}

func (s *LoggingService) LogToFiltered(message string, filter func(name string) bool) {
	fmt.Printf("\n🔍 Logging to filtered loggers (%s):\n", "contains 'log'")
	for name, logger := range s.loggers {
		if filter(name) {
			logger.Log(message)
		}
	}
}

// =====================================================
// APPLICATION LIFECYCLE
// =====================================================

func RunDecorationDemo(lc fx.Lifecycle, service *LoggingService) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			fmt.Println("\n🚀 Starting Map Decoration Demo")
			fmt.Println("===================================")

			// Test specific logger
			fmt.Println("\n🎯 Testing specific logger (decorated):")
			service.LogToSpecific("console", "This is a console message")

			// Test all loggers (decorated)
			service.LogToAll("This message goes to all decorated loggers")

			// Test filtered logging (using map keys)
			service.LogToFiltered("Filtered message", func(name string) bool {
				return strings.Contains(name, "log") // This won't match our current names
			})

			// Test another filter
			fmt.Printf("\n🔍 Logging to filtered loggers (%s):\n", "starts with 'c'")
			service.LogToFiltered("Another filtered message", func(name string) bool {
				return strings.HasPrefix(name, "c")
			})

			fmt.Println("\n✅ Map decoration demo completed successfully!")
			fmt.Println("\n💡 Key insights:")
			fmt.Println("   - Map value groups can be decorated just like slices")
			fmt.Println("   - Decorators receive map[string]T and return map[string]T")
			fmt.Println("   - All loggers were enhanced with [ENHANCED] prefix")
			fmt.Println("   - Map keys (names) are preserved through decoration")
			fmt.Println("   - Easy filtering and lookup by name")

			return nil
		},
	})
}

// =====================================================
// MAIN APPLICATION
// =====================================================

func main() {
	app := fx.New(
		// Provide loggers with names and groups
		fx.Provide(
			fx.Annotate(
				ProvideFileLogger,
				fx.ResultTags(`name:"file" group:"loggers"`),
			),
			fx.Annotate(
				ProvideConsoleLogger,
				fx.ResultTags(`name:"console" group:"loggers"`),
			),
			fx.Annotate(
				ProvideDatabaseLogger,
				fx.ResultTags(`name:"database" group:"loggers"`),
			),
		),

		// 🎨 DECORATE the logger map - add enhancements
		fx.Decorate(DecorateLoggers),

		// Provide the logging service (receives decorated loggers)
		fx.Provide(NewLoggingService),

		// Register the demo lifecycle hook
		fx.Invoke(RunDecorationDemo),

		// Suppress logs for cleaner demo output
		fx.NopLogger,
	)

	fmt.Println("🎨 Fx Map Value Groups + Decoration Demo")
	fmt.Println("==========================================")
	fmt.Println("This demo shows map decoration with our new feature!")

	if err := app.Start(context.Background()); err != nil {
		log.Fatal(err)
	}

	if err := app.Stop(context.Background()); err != nil {
		log.Fatal(err)
	}
}