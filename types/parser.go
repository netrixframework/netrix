package types

// MessageParser defines a generic parsing interface.
//
// The specific implementation is particular to the protocol being tested.
type MessageParser interface {
	// Parse accepts a byte array and returns a ParsedMessage when successfully parsed, an error otherwise.
	Parse([]byte) (ParsedMessage, error)
}

// ParsedMessage encapsulates a message returned by [MessageParser]
type ParsedMessage interface {
	// String should return a succinct representation of the message. used for logging.
	String() string
	// Clone should create a copy of the current ParsedMessage
	Clone() ParsedMessage
	// Marshal should serialize the ParsedMessage.
	//
	// The byte stream when passed to [MessageParser.Parse] should result in the same ParsedMessage
	Marshal() ([]byte, error)
}
