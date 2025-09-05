package main

type MethodKind string

func (MethodKind) IsMethodKind() {}
func (mk MethodKind) String() string {
	return string(mk)
}

const (
	MethodKindEcho        MethodKind = "echo"
	MethodKindAdd         MethodKind = "add"
	MethodKindDouble      MethodKind = "double"
	MethodKindPing        MethodKind = "ping"
	MethodKindSubscribe   MethodKind = "subscribe"
	MethodKindUnsubscribe MethodKind = "unsubscribe"
)
