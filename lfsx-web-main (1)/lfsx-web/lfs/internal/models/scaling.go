package models

// Scaling contains predefined scaling templates with the correct
// mouse and text size
type Scaling struct {
	// Raw scaling percentage for sway based on 100%
	Scaling int

	// Scaling percanted baded on 100% for the font
	ScalingFont int

	// Cursor size in pixels
	CursorSize int
}
