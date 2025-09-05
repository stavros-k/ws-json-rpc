package wshub

type EventKinder interface {
	IsEventKind()
	String() string
}
