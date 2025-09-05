package wshub

type MethodKinder interface {
	IsMethodKind()
	String() string
}
