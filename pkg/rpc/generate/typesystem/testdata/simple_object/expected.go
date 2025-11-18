package models

// Person - A person entity
type Person struct {
	// Person's age
	Age int64 `json:"age"`
	// Person's email address
	Email string `json:"email,omitzero"`
	// Person's name
	Name string `json:"name"`
}
