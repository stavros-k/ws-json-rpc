package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"ws-json-rpc/pkg/wshub"
)

// Handlers - completely independent from Hub
type Handlers struct {
	hub *wshub.Hub
	// Add your dependencies here:
	// db      *sql.DB
	// service *SomeService
}

// Example handler methods
type EchoParams struct {
	Message string `json:"message"`
}

type EchoResult struct {
	Echo string `json:"echo"`
}

func (h *Handlers) Echo(ctx context.Context, params EchoParams) (EchoResult, error) {
	return EchoResult{Echo: params.Message}, nil
}

type AddParams struct {
	A int `json:"a"`
	B int `json:"b"`
}

type AddResult struct {
	Result int `json:"result"`
}

func (h *Handlers) Add(ctx context.Context, params AddParams) (AddResult, error) {
	return AddResult{Result: params.A + params.B}, nil
}

type DoubleParams struct {
	Value int `json:"value"`
}

func (h *Handlers) Double(ctx context.Context, params DoubleParams) (AddResult, error) {
	// Direct method call
	return h.Add(ctx, AddParams{A: params.Value, B: params.Value})
}

type SubscriptionParams struct {
	Event string `json:"event"`
}

func (h *Handlers) Subscribe(ctx context.Context, params SubscriptionParams) (map[string]bool, error) {
	client := ctx.Value("client").(*wshub.Client)
	// Handler that needs hub access
	if err := h.hub.Subscribe(client, EventKind(params.Event)); err != nil {
		return nil, err
	}

	return map[string]bool{"subscribed": true}, nil
}

type UnsubscriptionParams struct {
	Event string `json:"event"`
}

func (h *Handlers) Unsubscribe(ctx context.Context, params UnsubscriptionParams) (map[string]bool, error) {
	client := ctx.Value("client").(*wshub.Client)
	h.hub.Unsubscribe(client, EventKind(params.Event))
	return map[string]bool{"unsubscribed": true}, nil
}

func (h *Handlers) GetUser(ctx context.Context, params struct {
	UserID string `json:"userId"`
}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"id":   params.UserID,
		"name": "John Doe",
	}, nil
}

func MakeComplexHandler(hub *wshub.Hub) wshub.HandlerFunc[struct{ Data string }, map[string]interface{}] {
	return func(ctx context.Context, params struct{ Data string }) (map[string]interface{}, error) {
		// Can call other handlers via hub
		result, err := hub.Call(ctx, MethodKindEcho, EchoParams{Message: params.Data})
		if err != nil {
			return nil, err
		}

		// Can publish events
		hub.PublishEvent(wshub.Event{
			Name: EventKindDataProcessed,
			Data: map[string]string{"original": params.Data},
		})

		return map[string]interface{}{
			"processed": true,
			"echo":      result,
		}, nil
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	hub := wshub.New(logger)
	hub.RegisterEvent(EventKindUserUpdate)
	hub.RegisterEvent(EventKindUserLogin)

	handlers := &Handlers{hub: hub}

	// Register handlers
	wshub.RegisterHandler(hub, MethodKindEcho, handlers.Echo)
	wshub.RegisterHandler(hub, MethodKindAdd, handlers.Add)
	wshub.RegisterHandler(hub, MethodKindDouble, handlers.Double)
	wshub.RegisterHandler(hub, MethodKindSubscribe, handlers.Subscribe)

	// Handler that needs hub access
	wshub.RegisterHandler(hub, MethodKindComplex, MakeComplexHandler(hub))

	// Example with separate handler struct for user domain
	wshub.RegisterHandler(hub, MethodKindGetUser, handlers.GetUser)

	// Register inline handler
	wshub.RegisterHandler(hub, MethodKindPing, func(ctx context.Context, params struct{}) (string, error) {
		return "pong", nil
	})

	go hub.Run()
	go func() {
		eventChan := hub.GetEventChannel()
		// Simulate sending events
		eventChan <- wshub.Event{Name: EventKindUserUpdate, Data: map[string]string{"id": "123", "name": "John"}}
	}()

	http.HandleFunc("/ws", hub.ServeWS())
	logger.Info("WebSocket server starting", slog.String("address", ":8080"))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
	}
}
