package generate

type MockGenerator struct{}

func (g *MockGenerator) Generate() error                                                { return nil }
func (g *MockGenerator) AddEventType(name string, resp any, docs EventDocs)             {}
func (g *MockGenerator) AddHandlerType(name string, req any, resp any, docs MethodDocs) {}
