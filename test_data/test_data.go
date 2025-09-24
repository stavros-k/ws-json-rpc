package testdata

import (
	"time"
)

// MyEnum is a custom type used for testing purposes.
type MyEnum string

const (
	// Some comment
	MyEnum1 MyEnum = "MyEnum1" // Some inline comment
	MyEnum2 MyEnum = "MyEnum2"
)

type MyOtherEnum int

const (
	MyOtherEnum1 MyOtherEnum = 1
	MyOtherEnum2 MyOtherEnum = 2
)

// TestData is a struct used for testing purposes.
type TestData struct {
	StringField string  `json:"stringField"`
	IntField    int     `json:"intField"`
	Int8Field   int8    `json:"int8Field"`
	Int16Field  int16   `json:"int16Field"`
	Int32Field  int32   `json:"int32Field"`
	Int64Field  int64   `json:"int64Field"`
	UintField   uint    `json:"uintField"`
	Uint8Field  uint8   `json:"uint8Field"`
	Uint16Field uint16  `json:"uint16Field"`
	Uint32Field uint32  `json:"uint32Field"`
	Uint64Field uint64  `json:"uint64Field"`
	FloatField  float32 `json:"floatField"`
	DoubleField float64 `json:"doubleField"`
	BoolField   bool    `json:"boolField"`
	EnumField   MyEnum  `json:"enumField"`

	OptionalStringField *string  `json:"optionalStringField,omitempty"`
	OptionalIntField    *int     `json:"optionalIntField,omitempty"`
	OptionalInt8Field   *int8    `json:"optionalInt8Field,omitempty"`
	OptionalInt16Field  *int16   `json:"optionalInt16Field,omitempty"`
	OptionalInt32Field  *int32   `json:"optionalInt32Field,omitempty"`
	OptionalInt64Field  *int64   `json:"optionalInt64Field,omitempty"`
	OptionalUintField   *uint    `json:"optionalUintField,omitempty"`
	OptionalUint8Field  *uint8   `json:"optionalUint8Field,omitempty"`
	OptionalUint16Field *uint16  `json:"optionalUint16Field,omitempty"`
	OptionalUint32Field *uint32  `json:"optionalUint32Field,omitempty"`
	OptionalUint64Field *uint64  `json:"optionalUint64Field,omitempty"`
	OptionalFloatField  *float32 `json:"optionalFloatField,omitempty"`
	OptionalDoubleField *float64 `json:"optionalDoubleField,omitempty"`
	OptionalBoolField   *bool    `json:"optionalBoolField,omitempty"`
	OptionalEnumField   *MyEnum  `json:"optionalEnumField,omitempty"`

	StringsField  []string  `json:"stringsField"`
	IntsField     []int     `json:"intsField"`
	Int8sField    []int8    `json:"int8sField"`
	Int16sField   []int16   `json:"int16sField"`
	Int32sField   []int32   `json:"int32sField"`
	Int64sField   []int64   `json:"int64sField"`
	UintsField    []uint    `json:"uintsField"`
	Uint8sField   []uint8   `json:"uint8sField"`
	Uint16sField  []uint16  `json:"uint16sField"`
	Uint32sField  []uint32  `json:"uint32sField"`
	Uint64sField  []uint64  `json:"uint64sField"`
	Float16sField []float32 `json:"float16sField"`
	FloatsField   []float32 `json:"floatsField"`
	DoublesField  []float64 `json:"doublesField"`
	BoolsField    []bool    `json:"boolsField"`
	EnumsField    []MyEnum  `json:"enumsField"`

	OptionalStringsField  *[]string  `json:"optionalStringsField,omitempty"`
	OptionalIntsField     *[]int     `json:"optionalIntsField,omitempty"`
	OptionalInt8sField    *[]int8    `json:"optionalInt8sField,omitempty"`
	OptionalInt16sField   *[]int16   `json:"optionalInt16sField,omitempty"`
	OptionalInt32sField   *[]int32   `json:"optionalInt32sField,omitempty"`
	OptionalInt64sField   *[]int64   `json:"optionalInt64sField,omitempty"`
	OptionalUintsField    *[]uint    `json:"optionalUintsField,omitempty"`
	OptionalUint8sField   *[]uint8   `json:"optionalUint8sField,omitempty"`
	OptionalUint16sField  *[]uint16  `json:"optionalUint16sField,omitempty"`
	OptionalUint32sField  *[]uint32  `json:"optionalUint32sField,omitempty"`
	OptionalUint64sField  *[]uint64  `json:"optionalUint64sField,omitempty"`
	OptionalFloat32sField *[]float32 `json:"optionalFloat32sField,omitempty"`
	OptionalFloat64sField *[]float64 `json:"optionalFloat64sField,omitempty"`
	OptionalBoolsField    *[]bool    `json:"optionalBoolsField,omitempty"`
	OptionalEnumsField    *[]MyEnum  `json:"optionalEnumsField,omitempty"`

	NestedTypeField        NestedType `json:"nestedTypeField"`
	NestedEmbededTypeField JSONTime   `json:"nestedEmbededTypeField"`

	// TODO: Maps
	// TestData12 map[string]string `json:"testData12"`
	// TestData13 map[string]string `json:"testData13,omitempty"`
	// TestData14 map[string]int    `json:"testData14"`
	// TestData15 map[string]int    `json:"testData15,omitempty"`
}

type NestedType struct {
	StringField string `json:"stringField"`
}

type JSONTime struct {
	time.Time
}
