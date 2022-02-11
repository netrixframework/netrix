package types

type MessageParser interface {
	Parse([]byte) (ParsedMessage, error)
}

type ParsedMessage interface {
	String() string
	Clone() ParsedMessage
	Marshal() ([]byte, error)
}
