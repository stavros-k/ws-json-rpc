package main

const (
	CodeBadRequest = -32700
)

type APIError struct {
	code    int
	message string
}

func (e *APIError) Error() string {
	return e.message
}

func (e *APIError) Code() int {
	return e.code
}

func BadRequest(message string) error {
	return &APIError{code: CodeBadRequest, message: message}
}

type AddRequest struct {
	A int `json:"a"`
	B int `json:"b"`
}

type GetUserRequest struct {
	UserID int      `json:"user_id"`
	Fields []string `json:"fields,omitempty"`
}

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
