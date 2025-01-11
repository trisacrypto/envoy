package postman

type Direction uint8

const (
	Unknown Direction = iota
	DirectionIncoming
	DirectionOutgoing
)

func (d Direction) String() string {
	switch d {
	case DirectionIncoming:
		return "incoming"
	case DirectionOutgoing:
		return "outgoing"
	default:
		return "unknown"
	}
}

func (d Direction) Valid() bool {
	return d > Unknown && d <= DirectionOutgoing
}
