package models

// Color - Available colors
type Color string

const (
	// Red color
	ColorRed Color = "red"
	// Green color
	ColorGreen Color = "green"
	// Blue color
	ColorBlue Color = "blue"
)

// Valid returns true if the Color value is valid
func (e Color) Valid() bool {
	switch e {
	case ColorRed, ColorGreen, ColorBlue:
		return true
	default:
		return false
	}
}
