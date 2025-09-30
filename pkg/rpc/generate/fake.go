package generate

type fakeGenerator struct{}

func (g *fakeGenerator) AddEventType(name string, resp any, docs EventDocs)              {}
func (g *fakeGenerator) AddHandlerType(name string, req any, resp any, docs HandlerDocs) {}
func (g *fakeGenerator) Run()                                                            {}
