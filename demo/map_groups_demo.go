package main

import (
	"context"
	"fmt"
	"log"

	"go.uber.org/fx"
)

// =====================================================
// DEMO: Map Value Groups in Fx
// =====================================================

// Handler represents a generic handler interface
type Handler interface {
	Handle(ctx context.Context, message string) error
	Name() string
}

// EmailHandler handles email notifications
type EmailHandler struct{}

func (h *EmailHandler) Handle(ctx context.Context, message string) error {
	fmt.Printf("📧 EmailHandler: Sending email - %s\n", message)
	return nil
}

func (h *EmailHandler) Name() string {
	return "email"
}

// SlackHandler handles Slack notifications
type SlackHandler struct{}

func (h *SlackHandler) Handle(ctx context.Context, message string) error {
	fmt.Printf("💬 SlackHandler: Sending Slack message - %s\n", message)
	return nil
}

func (h *SlackHandler) Name() string {
	return "slack"
}

// SMSHandler handles SMS notifications
type SMSHandler struct{}

func (h *SMSHandler) Handle(ctx context.Context, message string) error {
	fmt.Printf("📱 SMSHandler: Sending SMS - %s\n", message)
	return nil
}

func (h *SMSHandler) Name() string {
	return "sms"
}

// =====================================================
// NOTIFICATION SERVICE USING MAP VALUE GROUPS
// =====================================================

type NotificationService struct {
	// This is the NEW feature - consuming value groups as map[string]T!
	handlerMap map[string]Handler `group:"handlers"`

	// Still works - consuming as slice like before
	handlerSlice []Handler `group:"handlers"`
}

type NotificationParams struct {
	fx.In

	// 🎯 NEW: Map consumption - handlers indexed by name
	HandlerMap map[string]Handler `group:"handlers"`

	// ✅ EXISTING: Slice consumption still works
	HandlerSlice []Handler `group:"handlers"`
}

func NewNotificationService(params NotificationParams) *NotificationService {
	fmt.Println("\n🔧 NotificationService created with:")
	fmt.Printf("   Map handlers: %d entries\n", len(params.HandlerMap))
	fmt.Printf("   Slice handlers: %d entries\n", len(params.HandlerSlice))

	// Show map contents
	fmt.Println("   📋 Handler map contents:")
	for name, handler := range params.HandlerMap {
		fmt.Printf("     - %s: %T\n", name, handler)
	}

	return &NotificationService{
		handlerMap:   params.HandlerMap,
		handlerSlice: params.HandlerSlice,
	}
}

func (s *NotificationService) SendToSpecificHandler(ctx context.Context, handlerName, message string) error {
	// 🎯 NEW CAPABILITY: Direct lookup by name using map!
	handler, exists := s.handlerMap[handlerName]
	if !exists {
		return fmt.Errorf("handler %q not found", handlerName)
	}

	fmt.Printf("📤 Sending via specific handler '%s':\n", handlerName)
	return handler.Handle(ctx, message)
}

func (s *NotificationService) BroadcastToAll(ctx context.Context, message string) error {
	fmt.Println("📢 Broadcasting to all handlers:")
	for _, handler := range s.handlerSlice {
		if err := handler.Handle(ctx, message); err != nil {
			return err
		}
	}
	return nil
}

func (s *NotificationService) ListAvailableHandlers() []string {
	// 🎯 NEW: Easy to get all handler names from map keys
	names := make([]string, 0, len(s.handlerMap))
	for name := range s.handlerMap {
		names = append(names, name)
	}
	return names
}

// =====================================================
// PROVIDER FUNCTIONS
// =====================================================

// Provide handlers using BOTH dig.Name() AND dig.Group()
// This was impossible before this PR!

func ProvideEmailHandler() Handler {
	return &EmailHandler{}
}

func ProvideSlackHandler() Handler {
	return &SlackHandler{}
}

func ProvideSMSHandler() Handler {
	return &SMSHandler{}
}

// =====================================================
// APPLICATION LIFECYCLE HOOKS
// =====================================================

func RunDemo(lc fx.Lifecycle, service *NotificationService) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			fmt.Println("\n🚀 Starting Map Value Groups Demo")
			fmt.Println("=====================================")

			// Show available handlers
			fmt.Println("\n📋 Available handlers:")
			for _, name := range service.ListAvailableHandlers() {
				fmt.Printf("   - %s\n", name)
			}

			// Test specific handler lookup (NEW CAPABILITY)
			fmt.Println("\n🎯 Testing specific handler lookup:")
			if err := service.SendToSpecificHandler(ctx, "email", "Welcome to Fx Map Groups!"); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}

			if err := service.SendToSpecificHandler(ctx, "slack", "Fx now supports map[string]T groups!"); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}

			// Test non-existent handler
			fmt.Println("\n🧪 Testing non-existent handler:")
			if err := service.SendToSpecificHandler(ctx, "telegram", "This should fail"); err != nil {
				fmt.Printf("✅ Expected error: %v\n", err)
			}

			// Test broadcast (existing capability)
			fmt.Println("\n📢 Testing broadcast to all handlers:")
			if err := service.BroadcastToAll(ctx, "System maintenance in 10 minutes"); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}

			fmt.Println("\n✅ Demo completed successfully!")
			return nil
		},
	})
}

// =====================================================
// MAIN APPLICATION
// =====================================================

func main() {
	app := fx.New(
		// Provide handlers with BOTH name AND group
		fx.Provide(
			fx.Annotate(
				ProvideEmailHandler,
				fx.ResultTags(`name:"email" group:"handlers"`),
			),
			fx.Annotate(
				ProvideSlackHandler,
				fx.ResultTags(`name:"slack" group:"handlers"`),
			),
			fx.Annotate(
				ProvideSMSHandler,
				fx.ResultTags(`name:"sms" group:"handlers"`),
			),
		),

		// Provide the notification service
		fx.Provide(NewNotificationService),

		// Register the demo lifecycle hook
		fx.Invoke(RunDemo),

		// Suppress logs for cleaner demo output
		fx.NopLogger,
	)

	fmt.Println("🔥 Fx Map Value Groups Proof of Concept")
	fmt.Println("========================================")
	fmt.Println("This demo shows the new map[string]T value group feature!")

	if err := app.Start(context.Background()); err != nil {
		log.Fatal(err)
	}

	if err := app.Stop(context.Background()); err != nil {
		log.Fatal(err)
	}
}