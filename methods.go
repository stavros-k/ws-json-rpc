package main

type MethodKind string

func (MethodKind) IsMethodKind() {}
func (mk MethodKind) String() string {
	return string(mk)
}

const (
	MethodKindEcho      MethodKind = "echo"
	MethodKindAdd       MethodKind = "add"
	MethodKindDouble    MethodKind = "double"
	MethodKindComplex   MethodKind = "complex"
	MethodKindPing      MethodKind = "ping"
	MethodKindGetUser   MethodKind = "get.user"
	MethodKindSubscribe MethodKind = "subscribe"
)
