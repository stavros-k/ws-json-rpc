package typesystem

import (
	"fmt"
	"log/slog"
	"maps"
	"sync"
)

// TypeRegistry manages type nodes for the new type system.
// It provides thread-safe registration and retrieval of types.
type TypeRegistry struct {
	l     *slog.Logger
	mu    sync.RWMutex
	types map[string]TypeNode
}

// NewTypeRegistry creates a new type registry.
func NewTypeRegistry(l *slog.Logger) *TypeRegistry {
	return &TypeRegistry{
		l:     l,
		types: make(map[string]TypeNode),
	}
}

// Register registers a type node with the given name.
// Returns an error if a type with the same name already exists.
func (r *TypeRegistry) Register(name string, node TypeNode) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.types[name]; exists {
		r.l.Error("duplicate type registration attempted", slog.String("name", name))
		return fmt.Errorf("type %s is already registered", name)
	}

	r.types[name] = node
	r.l.Debug("registered type", slog.String("name", name), slog.String("kind", string(node.GetKind())))
	return nil
}

// Get retrieves a type node by name.
// Returns nil if the type doesn't exist.
func (r *TypeRegistry) Get(name string) TypeNode {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.types[name]
}

// GetAll returns all registered types.
func (r *TypeRegistry) GetAll() map[string]TypeNode {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]TypeNode, len(r.types))
	maps.Copy(result, r.types)
	return result
}

// Exists checks if a type with the given name is registered.
func (r *TypeRegistry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.types[name]
	return exists
}
