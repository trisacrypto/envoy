package contextkey

type ContextKey uint8

const (
	KeyUnknown ContextKey = iota
	KeyRequestID
	KeyActorID
	KeyActorType
)
