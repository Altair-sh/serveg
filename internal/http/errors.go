package http

type HttpError struct {
	Status StatusCode
	Msg    string
}

func (e *HttpError) Error() string {
	return e.Status.String() + ": " + e.Msg
}

func NewHttpError(code StatusCode, msg string) *HttpError {
	return &HttpError{Status: code, Msg: msg}
}
