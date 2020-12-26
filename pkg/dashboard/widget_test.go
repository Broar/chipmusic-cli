package dashboard

import (
	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

type MockScreen struct {
	tcell.Screen
	called int
}

func (m *MockScreen) SetContent(x int, y int, mainc rune, combc []rune, style tcell.Style) {
	m.called++
}

func (m *MockScreen) Show() {
	// Nothing to do
}

func TestTextWidget_Draw_NilBaseWidget(t *testing.T) {
	screen := &MockScreen{}
	widget := &TextWidget{}
	widget.Draw(screen)
	assert.Zero(t, screen.called)
}

func TestTextWidget_Draw(t *testing.T) {
	testCases := []struct {
		name   string
		text   string
		called int
	}{
		{"NoText", "", 0},
		{"OneCharacter", "a", 1},
		{"MultipleCharacters", "abc", 3},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(tt *testing.T) {
			screen := &MockScreen{}
			widget := NewTextWidget(0, 0, testCase.text, tcell.StyleDefault)
			widget.Draw(screen)
			assert.Equal(tt, testCase.called, screen.called)
		})
	}
}

func TestWidget_Draw(t *testing.T) {
	testCases := []struct {
		name    string
		drawing []string
		called  int
	}{
		{"NilDrawing", nil, 0},
		{"EmptyDrawing", []string{}, 0},
		{"OneCharacter", []string{"a"}, 1},
		{"MultipleCharacters", []string{"abc"}, 3},
		{"MixedRows", []string{"abc", ""}, 3},
		{"2DDrawing", []string{"a", "b", "c"}, 3},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(tt *testing.T) {
			screen := &MockScreen{}
			widget := NewWidget(0, 0, testCase.drawing, tcell.StyleDefault)
			widget.Draw(screen)
			assert.Equal(tt, testCase.called, screen.called)
		})
	}
}

func TestCoordinate_String(t *testing.T) {
	testCases := []struct {
		name       string
		coordinate Coordinate
		expected   string
	}{
		{"Default", Coordinate{}, "(0, 0)"},
		{"OnlyX", Coordinate{X: 1}, "(1, 0)"},
		{"OnlyY", Coordinate{Y: 1}, "(0, 1)"},
		{"Both", Coordinate{X: 1, Y: 1}, "(1, 1)"},
		{"Negative", Coordinate{X: -1, Y: -1}, "(-1, -1)"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(tt *testing.T) {
			assert.Equal(tt, testCase.expected, testCase.coordinate.String())
		})
	}
}
