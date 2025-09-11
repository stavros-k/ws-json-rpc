package generate

import "reflect"

type handlerInfo struct {
	reqType  any
	respType any
	docs     HandlerDocs
}

type HandlerDocs struct {
}

type eventType struct {
	respType any
	docs     EventDocs
}

type EventDocs struct {
}

type Generator struct {
	typeCache    map[reflect.Type]string
	eventTypes   map[string]eventType
	handlerTypes map[string]handlerInfo
}

func NewGenerator() *Generator {
	return &Generator{
		typeCache:    make(map[reflect.Type]string),
		eventTypes:   make(map[string]eventType),
		handlerTypes: make(map[string]handlerInfo),
	}
}

func (g *Generator) AddEventType(name string, resp any, docs EventDocs) {
	g.eventTypes[name] = eventType{respType: resp, docs: docs}
}

func (g *Generator) AddHandlerType(name string, req any, resp any, docs HandlerDocs) {
	g.handlerTypes[name] = handlerInfo{reqType: req, respType: resp, docs: docs}
}
