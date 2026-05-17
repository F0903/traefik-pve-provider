package target

type CaptureNames struct {
	stringCaptures map[string]bool
	intCaptures    map[string]bool
}

func NewCaptureNames() CaptureNames {
	return CaptureNames{
		stringCaptures: make(map[string]bool),
		intCaptures:    make(map[string]bool),
	}
}

func (c CaptureNames) AddString(capture string) {
	c.stringCaptures[capture] = true
}

func (c CaptureNames) AddInt(capture string) {
	c.intCaptures[capture] = true
}

func (c CaptureNames) HasString(capture string) bool {
	return c.stringCaptures[capture]
}

func (c CaptureNames) HasInt(capture string) bool {
	return c.intCaptures[capture]
}

type Match struct {
	stringCaptures map[string]string
	intCaptures    map[string]int
}

func NewMatch() Match {
	return Match{
		stringCaptures: make(map[string]string),
		intCaptures:    make(map[string]int),
	}
}

func (m Match) SetString(capture, value string) {
	m.stringCaptures[capture] = value
}

func (m Match) SetInt(capture string, value int) {
	m.intCaptures[capture] = value
}

func (m Match) String(capture string) string {
	return m.stringCaptures[capture]
}

func (m Match) Int(capture string) int {
	return m.intCaptures[capture]
}
