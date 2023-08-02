package cloudslack

type CloudSlackErrorType string

const (
	CloudSlackErrorQuotaExceeded CloudSlackErrorType = "QuotaExceeded"
)

type CloudSlackError struct {
	Msg     string              `json:"message"`
	ErrType CloudSlackErrorType `json:"type"`
}

func NewQuotaExceededError(msg string) *CloudSlackError {
	return &CloudSlackError{
		Msg:     msg,
		ErrType: CloudSlackErrorQuotaExceeded,
	}
}

func (e *CloudSlackError) Error() string {
	return e.Msg
}

func (e *CloudSlackError) IsQuotaExceeded() bool {
	return e.ErrType == CloudSlackErrorQuotaExceeded
}

func IsQuotaExceededErr(err error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*CloudSlackError)
	return ok && e.IsQuotaExceeded()
}
