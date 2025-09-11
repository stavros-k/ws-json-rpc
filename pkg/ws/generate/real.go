package generate

import "reflect"

type realGenerator struct {
	typeCache    map[reflect.Type]string
	eventTypes   map[string]eventType
	handlerTypes map[string]handlerInfo
}

func (g *realGenerator) AddEventType(name string, resp any, docs EventDocs) {
	g.eventTypes[name] = eventType{respType: resp, docs: docs}
}

func (g *realGenerator) AddHandlerType(name string, req any, resp any, docs HandlerDocs) {
	g.handlerTypes[name] = handlerInfo{reqType: req, respType: resp, docs: docs}
}
