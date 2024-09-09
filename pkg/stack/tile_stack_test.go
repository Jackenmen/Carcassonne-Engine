package stack

import (
	"errors"
	"slices"
	"testing"
)

type Tile struct {
	id int
}

func (t Tile) Equals(other Tile) bool {
	return t.id == other.id
}

func TestDeepClone(t *testing.T) {
	tiles := []Tile{{0}, {1}, {2}, {3}}

	original := NewOrdered(tiles)
	expected := original.GetRemainingTileCount()
	clone := original.DeepClone()

	actual := clone.GetRemainingTileCount()
	if actual != expected {
		t.Fatalf("expected %#v, got %#v instead", expected, actual)
	}

	if _, err := clone.Next(); err != nil {
		t.Fatal(err.Error())
	}

	actual = clone.GetRemainingTileCount()
	if actual != expected-1 {
		t.Fatalf("expected %#v, got %#v instead", expected-1, actual)
	}
	actual = original.GetRemainingTileCount()
	if actual != expected {
		t.Fatalf("expected %#v, got %#v instead", expected, actual)
	}
}

func TestStandardOrder(t *testing.T) {
	tiles := []Tile{{0}, {1}, {2}, {3}}
	stack := NewOrdered(tiles)
	for i := range len(tiles) {
		tile, err := stack.Next()
		if err != nil {
			t.Fatal(err.Error())
		}
		if tile != tiles[i] {
			t.Fail()
		}

	}
}

func TestSetSeed(t *testing.T) {
	tiles := []Tile{{0}, {1}, {2}, {3}}
	expectedOrder := []int32{2, 3, 0, 1}
	stack := NewSeeded(tiles, 42)
	for i := range len(tiles) {
		tile, err := stack.Next()
		if err != nil {
			t.Fatal(err.Error())
		}
		if int32(tile.id) != expectedOrder[i] {
			t.Fail()
		}

	}
}

func TestPeek(t *testing.T) {
	tiles := []Tile{{0}, {1}, {2}, {3}}
	stack := NewOrdered(tiles)
	for range len(tiles) {
		tileA, err := stack.Peek()
		if err != nil {
			t.Fatal(err.Error())
		}
		tileB, err := stack.Next()
		if err != nil {
			t.Fatal(err.Error())
		}
		if tileA != tileB {
			t.Fail()
		}
	}
}

func TestOutOfBounds(t *testing.T) {
	tiles := []Tile{{0}}
	stack := NewOrdered(tiles)
	_, err := stack.Next()
	if err != nil {
		t.Fail()
	}
	_, err = stack.Peek()
	if err == nil {
		t.Fail()
	}
	if err == nil || !errors.Is(err, ErrStackOutOfBounds) {
		t.Fail()
	}
	_, err = stack.Next()
	if err == nil || !errors.Is(err, ErrStackOutOfBounds) {
		t.Fail()
	}
}

func TestRemaining(t *testing.T) {
	tiles := []Tile{{0}, {1}, {2}, {3}}
	stack := NewOrdered(tiles)
	for range 2 {
		_, err := stack.Next()
		if err != nil {
			t.Fail()
		}
	}
	remaining := stack.GetRemaining()
	if remaining[0] != tiles[2] {
		t.Fail()
	}
	if remaining[1] != tiles[3] {
		t.Fail()
	}
}

func TestGetTiles(t *testing.T) {
	expected := []Tile{{0}, {1}, {2}, {3}}
	stack := NewOrdered(expected)
	for range 2 {
		_, err := stack.Next()
		if err != nil {
			t.Fail()
		}
	}
	actual := stack.GetTiles()
	if !slices.Equal(actual, expected) {
		t.Fail()
	}
}

func TestTotalTileCount(t *testing.T) {
	tiles := []Tile{{0}, {1}, {2}, {3}}
	tileCount := int32(len(tiles))
	stack := NewOrdered(tiles)
	if stack.GetTotalTileCount() != tileCount {
		t.Fail()
	}
	for range 2 {
		_, err := stack.Next()
		if err != nil {
			t.Fail()
		}
	}
	if stack.GetTotalTileCount() != tileCount {
		t.Fail()
	}
}

func TestRemainingTileCount(t *testing.T) {
	tiles := []Tile{{0}, {1}, {2}, {3}}
	tileCount := int32(len(tiles))
	stack := NewOrdered(tiles)
	if stack.GetRemainingTileCount() != tileCount {
		t.Fail()
	}
	for range 2 {
		_, err := stack.Next()
		if err != nil {
			t.Fail()
		}
	}
	if stack.GetRemainingTileCount() != tileCount-2 {
		t.Fail()
	}
}

func TestMoveToTopReturnsErrorWhenNoMatchingTileLeft(t *testing.T) {
	tiles := []Tile{{0}, {1}, {2}, {3}}
	stack := NewOrdered(tiles)
	if _, err := stack.Next(); err != nil {
		t.Fatal(err)
	}

	err := stack.MoveToTop(Tile{0})
	if err == nil || !errors.Is(err, ErrTileNotFound) {
		t.Fatal(err)
	}

	expectedRemaining := []Tile{{1}, {2}, {3}}
	remaining := stack.GetRemaining()
	if !slices.Equal(remaining, expectedRemaining) {
		t.Fatalf("expected %#v, got %#v instead", expectedRemaining, remaining)
	}
}

func TestMoveToTopUpdatesOrderProperly(t *testing.T) {
	tiles := []Tile{{0}, {1}, {2}, {3}}
	stack := NewOrdered(tiles)
	if _, err := stack.Next(); err != nil {
		t.Fatal(err)
	}

	if err := stack.MoveToTop(Tile{2}); err != nil {
		t.Fatal(err)
	}

	expectedRemaining := []Tile{{2}, {1}, {3}}
	remaining := stack.GetRemaining()
	if !slices.Equal(remaining, expectedRemaining) {
		t.Fatalf("expected %#v, got %#v instead", expectedRemaining, remaining)
	}
}
