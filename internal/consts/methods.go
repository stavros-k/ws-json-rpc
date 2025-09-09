package consts

type MethodKind string

func (m MethodKind) String() string { return string(m) }

const (
	MethodKindEcho        MethodKind = "echo"
	MethodKindAdd         MethodKind = "add"
	MethodKindDouble      MethodKind = "double"
	MethodKindPing        MethodKind = "ping"
	MethodKindSubscribe   MethodKind = "subscribe"
	MethodKindUnsubscribe MethodKind = "unsubscribe"
)
