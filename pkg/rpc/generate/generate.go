package generate

import (
	"os"
)

type Generator interface {
	AddEventType(name string, resp any, docs EventDocs)
	AddHandlerType(name string, req any, resp any, docs HandlerDocs)
	Run()
}

func NewGenerator(tsTypes map[string]string) Generator {
	// Return a no-op generator if GENERATE is not set
	// So production does not waste resources on code generation
	isGen := os.Getenv("GENERATE") == "true"
	if !isGen {
		return &fakeGenerator{}
	}

	return newRealGenerator(tsTypes)
}
