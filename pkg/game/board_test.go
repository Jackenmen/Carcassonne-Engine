package game

import (
	"reflect"
	"testing"

	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/game/elements"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/game/position"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/game/test"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tiles"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tiles/tiletemplates"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tilesets"
)

func TestBoardTileCountReturnsOnlyPlacedTiles(t *testing.T) {
	// starting tile has a city on top, we want to close it with a single city tile
	// and then try finding legal moves of a tile filled with a city terrain
	board := NewBoard(tilesets.StandardTileSet())
	_, err := board.PlaceTile(test.GetTestPlacedTile())
	if err != nil {
		t.Fatal(err.Error())
	}

	expected := 2
	actual := board.TileCount()

	if expected != actual {
		t.Fatalf("expected %#v, got %#v instead", expected, actual)
	}
}

func TestBoardGetTilePlacementsForReturnsEmptySliceWhenCityCannotBePlaced(t *testing.T) {
	// starting tile has a city on top, we want to close it with a single city tile
	// and then try finding legal moves of a tile filled with a city terrain
	board := NewBoard(tilesets.StandardTileSet())
	ptile := elements.ToPlacedTile(tiletemplates.SingleCityEdgeNoRoads().Rotate(2))
	ptile.Position = position.New(0, 1)
	_, err := board.PlaceTile(ptile)
	if err != nil {
		t.Fatal(err.Error())
	}

	expected := []elements.PlacedTile{}
	actual := board.GetTilePlacementsFor(tiletemplates.FourCityEdgesConnectedShield())

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %#v, got %#v instead", expected, actual)
	}
}

func TestBoardTileHasValidPlacementReturnsTrueWhenValidPlacementExists(t *testing.T) {
	board := NewBoard(tilesets.StandardTileSet())

	expected := true
	actual := board.TileHasValidPlacement(tiletemplates.SingleCityEdgeNoRoads())

	if expected != actual {
		t.Fatalf("expected %#v, got %#v instead", expected, actual)
	}
}

func TestBoardTileIsPositionValidReturnsTrueWhenJoiningMonasteryWithExistingRoad(t *testing.T) {
	board := NewBoard(tilesets.StandardTileSet()).(*board)

	// tile with road on its left side, monastery at the center, and fields everywhere
	tile := tiletemplates.MonasteryWithSingleRoad().Rotate(1)
	tilePlacement := elements.ToPlacedTile(tile)
	tilePlacement.Position = position.New(1, 0)

	expected := true
	actual := board.isPositionValid(tilePlacement)

	if expected != actual {
		t.Fatalf("expected %#v, got %#v instead", expected, actual)
	}
}

func TestBoardCanBePlacedReturnsTrueWhenPlacedTileCanBePlaced(t *testing.T) {
	board := NewBoard(tilesets.StandardTileSet())

	expected := true
	actual := board.CanBePlaced(test.GetTestPlacedTile())

	if expected != actual {
		t.Fatalf("expected %#v, got %#v instead", expected, actual)
	}
}

func TestBoardPlaceTileErrorsWhenCapacityIsExceeded(t *testing.T) {
	tileSet := tilesets.StandardTileSet()
	tileSet.Tiles = []tiles.Tile{}
	board := NewBoard(tileSet)

	_, err := board.PlaceTile(test.GetTestPlacedTile())
	if err == nil {
		t.Fatal("expected capacity exceeded error to be returned")
	}
}

func TestBoardPlaceTileUpdatesBoardFields(t *testing.T) {
	tileSet := tilesets.StandardTileSet()
	tileSet.Tiles = []tiles.Tile{
		test.GetTestTile(), tiletemplates.FourCityEdgesConnectedShield(),
	}
	board := NewBoard(tileSet)
	expected := test.GetTestPlacedTile()

	_, err := board.PlaceTile(expected)
	if err != nil {
		t.Fatal(err.Error())
	}

	actual := board.Tiles()[1]
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %#v, got %#v instead", expected, actual)
	}

	actual, ok := board.GetTileAt(expected.Position)
	if !ok || !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %#v, got %#v instead (ok = %#v)", expected, actual, ok)
	}
}

