package generate

// RouteMetadataCollector is the interface that RouteBuilder uses to collect route metadata
type RouteMetadataCollector interface {
	RegisterRoute(route *RouteInfo)
	Generate() error
}

// NoopCollector is a no-op implementation of RouteMetadataCollector
type NoopCollector struct{}

func (n *NoopCollector) RegisterRoute(route *RouteInfo) {}
func (n *NoopCollector) Generate() error                { return nil }
