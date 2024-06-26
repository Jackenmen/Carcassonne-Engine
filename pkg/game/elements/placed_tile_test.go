package elements

import (
	"slices"
	"testing"

	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tiles/tiletemplates"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tilesets"
)

func TestPositionMarshalTextWithPositiveCoords(t *testing.T) {
	pos := NewPosition(1, 3)
	expected := []byte("1,3")
	actual, err := pos.MarshalText()
	if err != nil {
		t.Fatal(err.Error())
	}
	if !slices.Equal(actual, expected) {
		t.Fatalf("expected %#v, got %#v instead", expected, actual)
	}
}

func TestPositionMarshalTextWithNegativeCoords(t *testing.T) {
	pos := NewPosition(-31, -5)
	expected := []byte("-31,-5")
	actual, err := pos.MarshalText()
	if err != nil {
		t.Fatal(err.Error())
	}
	if !slices.Equal(actual, expected) {
		t.Fatalf("expected %#v, got %#v instead", expected, actual)
	}
}

func TestPositionUnmarshalTextWithPositiveCoords(t *testing.T) {
	text := []byte("1,3")
	expectedX := int16(1)
	expectedY := int16(3)

	actual := Position{}
	err := actual.UnmarshalText(text)
	if err != nil {
		t.Fatal(err.Error())
	}
	if actual.X() != expectedX {
		t.Fatalf("expected %#v, got %#v instead", expectedX, actual)
	}
	if actual.Y() != expectedY {
		t.Fatalf("expected %#v, got %#v instead", expectedY, actual)
	}
}

func TestPositionUnmarshalTextWithNegativeCoords(t *testing.T) {
	text := []byte("-31,-5")
	expectedX := int16(-31)
	expectedY := int16(-5)

	actual := Position{}
	err := actual.UnmarshalText(text)
	if err != nil {
		t.Fatal(err.Error())
	}
	if actual.X() != expectedX {
		t.Fatalf("expected %#v, got %#v instead", expectedX, actual)
	}
	if actual.Y() != expectedY {
		t.Fatalf("expected %#v, got %#v instead", expectedY, actual)
	}
}

func TestTilePlacementRotate(t *testing.T) {
	move := TilePlacement{
		Tile: tiletemplates.SingleCityEdgeNoRoads(),
		Pos:  NewPosition(0, 1),
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected TilePlacement.Rotate() to panic")
		}
	}()

	move.Rotate(1)
}

func TestNewStartingTile(t *testing.T) {
	tileSet := tilesets.StandardTileSet()
	actual := NewStartingTile(tileSet)

	expectedTile := tileSet.StartingTile
	if !actual.Tile.Equals(expectedTile) {
		t.Fatalf("expected %#v, got %#v instead", expectedTile, actual.Tile)
	}

	expectedPos := NewPosition(0, 0)
	if actual.Pos != expectedPos {
		t.Fatalf("expected %#v, got %#v instead", expectedPos, actual.Pos)
	}
}
