package http

// Response a type for HTTP responses
type Response struct {
	Messages []string `json:"messages"`
}

func NewResponse(messages ...string) Response {
	return Response{
		Messages: messages,
	}
}
