package corbierror

var HTTPCodes = map[ErrorType]int{
	BadRequest:   400,
	Unauthorized: 401,
	Forbidden:    403,
	NotFound:     404,
	Conflict:     409,
	Internal:     500,
}

var HTTPMessages = map[ErrorType]string{
	BadRequest:   "Bad request",
	Unauthorized: "Unauthorized",
	Forbidden:    "Forbidden",
	NotFound:     "Not Found",
	Conflict:     "Conflict",
	Internal:     "Internal error",
}

func HTTPStatusCode(e Error) int {
	if errorStatus, ok := HTTPCodes[e.Type]; ok {
		return errorStatus
	}
	return 500
}

func HTTPErrorMessage(e Error) string {
	if message, ok := HTTPMessages[e.Type]; ok {
		return message
	}
	return "Internal error"
}

func HTTPError(e Error) (Error, int) {
	status := HTTPStatusCode(e)
	if e.Exposable {
		return e, status
	}
	e.Messages = []string{HTTPErrorMessage(e)}
	return e, status
}
