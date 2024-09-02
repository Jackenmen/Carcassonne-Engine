package game

import (
	"reflect"
	"testing"

	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/game/elements"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/game/field"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/game/position"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tiles/feature"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tiles/side"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tiles/tiletemplates"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tilesets"
)

func TestScoreFieldOnePlayerGetsPoints(t *testing.T) {
	/*
		the board setup is as follows:
		..C..
		─┼SM·
		··┌─┐
		··M··

		· - empty
		M - monastery with a single road
		S - starting tile
		C - city closing the starting tile
		┌, ─, ┐, ┼ - roads

		The meeple is placed on the field feature in the higher monastery tile (position: 1,0).
	*/

	boardInterface := NewBoard(tilesets.StandardTileSet())
	board := boardInterface.(*board)

	tiles := []elements.PlacedTile{
		elements.ToPlacedTile(tiletemplates.MonasteryWithSingleRoad().Rotate(1)),
		elements.ToPlacedTile(tiletemplates.XCrossRoad()),
		elements.ToPlacedTile(tiletemplates.StraightRoads()),
		elements.ToPlacedTile(tiletemplates.StraightRoads()),
		elements.ToPlacedTile(tiletemplates.RoadsTurn()),
		elements.ToPlacedTile(tiletemplates.RoadsTurn().Rotate(3)),
		elements.ToPlacedTile(tiletemplates.MonasteryWithSingleRoad().Rotate(2)),
		elements.ToPlacedTile(tiletemplates.SingleCityEdgeNoRoads().Rotate(2)),
	}

	// add meeple to the field
	tiles[0].GetPlacedFeatureAtSide(side.All, feature.Field).Meeple =
		elements.Meeple{PlayerID: 1, Type: elements.NormalMeeple}

	// set positions
	tiles[0].Position = position.New(1, 0)
	tiles[1].Position = position.New(-1, 0)
	tiles[2].Position = position.New(-2, 0)

	tiles[3].Position = position.New(1, -1)
	tiles[4].Position = position.New(2, -1)
	tiles[5].Position = position.New(0, -1)

	tiles[6].Position = position.New(0, -2)

	tiles[7].Position = position.New(0, 1)

	// place tiles
	for i, tile := range tiles {
		_, err := board.PlaceTile(tile)
		if err != nil {
			t.Fatalf("error placing tile number: %#v: %#v", i, err)
		}
	}

	// test field.Expand()
	field := field.New(*tiles[0].GetPlacedFeatureAtSide(side.All, feature.Field), tiles[0].Position)
	field.Expand(board, board.cityManager)

	if field.FeaturesCount() != 12 {
		t.Fatalf("expected %#v, got %#v instead", 12, field.FeaturesCount())
	}

	if field.CitiesCount() != 1 {
		t.Fatalf("expected %#v, got %#v instead", 1, field.CitiesCount())
	}

	// test field.GetScoreReport()
	expectedReport := elements.NewScoreReport()
	expectedReport.ReceivedPoints = map[elements.ID]uint32{
		1: 3,
	}
	expectedReport.ReturnedMeeples = map[elements.ID][]elements.MeepleWithPosition{
		1: {elements.NewMeepleWithPosition(
			elements.Meeple{Type: elements.NormalMeeple, PlayerID: elements.ID(1)},
			position.New(1, 0))},
	}

	actualReport := field.GetScoreReport()

	if !reflect.DeepEqual(expectedReport, actualReport) {
		t.Fatalf("expected %#v, got %#v instead", expectedReport, actualReport)
	}
}