func TestBoardPlaceTilePlacesTwoTilesOfSameTypeProperly(t *testing.T) {
	tileSet := tilesets.StandardTileSet()
	tileSet.Tiles = []tiles.Tile{
		test.GetTestTile(),
		tiletemplates.FourCityEdgesConnectedShield(),
		test.GetTestTile(),
	}
	board := NewBoard(tileSet)
	startingPlacedTile := elements.NewStartingTile(tileSet)
	expected := []elements.PlacedTile{
		startingPlacedTile,
		test.GetTestPlacedTile(),
		{},
		test.GetTestPlacedTile(),
	}
	// place the test tile (single city edge) below starting tile
	// (connecting with the field)
	expected[3].Position = position.New(0, -1)

	_, err := board.PlaceTile(expected[1])
	if err != nil {
		t.Fatal(err.Error())
	}

	_, err = board.PlaceTile(expected[3])
	if err != nil {
		t.Fatal(err.Error())
	}

	actual := board.Tiles()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %#v, got %#v instead", expected, actual)
	}
}

func TestBoardScoreInclompleteMonastery(t *testing.T) {
	var report elements.ScoreReport
	var extendedTileSet = tilesets.StandardTileSet()
	for range 3 {
		extendedTileSet.Tiles = append(extendedTileSet.Tiles, tiletemplates.TestOnlyField())
	}
	boardInterface := NewBoard(extendedTileSet)
	board := boardInterface.(*board)

	tiles := []elements.PlacedTile{
		test.GetTestCustomPlacedTile(tiletemplates.MonasteryWithoutRoads()),
		test.GetTestCustomPlacedTile(tiletemplates.TestOnlyField()),
		test.GetTestCustomPlacedTile(tiletemplates.TestOnlyField()),
		test.GetTestCustomPlacedTile(tiletemplates.TestOnlyField()),
	}

	// add meeple to the monastery
	tiles[0].Monastery().Meeple.PlayerID = 1
	tiles[0].Monastery().Meeple.Type = elements.NormalMeeple

	// set positions
	tiles[0].Position = position.New(0, 1)
	tiles[1].Position = position.New(1, 1)
	tiles[2].Position = position.New(0, 2)
	tiles[3].Position = position.New(1, 2)

	// place tiles
	for i, tile := range tiles {
		_, err := board.PlaceTile(tile)
		if err != nil {
			t.Fatalf("error placing tile number: %#v: %#v", i, err)
		}

		report = board.ScoreMonasteries(tile, false)
		if !reflect.DeepEqual(report, elements.NewScoreReport()) {
			t.Fatalf("ScoreMonasteries failed on tile number: %#v. expected %#v, got %#v instead", i, elements.NewScoreReport(), report)
		}
	}

	// test forceScore
	report = board.ScoreMonasteries(tiles[0], true)

	expectedReport := elements.NewScoreReport()
	expectedReport.ReceivedPoints = map[elements.ID]uint32{
		1: 5,
	}
	expectedReport.ReturnedMeeples = map[elements.ID][]elements.MeepleWithPosition{
		1: {elements.NewMeepleWithPosition(
			elements.Meeple{Type: elements.NormalMeeple, PlayerID: elements.ID(1)},
			position.New(0, 1),
		)},
	}

	if !reflect.DeepEqual(report, expectedReport) {
		t.Fatalf("ScoreMonasteries failed when forceScore=true. expected:\n%#v,\ngot:\n%#v instead", expectedReport, report)
	}
}

