package generate

import (
	"os"
	"reflect"
	"sort"
	"strings"
	"ws-json-rpc/pkg/utils"
)

// Docs is the top level structure for the generated docs
type Docs struct {
	Events          map[string]EventDocs   `json:"events"`
	Handlers        map[string]HandlerDocs `json:"handlers"`
	TypescriptTypes map[string]string      `json:"typescriptTypes"`
	JSONTypes       map[string]string      `json:"jsonTypes"`
}

// ReqResType is a type that contains JSON and Typescript representations
type ReqResType struct {
	JSONKey       string `json:"jsonKey"`
	TypescriptKey string `json:"typescriptKey"`
}

// EventDocs is the structure for the docs of an event
type EventDocs struct {
	Title         string         `json:"title"`
	Group         string         `json:"group"`
	Description   string         `json:"description"`
	Deprecated    bool           `json:"deprecated"`
	ResultTypeKey string         `json:"resultKey"`
	Examples      []EventExample `json:"examples"`
}

// EventExample is an example of an event
type EventExample struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Result      any    `json:"-"`
	ResultType  string `json:"result"`
}

// HandlerDocs is the structure for the docs of a handler
type HandlerDocs struct {
	Title         string           `json:"title"`
	Group         string           `json:"group"`
	Description   string           `json:"description"`
	Deprecated    bool             `json:"deprecated"`
	ParamsTypeKey string           `json:"paramsKey"`
	ResultTypeKey string           `json:"resultKey"`
	Examples      []HandlerExample `json:"examples"`
}

type StringifiedType[T any] struct {
	Value T
}

func (s StringifiedType[T]) MarshalJSON() ([]byte, error) {
	return utils.ToJSON(s.Value)
}

// HandlerExample is an example of a handler
type HandlerExample struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Params      any    `json:"-"`
	Result      any    `json:"-"`
	ParamsType  string `json:"params"`
	ResultType  string `json:"result"`
}

type realGenerator struct {
	typescriptTypeCache map[string]string
	docs                Docs
}

func newRealGenerator(tsTypes map[string]string) *realGenerator {
	return &realGenerator{
		typescriptTypeCache: tsTypes,
		docs: Docs{
			Events:          make(map[string]EventDocs),
			Handlers:        make(map[string]HandlerDocs),
			TypescriptTypes: make(map[string]string),
			JSONTypes:       make(map[string]string),
		},
	}
}
func (g *realGenerator) mustGetJSONRepresentation(v any) string {
	if v == nil || v == struct{}{} {
		return "null"
	}

	jsonStr, err := utils.ToJSON(v)
	if err != nil {
		panic("failed to marshal event type to JSON: " + err.Error())
	}
	return string(jsonStr)
}

func (g *realGenerator) mustGetJSONName(v any) string {
	if v == nil || v == struct{}{} {
		return "null"
	}

	return reflect.TypeOf(v).Name()
}

func (g *realGenerator) AddEventType(name string, resp any, docs EventDocs) {
	for i, ex := range docs.Examples {
		if reflect.TypeOf(ex.Result) != reflect.TypeOf(resp) {
			panic("example result type does not match event result type")
		}
		docs.Examples[i].ResultType = g.mustGetJSONRepresentation(ex.Result)
	}

	resultTypeName := g.mustGetJSONName(resp)
	docs.ResultTypeKey = resultTypeName

	g.addJSONType(resultTypeName, resp)
	g.docs.Events[name] = docs
}

func (g *realGenerator) addJSONType(name string, v any) {
	jsonRep := g.mustGetJSONRepresentation(v)
	if typ, exists := g.docs.JSONTypes[name]; exists {
		if typ != jsonRep {
			panic("conflicting JSON type for " + name)
		}
	}
	g.docs.JSONTypes[name] = jsonRep
}

func (g *realGenerator) AddHandlerType(name string, req any, resp any, docs HandlerDocs) {
	for i, ex := range docs.Examples {
		if reflect.TypeOf(ex.Params) != reflect.TypeOf(req) {
			panic("example params type does not match handler params type")
		}
		if reflect.TypeOf(ex.Result) != reflect.TypeOf(resp) {
			panic("example result type does not match handler result type")
		}
		docs.Examples[i].ParamsType = g.mustGetJSONRepresentation(ex.Params)
		docs.Examples[i].ResultType = g.mustGetJSONRepresentation(ex.Result)
	}

	paramTypeName := g.mustGetJSONName(req)
	docs.ParamsTypeKey = paramTypeName
	g.addJSONType(paramTypeName, req)

	resultTypeName := g.mustGetJSONName(resp)
	docs.ResultTypeKey = resultTypeName
	g.addJSONType(resultTypeName, resp)

	g.docs.Handlers[name] = docs
}

func (g *realGenerator) Run() {
	const rpcTypes = "rpc.ts"
	const rpcDocs = "rpc.json"

	// Write typescript types
	sortedTypesNames := make([]string, 0, len(g.typescriptTypeCache))
	for typeName := range g.typescriptTypeCache {
		sortedTypesNames = append(sortedTypesNames, typeName)
	}
	sort.Strings(sortedTypesNames)
	var sb strings.Builder
	// TODO: this should be a map type with method: req/res types and event: res types
	for _, typeName := range sortedTypesNames {
		typ := g.typescriptTypeCache[typeName]
		sb.WriteString(typ)
		sb.WriteString("\n")
		g.docs.TypescriptTypes[typeName] = typ
	}
	if err := os.WriteFile(rpcTypes, []byte(sb.String()), 0644); err != nil {
		panic("failed to write types to file: " + err.Error())
	}

	// Write docs
	jsonStr, err := utils.ToJSON(g.docs)
	if err != nil {
		panic("failed to marshal docs to JSON: " + err.Error())
	}
	if err := os.WriteFile(rpcDocs, jsonStr, 0644); err != nil {
		panic("failed to write docs to file: " + err.Error())
	}

}
