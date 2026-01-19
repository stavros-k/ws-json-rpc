# Type Definition System

This document describes the type system used by the RPC API documentation generator. The generator parses `.type.json` files and generates type definitions for Go, TypeScript, and C#.

## Overview

The type system uses a custom `.type.json` format that allows **multiple type definitions per file**. Each file is validated against `type-schema.json` which defines the structure and constraints.

### Key Features

- ✅ **Multiple types per file** - Define related types together
- ✅ **Five type kinds** - enum, object, alias, map, array
- ✅ **Strong typing** - No `any`/`interface{}` support - all types must be explicit
- ✅ **String formats** - Built-in support for `uuid` and `date-time`
- ✅ **Nullable & Optional** - Fine-grained control over field presence and nullability
- ✅ **Type references** - Reference other types by name (not file path)
- ✅ **Automatic imports** - Import tracking for generated code

---

## Supported Type Kinds

1. **Enums** - String enumeration types with predefined values
2. **Objects** - Structured types with named fields
3. **Aliases** - Type aliases that reference primitives or other types
4. **Maps** - Dictionary types with typed keys and values
5. **Arrays** - List types with typed items

---

## File Format

### Basic Structure

```json
{
  "$schema": "../type-schema.json",
  "TypeName1": {
    "kind": "enum",
    "description": "...",
    ...
  },
  "TypeName2": {
    "kind": "object",
    "description": "...",
    ...
  }
}
```

### Rules

- Type names must be **PascalCase** starting with uppercase letter: `^[A-Z][a-zA-Z0-9]*$`
- Field names must be **camelCase** starting with lowercase letter: `^[a-z][a-zA-Z0-9]*$`
- Each type must have a `kind` field
- All types must have a `description`

---

## 1. Enums

Enums are string types with a fixed set of possible values.

### Type Definition

```json
{
  "PingStatus": {
    "kind": "enum",
    "description": "Status of a ping request",
    "values": [
      {
        "value": "success",
        "description": "Ping succeeded"
      },
      {
        "value": "timeout",
        "description": "Ping timed out"
      },
      {
        "value": "error",
        "description": "Ping failed with error"
      }
    ]
  }
}
```

### Generated Code

**TypeScript:**

```typescript
/** Ping succeeded */
export const PingStatusSuccess = "success";
/** Ping timed out */
export const PingStatusTimeout = "timeout";
/** Ping failed with error */
export const PingStatusError = "error";

/** Status of a ping request */
export type PingStatus = "success" | "timeout" | "error";

/** Array of all valid PingStatus values */
export const PingStatusValues = [
  PingStatusSuccess,
  PingStatusTimeout,
  PingStatusError,
] as const;

/** Type guard to check if a value is a valid PingStatus */
export function isPingStatus(value: unknown): value is PingStatus {
  return PingStatusValues.includes(value as PingStatus);
}
```

**Go:**

```go
package rpcapi

// Status of a ping request
type PingStatus string

const (
    // Ping succeeded
    PingStatusSuccess PingStatus = "success"
    // Ping timed out
    PingStatusTimeout PingStatus = "timeout"
    // Ping failed with error
    PingStatusError PingStatus = "error"
)

// IsValid checks if the enum value is valid
func (e PingStatus) IsValid() bool {
    switch e {
    case PingStatusSuccess, PingStatusTimeout, PingStatusError:
        return true
    default:
        return false
    }
}
```

---

## 2. Objects

Objects are structured types with named fields. Fields can be required/optional and nullable/non-nullable.

### Type Definition

```json
{
  "User": {
    "kind": "object",
    "description": "User account information",
    "fields": [
      {
        "name": "id",
        "description": "Unique user identifier",
        "type": "string",
        "format": "uuid",
        "optional": false,
        "nullable": false
      },
      {
        "name": "email",
        "description": "User email address",
        "type": "string",
        "optional": false,
        "nullable": false
      },
      {
        "name": "name",
        "description": "User display name",
        "type": "string",
        "optional": true,
        "nullable": false
      },
      {
        "name": "age",
        "description": "User age",
        "type": "integer",
        "optional": true,
        "nullable": true
      },
      {
        "name": "status",
        "description": "User status",
        "$ref": "UserStatus",
        "optional": false,
        "nullable": false
      }
    ]
  }
}
```

### Field Types

Fields can be one of:

1. **Primitive with optional format**:

   ```json
   {
     "type": "string",
     "format": "uuid"
   }
   ```

2. **Type reference**:

   ```json
   {
     "$ref": "OtherType"
   }
   ```

3. **Array**:

   ```json
   {
     "items": {
       "type": "string"
     }
   }
   ```

4. **Map**:

   ```json
   {
     "map": {
       "keyType": "string",
       "valueType": {
         "type": "number"
       }
     }
   }
   ```

### Primitive Types