func TestBoardCompleteTwoMonasteriesAtOnce(t *testing.T) {
	/*
		the board setup is as follows:
		FFFF
		FMMF
		FFFF
		 S

		F - field
		M - monastery
		S - starting tile

		left monastery is at (0,2)
		right monastery is at (1,2)
		right monastery is placed as the last tile
	*/

	var report elements.ScoreReport
	var extendedTileSet = tilesets.StandardTileSet()
	for range 10 {
		extendedTileSet.Tiles = append(extendedTileSet.Tiles, tiletemplates.TestOnlyField())
	}
	boardInterface := NewBoard(extendedTileSet)
	board := boardInterface.(*board)

	tiles := []elements.PlacedTile{
		test.GetTestCustomPlacedTile(tiletemplates.TestOnlyField()),
		test.GetTestCustomPlacedTile(tiletemplates.MonasteryWithoutRoads()),
		test.GetTestCustomPlacedTile(tiletemplates.TestOnlyField()),
		test.GetTestCustomPlacedTile(tiletemplates.TestOnlyField()),
		test.GetTestCustomPlacedTile(tiletemplates.TestOnlyField()),
		test.GetTestCustomPlacedTile(tiletemplates.TestOnlyField()),
		test.GetTestCustomPlacedTile(tiletemplates.TestOnlyField()),
		test.GetTestCustomPlacedTile(tiletemplates.TestOnlyField()),
		test.GetTestCustomPlacedTile(tiletemplates.TestOnlyField()),
		test.GetTestCustomPlacedTile(tiletemplates.TestOnlyField()),
		test.GetTestCustomPlacedTile(tiletemplates.TestOnlyField()),
		test.GetTestCustomPlacedTile(tiletemplates.MonasteryWithoutRoads()),
	}

	// add meeple to the monastery
	tiles[1].Monastery().Meeple.PlayerID = 1
	tiles[1].Monastery().Meeple.Type = elements.NormalMeeple

	tiles[11].Monastery().Meeple.PlayerID = 2
	tiles[11].Monastery().Meeple.Type = elements.NormalMeeple

	// set positions
	tiles[0].Position = position.New(0, 1)
	tiles[1].Position = position.New(0, 2)
	tiles[2].Position = position.New(0, 3)

	tiles[3].Position = position.New(-1, 1)
	tiles[4].Position = position.New(-1, 2)
	tiles[5].Position = position.New(-1, 3)

	tiles[6].Position = position.New(1, 1)

	tiles[7].Position = position.New(2, 1)
	tiles[8].Position = position.New(2, 2)
	tiles[9].Position = position.New(2, 3)

	tiles[10].Position = position.New(1, 3)

	tiles[11].Position = position.New(1, 2)

	// place tiles
	for i, tile := range tiles[:len(tiles)-1] {
		_, err := board.PlaceTile(tile)
		if err != nil {
			t.Fatalf("error placing tile number: %#v: %#v", i, err)
		}

		report = board.ScoreMonasteries(tile, false)
		if !reflect.DeepEqual(report, elements.NewScoreReport()) {
			t.Fatalf("ScoreMonasteries failed on tile number: %#v. expected %#v, got %#v instead", i, elements.NewScoreReport(), report)
		}
	}

	// place the last tile
	_, err := board.PlaceTile(tiles[11])
	if err != nil {
		t.Fatalf("error placing tile number: %#v: %#v", 11, err)
	}
	report = board.ScoreMonasteries(tiles[11], false)
	expectedReport := elements.NewScoreReport()
	expectedReport.ReceivedPoints = map[elements.ID]uint32{
		1: 9,
		2: 9,
	}
	expectedReport.ReturnedMeeples = map[elements.ID][]elements.MeepleWithPosition{
		1: {elements.NewMeepleWithPosition(
			elements.Meeple{Type: elements.NormalMeeple, PlayerID: elements.ID(1)},
			position.New(0, 2),
		)},
		2: {elements.NewMeepleWithPosition(
			elements.Meeple{Type: elements.NormalMeeple, PlayerID: elements.ID(2)},
			position.New(1, 2),
		)},
	}
	if !reflect.DeepEqual(report, expectedReport) {
		t.Fatalf("ScoreMonasteries failed on tile number: %#v. expected:\n%#v,\ngot:\n%#v instead", 11, expectedReport, report)
	}
}
