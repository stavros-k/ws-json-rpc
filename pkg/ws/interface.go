package ws

type MethodKinder interface {
	IsMethodKind()
	String() string
}

type EventKinder interface {
	IsEventKind()
	String() string
}
