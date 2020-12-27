package dashboard

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
)

// Drawer is a interface for drawing components of the view to a screen
type Drawer interface {

	// Draw draws to the given screen
	Draw(screen tcell.Screen)

	// Clear deletes all if any associated drawings from the screen
	Clear(screen tcell.Screen)
}

// Coordinate is a x-y coordinate. The origin (i.e. (0, 0)) of the coordinate system is located at the top left of the
// screen
type Coordinate struct {
	X int
	Y int
}

func (c Coordinate) String() string {
	return fmt.Sprintf("(%d, %d)", c.X, c.Y)
}

// Widget is a basic component which is able to draw a 2D array of text to a screen. If possible, other implementations
// of Drawer should defer to Widget when drawing
type Widget struct {
	Coordinate
	drawing []string
	style   tcell.Style
}

// NewWidget returns a Widget object which is able to draw itself with a style at the x-y offset
func NewWidget(x, y int, drawing []string, style tcell.Style) *Widget {
	return &Widget{
		Coordinate: Coordinate{x, y},
		drawing:    drawing,
		style:      style,
	}
}

func (w *Widget) Draw(screen tcell.Screen) {
	for y, row := range w.drawing {
		for x, char := range []rune(row) {
			screen.SetContent(w.X+x, w.Y+y, char, nil, w.style)
		}
	}
}

func (w *Widget) Clear(screen tcell.Screen) {
	for y := range w.drawing {
		for x := range w.drawing[y] {
			screen.SetContent(w.X+x, w.Y+y, ' ', nil, w.style)
		}
	}
}

// TextWidget is able draw a line of text with a style to at an x-y offset. TextWidget is only able to draw text
// in a left-to-right direction
type TextWidget struct {
	base  *Widget
	style tcell.Style
}

// NewTextWidget returns a new TextWidget object
func NewTextWidget(x, y int, text string, style tcell.Style) *TextWidget {
	return &TextWidget{
		base:  NewWidget(x, y, []string{text}, style),
		style: style,
	}
}

func (t *TextWidget) Draw(screen tcell.Screen) {
	if t.base == nil {
		return
	}

	t.base.Draw(screen)
}

func (t *TextWidget) Clear(screen tcell.Screen) {
	if t.base == nil {
		return
	}

	t.base.Clear(screen)
}

func (t *TextWidget) SetText(text string) {
	t.base.drawing = []string{text}
}

func (t *TextWidget) SetStyle(style tcell.Style) {
	t.base.style = style
}
