package consts

type EventKind string

func (EventKind) IsEventKind() {}
func (e EventKind) String() string {
	return string(e)
}

const (
	EventKindUserUpdate    EventKind = "user.update"
	EventKindUserLogin     EventKind = "user.login"
	EventKindDataProcessed EventKind = "data.processed"
)
