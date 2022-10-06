package execute

type ResultsFilter interface {
	filter(string) string
}

type EchoFilter struct{}

func (f EchoFilter) filter(in string) string {
	return in
}

type TextFilter struct {
	value string
}

func (f TextFilter) filter(in string) string {
	return in
}
