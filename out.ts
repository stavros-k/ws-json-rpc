// From: ws-json-rpc/test_data.test_data.go:7
export type Anything = any;

// From: ws-json-rpc/test_data.test_data.go:165
export type JSONTime = {
  ...time.Time;
};

// From: ws-json-rpc/test_data.test_data.go:33
// MyEnum is a custom type used for testing purposes.
export type MyEnum = 
  // Some comment | Some inline comment
  | "MyEnumValue1"
  | "MyEnumValue2";

// From: ws-json-rpc/test_data.test_data.go:33
// MyEnum is a custom type used for testing purposes.
export const MyEnumValues = {
  // Some comment | Some inline comment
  MyEnum1: "MyEnumValue1",
  MyEnum2: "MyEnumValue2",
} as const;

// From: ws-json-rpc/test_data.test_data.go:21
// MyInt is a custom integer type used for testing purposes.
export type MyInt = number; // int;

// From: ws-json-rpc/test_data.test_data.go:11
export type MyMap = Record<string, number>;

// From: ws-json-rpc/test_data.test_data.go:49
export type MyOtherEnum = 
  | 1
  | 2;

// From: ws-json-rpc/test_data.test_data.go:49
export const MyOtherEnumValues = {
  MyOtherEnum1: 1,
  MyOtherEnum2: 2,
} as const;

// From: ws-json-rpc/test_data.test_data.go:161
export type NestedType = {
  stringField: string;
};

// From: ws-json-rpc/test_data.test_data.go:24
// SomeStruct is a struct used for testing purposes.
export type SomeStruct = {
  field1: string;
  field2: number;
};

// From: ws-json-rpc/test_data.test_data.go:64
// TestData is a struct used for testing purposes.
export type TestData = {
  interfaceField: any;
  stringField: string;
  intField: number;
  int8Field: number;
  int16Field: number;
  int32Field: number;
  int64Field: number;
  uintField: number;
  uint8Field: number;
  uint16Field: number;
  uint32Field: number;
  uint64Field: number;
  floatField: number;
  doubleField: number;
  boolField: boolean;
  enumField: MyEnum;
  optionalStringField?: string | null;
  optionalIntField?: number | null;
  optionalInt8Field?: number | null;
  optionalInt16Field?: number | null;
  optionalInt32Field?: number | null;
  optionalInt64Field?: number | null;
  optionalUintField?: number | null;
  optionalUint8Field?: number | null;
  optionalUint16Field?: number | null;
  optionalUint32Field?: number | null;
  optionalUint64Field?: number | null;
  optionalFloatField?: number | null;
  optionalDoubleField?: number | null;
  optionalBoolField?: boolean | null;
  optionalEnumField?: MyEnum | null;
  stringsField: Array<string>;
  intsField: Array<number>;
  int8sField: Array<number>;
  int16sField: Array<number>;
  int32sField: Array<number>;
  int64sField: Array<number>;
  uintsField: Array<number>;
  uint8sField: Array<number>;
  uint16sField: Array<number>;
  uint32sField: Array<number>;
  uint64sField: Array<number>;
  float16sField: Array<number>;
  floatsField: Array<number>;
  doublesField: Array<number>;
  boolsField: Array<boolean>;
  enumsField: Array<MyEnum>;
  fixedStringsField: Array<string>;
  fixedIntsField: Array<number>;
  fixedInt8sField: Array<number>;
  fixedInt16sField: Array<number>;
  fixedInt32sField: Array<number>;
  fixedInt64sField: Array<number>;
  fixedUintsField: Array<number>;
  fixedUint8sField: Array<number>;
  fixedUint16sField: Array<number>;
  fixedUint32sField: Array<number>;
  fixedUint64sField: Array<number>;
  fixedFloat32sField: Array<number>;
  fixedFloat64sField: Array<number>;
  fixedBoolsField: Array<boolean>;
  fixedEnumsField: Array<MyEnum>;
  optionalStringsField?: Array<string> | null;
  optionalIntsField?: Array<number> | null;
  optionalInt8sField?: Array<number> | null;
  optionalInt16sField?: Array<number> | null;
  optionalInt32sField?: Array<number> | null;
  optionalInt64sField?: Array<number> | null;
  optionalUintsField?: Array<number> | null;
  optionalUint8sField?: Array<number> | null;
  optionalUint16sField?: Array<number> | null;
  optionalUint32sField?: Array<number> | null;
  optionalUint64sField?: Array<number> | null;
  optionalFloat32sField?: Array<number> | null;
  optionalFloat64sField?: Array<number> | null;
  optionalBoolsField?: Array<boolean> | null;
  optionalEnumsField?: Array<MyEnum> | null;
  nestedTypeField: NestedType;
  nestedEmbededTypeField: JSONTime;
  mapStringStringField: Record<string, string>;
  mapStringIntField: Record<string, number>;
  mapIntStringField: Record<number, string>;
  mapIntIntField: Record<number, number>;
  mapStringMapField: Record<string, Record<string, number>>;
  mapStringEnumField: Record<string, MyEnum>;
};

// From: ws-json-rpc/test_data.test_data.go:8
export type UUID = string;

// From: ws-json-rpc/test_data.test_data.go:9
export type UUIDs = Array<UUID>;

