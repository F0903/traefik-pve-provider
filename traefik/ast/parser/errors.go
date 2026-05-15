package parser

type ParseErrorKind int

const (
	ErrUnsupported ParseErrorKind = iota
	ErrInvalidBoolean
	ErrInvalidInteger
)

type ParseError struct {
	Kind ParseErrorKind
}

func (e *ParseError) Error() string {
	switch e.Kind {
	case ErrInvalidBoolean:
		return "invalid boolean"
	case ErrInvalidInteger:
		return "invalid integer"
	default:
		return "unsupported label"
	}
}

type Context struct {
	DefaultName string
}

func unsupportedLabel() *ParseError {
	return &ParseError{Kind: ErrUnsupported}
}
