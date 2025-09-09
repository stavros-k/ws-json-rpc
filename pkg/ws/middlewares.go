package ws

type MiddlewareFunc func(HandlerFunc) HandlerFunc

func applyMiddleware(handler HandlerFunc, middlewares ...MiddlewareFunc) HandlerFunc {
	for _, mw := range middlewares {
		handler = mw(handler)
	}
	return handler
}
