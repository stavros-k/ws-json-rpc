# Object Field Enhancement

The `ObjectField` function has been enhanced to accept `TypeBuilder` instances instead of pre-built `TypeDefinition` objects, making the API more convenient and enabling proper type reuse.

## Before (Old API)
```go
// Had to pre-build nested types
addressType := NewType("Address").
    AddField("street", StringField()).
    AddField("city", StringField())

addressTypeDef, err := addressType.Build() // Manual building required
if err != nil {
    // Handle error
}

userType := NewType("User").
    AddField("address", ObjectField(addressTypeDef)) // Used pre-built type
```

## After (New API)
```go
// No need to pre-build - just pass the TypeBuilder
addressType := NewType("Address").
    AddField("street", StringField()).
    AddField("city", StringField())

userType := NewType("User").
    AddField("address", ObjectField(addressType)) // Pass TypeBuilder directly
```

## Type Reuse

The new API properly handles type reuse without errors:

```go
addressType := NewType("Address").
    AddField("street", StringField()).
    AddField("city", StringField())

userType := NewType("User").
    AddField("homeAddress", ObjectField(addressType).Optional()).
    AddField("workAddress", ObjectField(addressType).Optional()).
    AddField("billingAddress", ObjectField(addressType))

// All three fields correctly reference the same Address type
// No duplicate building or errors
```

## How It Works

1. **Deferred Building**: Field builders are stored and built only when the schema is finalized
2. **Type Caching**: Each TypeBuilder is built only once and cached for reuse
3. **Context-Aware Building**: Nested types are resolved with access to the schema context
4. **Error Propagation**: Building errors from nested types are properly propagated up

## Benefits

- ✅ **Cleaner API** - No manual building of nested types
- ✅ **Type Reuse** - Same TypeBuilder can be used multiple times without errors
- ✅ **Better Error Handling** - All validation happens at schema build time
- ✅ **Consistent Behavior** - Works the same way as other field types
- ✅ **Lazy Building** - Types are only built when actually needed

This enhancement makes the builder DSL more intuitive and prevents common errors when working with nested object structures.