func TestScoreFieldTwoPlayersGetPoints(t *testing.T) {
	/*
		the board setup is as follows:
		..C..
		─┼SM·C
		··┌─┐╚
		·cM··

		· - empty
		M - monastery with a single road
		S - starting tile
		C - single city edge, closing the adjacent city
		c - single city edge, not closed
		╚ - tile with two city features, on top and right sides
		┌, ─, ┐, ┼ - roads

		(two cities are closed - the starting one and the top part of the city that has two city features on one tile)
		(two cities are not clsed - the city on the left of the bottom monastery and the right part of the city that has two city features on one tile)

		The first meeple is placed on the field feature in the higher monastery tile (position: 1,0).
		The second meeple is placed on the bottom-right corner part of the ┌ road below the starting tile (position: 0,-1).
	*/

	boardInterface := NewBoard(tilesets.StandardTileSet())
	board := boardInterface.(*board)

	tiles := []elements.PlacedTile{
		elements.ToPlacedTile(tiletemplates.MonasteryWithSingleRoad().Rotate(1)),
		elements.ToPlacedTile(tiletemplates.XCrossRoad()),
		elements.ToPlacedTile(tiletemplates.StraightRoads()),
		elements.ToPlacedTile(tiletemplates.StraightRoads()),
		elements.ToPlacedTile(tiletemplates.RoadsTurn()),
		elements.ToPlacedTile(tiletemplates.RoadsTurn().Rotate(3)),
		elements.ToPlacedTile(tiletemplates.MonasteryWithSingleRoad().Rotate(2)),
		elements.ToPlacedTile(tiletemplates.TwoCityEdgesCornerNotConnected()),
		elements.ToPlacedTile(tiletemplates.SingleCityEdgeNoRoads().Rotate(2)),
		elements.ToPlacedTile(tiletemplates.SingleCityEdgeNoRoads().Rotate(2)),
		elements.ToPlacedTile(tiletemplates.SingleCityEdgeNoRoads().Rotate(3)),
	}

	// add meeple to the fields
	tiles[0].GetPlacedFeatureAtSide(side.All, feature.Field).Meeple =
		elements.Meeple{PlayerID: 1, Type: elements.NormalMeeple}
	tiles[5].GetPlacedFeatureAtSide(side.BottomRightEdge|side.RightBottomEdge, feature.Field).Meeple =
		elements.Meeple{PlayerID: 2, Type: elements.NormalMeeple}

	// set positions
	tiles[0].Position = position.New(1, 0)
	tiles[1].Position = position.New(-1, 0)
	tiles[2].Position = position.New(-2, 0)

	tiles[3].Position = position.New(1, -1)
	tiles[4].Position = position.New(2, -1)
	tiles[5].Position = position.New(0, -1)

	tiles[6].Position = position.New(0, -2)

	tiles[7].Position = position.New(3, -1)
	tiles[8].Position = position.New(0, 1)
	tiles[9].Position = position.New(3, 0)
	tiles[10].Position = position.New(-1, -2)

	// place tiles
	for i, tile := range tiles {
		_, err := board.PlaceTile(tile)
		if err != nil {
			t.Fatalf("error placing tile number: %#v: %#v", i, err)
		}
	}

	// test field.Expand()
	field := field.New(*tiles[0].GetPlacedFeatureAtSide(side.All, feature.Field), tiles[0].Position)
	field.Expand(board, board.cityManager)

	if field.FeaturesCount() != 14 {
		t.Fatalf("expected %#v, got %#v instead", 14, field.FeaturesCount())
	}

	if field.CitiesCount() != 2 {
		t.Fatalf("expected %#v, got %#v instead", 3, field.CitiesCount())
	}

	// test field.GetScoreReport()
	expectedReport := elements.NewScoreReport()
	expectedReport.ReceivedPoints = map[elements.ID]uint32{
		1: 6,
		2: 6,
	}
	expectedReport.ReturnedMeeples = map[elements.ID][]elements.MeepleWithPosition{
		1: {elements.NewMeepleWithPosition(
			elements.Meeple{Type: elements.NormalMeeple, PlayerID: elements.ID(1)},
			position.New(1, 0))},
		2: {elements.NewMeepleWithPosition(
			elements.Meeple{Type: elements.NormalMeeple, PlayerID: elements.ID(2)},
			position.New(0, -1))},
	}
	actualReport := field.GetScoreReport()

	if !reflect.DeepEqual(expectedReport, actualReport) {
		t.Fatalf("expected %#v, got %#v instead", expectedReport, actualReport)
	}
}

func TestRegressionGHIssue76(t *testing.T) {
	boardInterface := NewBoard(tilesets.StandardTileSet())
	board := boardInterface.(*board)

	tiles := []elements.PlacedTile{
		elements.ToPlacedTile(tiletemplates.ThreeCityEdgesConnectedRoadShield().Rotate(1)),
		elements.ToPlacedTile(tiletemplates.ThreeCityEdgesConnected()),
		elements.ToPlacedTile(tiletemplates.FourCityEdgesConnectedShield()),
		elements.ToPlacedTile(tiletemplates.TwoCityEdgesCornerNotConnected().Rotate(1)),
	}

	tiles[0].Position = position.New(1, 0)
	tiles[1].Position = position.New(2, 0)
	tiles[2].Position = position.New(2, 1)
	tiles[3].Position = position.New(1, 1)

	// place tiles
	for i, tile := range tiles {
		_, err := board.PlaceTile(tile)
		if err != nil {
			t.Fatalf("error placing tile number: %#v: %#v", i, err)
		}
	}

	field := field.New(
		*tiles[3].GetPlacedFeatureAtSide(
			side.Top|side.Left,
			feature.Field,
		),
		tiles[3].Position,
	)
	field.Expand(board, board.cityManager)
}
