package cloudslack

type CloudSlackErrorType string

const (
	CloudSlackErrorBadRequest CloudSlackErrorType = "BadRequest"
)

type CloudSlackError struct {
	Msg     string              `json:"message"`
	ErrType CloudSlackErrorType `json:"type"`
}

func NewBadRequestError(msg string) *CloudSlackError {
	return &CloudSlackError{
		Msg:     msg,
		ErrType: CloudSlackErrorBadRequest,
	}
}

func (e *CloudSlackError) Error() string {
	return e.Msg
}

func (e *CloudSlackError) IsBadRequest() bool {
	return e.ErrType == CloudSlackErrorBadRequest
}

func IsBadRequestErr(err error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*CloudSlackError)
	return ok && e.IsBadRequest()
}