- `string` - UTF-8 string
- `number` - Floating point number (float64 in Go, number in TS, double in C#)
- `integer` - Integer number (int in Go, number in TS, int in C#)
- `boolean` - Boolean value

### String Formats

- `uuid` - UUID (Universally Unique Identifier)
  - Go: `uuid.UUID` (imports `github.com/google/uuid`)
  - TypeScript: `string`
  - C#: `Guid`

- `date-time` - RFC 3339 timestamp
  - Go: `time.Time`
  - TypeScript: `string` (use `new Date()` to parse)
  - C#: `DateTime`

### Optional vs Nullable

- **`optional: false`** (required) - Field must be present in JSON
- **`optional: true`** - Field can be omitted from JSON
- **`nullable: false`** - Field value cannot be null
- **`nullable: true`** - Field value can be null

### Four Combinations

1. **Required + Non-nullable**: Must be present, cannot be null
   - TS: `id: string`
   - Go: `ID string`

2. **Required + Nullable**: Must be present, can be null
   - TS: `userId: string | null`
   - Go: `UserID *string`

3. **Optional + Non-nullable**: Can be omitted, cannot be null if present
   - TS: `name?: string`
   - Go: `Name *string` with `omitzero` tag

4. **Optional + Nullable**: Can be omitted, can be null if present
   - TS: `age?: number | null`
   - Go: `Age *int` with `omitzero` tag

### Generated Code

**TypeScript:**

```typescript
/** User account information */
export type User = {
  /** Unique user identifier */
  id: string;
  /** User email address */
  email: string;
  /** User display name */
  name?: string;
  /** User age */
  age?: number | null;
  /** User status */
  status: UserStatus;
};
```

**Go:**

```go
package rpcapi

import (
    "github.com/google/uuid"
)

// User - User account information
type User struct {
    // Unique user identifier
    ID uuid.UUID `json:"id"`
    // User email address
    Email string `json:"email"`
    // User display name
    Name *string `json:"name,omitzero"`
    // User age
    Age *int `json:"age,omitzero"`
    // User status
    Status UserStatus `json:"status"`
}
```

**C#:**

```csharp
namespace rpcapi
{
    /// <summary>User account information</summary>
    public class User
    {
        /// <summary>Unique user identifier</summary>
        [JsonProperty("id", Required = Required.Always)]
        public Guid Id { get; set; }

        /// <summary>User email address</summary>
        [JsonProperty("email", Required = Required.Always)]
        public string Email { get; set; }

        /// <summary>User display name</summary>
        [JsonProperty("name")]
        public string Name { get; set; }

        /// <summary>User age</summary>
        [JsonProperty("age")]
        public int? Age { get; set; }

        /// <summary>User status</summary>
        [JsonProperty("status", Required = Required.Always)]
        public UserStatus Status { get; set; }
    }
}
```

---

## 3. Aliases

Aliases create a new type name that wraps a primitive or references another type.

### Primitive Alias

```json
{
  "UserId": {
    "kind": "alias",
    "description": "Unique identifier for a user",
    "target": "string",
    "format": "uuid"
  }
}
```

### Type Reference Alias

```json
{
  "AdminUser": {
    "kind": "alias",
    "description": "An admin user",
    "target": {
      "$ref": "User"
    }
  }
}
```

### Generated Code

**Primitive Alias (TypeScript):**

```typescript
/** Unique identifier for a user */
export type UserId = string;
```

**Primitive Alias (Go):**

```go
package rpcapi

import (
    "github.com/google/uuid"
)

// UserId - Unique identifier for a user
type UserId = uuid.UUID
```

**Type Reference Alias (TypeScript):**

```typescript
/** An admin user */
export type AdminUser = User;
```

**Type Reference Alias (Go):**

```go
// AdminUser - An admin user
type AdminUser = User
```

---

## 4. Maps

Maps are dictionary types with typed keys and values. Keys can be `string` or a type reference (enum). Values can be any field type.

### Type Definition

**String Keys, Primitive Values:**

```json
{
  "StringMap": {
    "kind": "map",
    "description": "A map with string keys and string values",
    "keyType": "string",
    "valueType": {
      "type": "string"
    }
  }
}
```

**Enum Keys, Object Values:**

```json
{
  "UsersByStatus": {
    "kind": "map",
    "description": "Users grouped by status",
    "keyType": {
      "$ref": "UserStatus"
    },
    "valueType": {
      "$ref": "User"
    }
  }
}
```

### Generated Code

**TypeScript:**

```typescript
/** A map with string keys and string values */
export type StringMap = Record<string, string>;

/** Users grouped by status */
export type UsersByStatus = Record<UserStatus, User>;
```

**Go:**

```go
// StringMap - A map with string keys and string values
type StringMap = map[string]string

// UsersByStatus - Users grouped by status
type UsersByStatus = map[UserStatus]User
```

**C#:**

```csharp
/// <summary>A map with string keys and string values</summary>
public class StringMap : Dictionary<string, string> { }

/// <summary>Users grouped by status</summary>
public class UsersByStatus : Dictionary<string, User> { }
```

---

## 5. Arrays

Arrays are list types with typed items.

### Type Definition

```json
{
  "UserList": {
    "kind": "array",
    "description": "A list of users",
    "itemType": {
      "$ref": "User"
    }
  },
  "TagList": {
    "kind": "array",
    "description": "A list of tags",
    "itemType": {
      "type": "string"
    }
  }
}
```

### Generated Code

**TypeScript:**

```typescript
/** A list of users */
export type UserList = User[];

/** A list of tags */
export type TagList = string[];
```

**Go:**

```go
// UserList - A list of users
type UserList = []User

// TagList - A list of tags
type TagList = []string
```

**C#:**

```csharp
/// <summary>A list of users</summary>
public class UserList : List<User> { }

/// <summary>A list of tags</summary>
public class TagList : List<string> { }
```

---

## Type References

Types reference each other by name (not file path):

```json
{
  "profile": {
    "$ref": "UserProfile"
  }
}
```

The generator:

- Validates that referenced types exist
- Tracks dependencies to generate types in the correct order
- Automatically adds imports for referenced types
- Computes "referenced by" relationships for documentation

---

## Nested Types

### Arrays in Object Fields

```json
{
  "tags": {
    "description": "User tags",
    "items": {
      "type": "string"
    }
  }
}
```

Generated as:

- TypeScript: `tags?: string[]`
- Go: `Tags []string`
- C#: `public string[] Tags { get; set; }`

### Maps in Object Fields

```json
{
  "metadata": {
    "description": "User metadata",
    "map": {
      "keyType": "string",
      "valueType": {
        "type": "string"
      }
    }
  }
}
```

Generated as:

- TypeScript: `metadata?: Record<string, string>`
- Go: `Metadata map[string]string`
- C#: `public Dictionary<string, string> Metadata { get; set; }`

### Nullable Items

```json
{
  "itemType": {
    "type": "string",
    "nullable": true
  }
}
```

Generated as:

- TypeScript: `Array<string | null>`
- Go: `[]*string`
- C#: `string?[]`

---

## Best Practices

### File Organization

- Group related types in the same `.type.json` file
- Use descriptive filenames (e.g., `user.type.json`, `auth.type.json`)
- Place all type files in the `schemas/` directory

### Naming Conventions

- **Type names**: PascalCase - `UserProfile`, `AuthToken`, `PingStatus`
- **Field names**: camelCase - `userId`, `createdAt`, `isActive`
- **Enum values**: lowercase with dots - `"user.create"`, `"data.updated"`

### Documentation

- Always provide a `description` for every type
- Add descriptions to all fields and enum values
- Keep descriptions concise but informative
- Use active voice: "User account information" not "Information about a user account"

### Type Design

- Use **enums** for fixed sets of string values (status codes, event types)
- Use **objects** for structured data with named fields
- Use **aliases** to create semantic types (`UserId` instead of bare `string`)
- Use **maps** for dictionary-like data with dynamic keys
- Use **arrays** for lists of items
- Avoid deeply nested structures - extract complex nested types as separate top-level types

### Strong Typing

- **Never use `any`** - This system intentionally does not support `interface{}`/`any`
- Model all data with explicit types
- Use union types via enums or proper object modeling
- If you need dynamic data, use `map<string, ConcreteType>` with a well-defined concrete type

---

## Validation

Type definitions are validated against `type-schema.json` which enforces:

- Valid type kind values
- Required fields for each type kind
- Valid primitive types
- Valid string formats
- Proper type reference patterns
- Valid field and type naming conventions

Validation can be run using:

```bash
npx ajv-cli validate -s type-schema.json -d "schemas/*.type.json"
```

---

## Metadata Export

The generator exports rich metadata for documentation in `api_docs.json`:

### Type Metadata

- **kind** - Type kind (enum, object, alias, map, array)
- **description** - Type description
- **typeDefinition** - Raw JSON definition from `.type.json`
- **jsonRepresentation** - Example JSON instance
- **goRepresentation** - Generated Go code
- **tsRepresentation** - Generated TypeScript code

### Enum Metadata

- **enumValues** - Array of `{value, description}` objects

### Object Metadata

- **fields** - Array of field metadata:
  - `name` - Field name
  - `description` - Field description
  - `type` - Base type (string, number, integer, boolean, or type name)
  - `format` - String format (uuid, date-time) if applicable
  - `optional` - Whether field can be omitted
  - `nullable` - Whether field can be null
  - `isRef` - Whether field is a type reference
  - `refTypeName` - Referenced type name if isRef

### Alias Metadata

- **aliasTarget** - Target type name or primitive

### Map Metadata

- **mapValueType** - Value type
- **mapValueIsRef** - Whether value is a type reference

### Array Metadata

- **arrayItemType** - Item type
- **arrayItemIsRef** - Whether item is a type reference

### Relationship Metadata

- **references** - Types this type uses
- **referencedBy** - Types that use this type (computed)

This metadata powers the interactive documentation website with:

- Syntax-highlighted code examples
- Clickable type links
- Field descriptions with format information
- Type dependency graphs
