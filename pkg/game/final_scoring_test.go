package game

import (
	"testing"

	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/deck"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/game/elements"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/game/position"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/stack"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tiles"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tiles/feature"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tiles/side"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tiles/tiletemplates"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tilesets"
)

/*
  All tests contains the board visualization. Each tile is represented by 5x5 ascii characters
*/

func makeTurn(game *Game, t *testing.T, tilePosition position.Position, rotations uint, meeple elements.MeepleType, featureSide side.Side, featureType feature.Type) {
	tile, err := game.GetCurrentTile()
	if err != nil {
		t.Fatal(err.Error())
	}

	var player = game.CurrentPlayer()

	ptile := elements.ToPlacedTile(tile.Rotate(rotations))
	ptile.Position = tilePosition
	if meeple != elements.NoneMeeple {
		ptile.GetPlacedFeatureAtSide(featureSide, featureType).Meeple = elements.Meeple{
			Type:     meeple,
			PlayerID: player.ID(),
		}
	}

	err = game.PlayTurn(ptile)

	if err != nil {
		t.Fatal(err.Error())
	}
}

/*
|                 0    1

|               |   |.....
|               .\ /......
|0              --0----1-!
|               ..........
|               ..........
*/
func TestFinalScoreRoad(t *testing.T) {
	// ------ create tileset --------
	var tiles []tiles.Tile
	var err error
	tiles = append(tiles, tiletemplates.StraightRoads())

	tileset := tilesets.TileSet{
		StartingTile: tiletemplates.SingleCityEdgeStraightRoads(),
		Tiles:        tiles,
	}

	// ------ create game --------
	deckStack := stack.NewOrdered(tileset.Tiles)
	deck := deck.Deck{Stack: &deckStack, StartingTile: tileset.StartingTile}

	game, err := NewFromDeck(deck, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	makeTurn(game, t, position.New(1, 0), 0, elements.NormalMeeple, side.Right, feature.Road)

	scores, err := game.Finalize()
	if err != nil {
		t.Fatal(err.Error())
	}
	var expectedScores = []uint32{2, 0}

	for i := range 2 {
		if scores.ReceivedPoints[elements.ID(i+1)] != expectedScores[i] {
			t.Fatalf("Player %d final score incorrect. Expected %d, got: %d", i+1, expectedScores[i], scores.ReceivedPoints[elements.ID(i+1)])
		}
	}
}

/*
|                 0    1

|               |   |.....
|               .\ /......
|0              --0----1-!
|               ..........
|               ..........
|               ..........
|               ..........
|-1             --2-@--3--
|               ..........
|               ..........
|               .....
|               .....
|-2             --4-@
|               .....
|               .....
*/
func TestFinalScoreMultipleRoads(t *testing.T) {
	// ------ create tileset --------
	var tiles []tiles.Tile
	var err error
	for range 4 {
		tiles = append(tiles, tiletemplates.StraightRoads())
	}

	tileset := tilesets.TileSet{
		StartingTile: tiletemplates.SingleCityEdgeStraightRoads(),
		Tiles:        tiles,
	}

	// ------ create game --------
	deckStack := stack.NewOrdered(tileset.Tiles)
	deck := deck.Deck{Stack: &deckStack, StartingTile: tileset.StartingTile}

	game, err := NewFromDeck(deck, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	makeTurn(game, t, position.New(1, 0), 0, elements.NormalMeeple, side.Right, feature.Road)
	makeTurn(game, t, position.New(0, -1), 0, elements.NormalMeeple, side.Right, feature.Road)
	makeTurn(game, t, position.New(1, -1), 0, elements.NoneMeeple, side.NoSide, feature.NoneType)
	makeTurn(game, t, position.New(0, -2), 0, elements.NormalMeeple, side.Right, feature.Road)

	scores, err := game.Finalize()
	if err != nil {
		t.Fatal(err.Error())
	}
	var expectedScores = []uint32{2, 3}

	for i := range 2 {
		if scores.ReceivedPoints[elements.ID(i+1)] != expectedScores[i] {
			t.Fatalf("Player %d final score incorrect. Expected %d, got: %d", i+1, expectedScores[i], scores.ReceivedPoints[elements.ID(i+1)])
		}
	}
}

func TestFinalScoreMultipleRoads123(t *testing.T) {
	// ------ create tileset --------
	var tiles []tiles.Tile
	var err error
	tiles = append(tiles, tiletemplates.TCrossRoad())
	tiles = append(tiles, tiletemplates.RoadsTurn())
	tiles = append(tiles, tiletemplates.RoadsTurn())
	tiles = append(tiles, tiletemplates.RoadsTurn())

	tileset := tilesets.TileSet{
		StartingTile: tiletemplates.SingleCityEdgeStraightRoads(),
		Tiles:        tiles,
	}

	// ------ create game --------
	deckStack := stack.NewOrdered(tileset.Tiles)
	deck := deck.Deck{Stack: &deckStack, StartingTile: tileset.StartingTile}

	game, err := NewFromDeck(deck, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	makeTurn(game, t, position.New(0, -1), 0, elements.NormalMeeple, side.Right, feature.Road)
	makeTurn(game, t, position.New(1, -1), 0, elements.NoneMeeple, side.NoSide, feature.NoneType)
	makeTurn(game, t, position.New(1, -2), 1, elements.NoneMeeple, side.NoSide, feature.NoneType)
	makeTurn(game, t, position.New(0, -2), 2, elements.NoneMeeple, side.NoSide, feature.NoneType)

	scores, err := game.Finalize()
	if err != nil {
		t.Fatal(err.Error())
	}
	var expectedScores = []uint32{4, 0}

	for i := range 2 {
		if scores.ReceivedPoints[elements.ID(i+1)] != expectedScores[i] {
			t.Fatalf("Player %d final score incorrect. Expected %d, got: %d", i+1, expectedScores[i], scores.ReceivedPoints[elements.ID(i+1)])
		}
	}
}

/*
|                 0    1

|               |   |.....
|               .\ /..[!].
|0              --0---[1].
|               ......[ ].
|               ..........
*/
func TestFinalScoreMonastery(t *testing.T) {
	// ------ create tileset --------
	var tiles []tiles.Tile
	var err error
	tiles = append(tiles, tiletemplates.MonasteryWithSingleRoad())

	tileset := tilesets.TileSet{
		StartingTile: tiletemplates.SingleCityEdgeStraightRoads(),
		Tiles:        tiles,
	}

	// ------ create game --------
	deckStack := stack.NewOrdered(tileset.Tiles)
	deck := deck.Deck{Stack: &deckStack, StartingTile: tileset.StartingTile}

	game, err := NewFromDeck(deck, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	makeTurn(game, t, position.New(1, 0), 1, elements.NormalMeeple, side.NoSide, feature.Monastery)

	scores, err := game.Finalize()
	if err != nil {
		t.Fatal(err.Error())
	}
	var expectedScores = []uint32{2, 0}

	for i := range 2 {
		if scores.ReceivedPoints[elements.ID(i+1)] != expectedScores[i] {
			t.Fatalf("Player %d final score incorrect. Expected %d, got: %d", i+1, expectedScores[i], scores.ReceivedPoints[elements.ID(i+1)])
		}
	}
}

/*
|                 0    1

|               |   |.....
|               .\ /..[!].
|0              --0---[1].
|               ......[ ].
|               ..........
|               ..........
|               .[@]..[!].
|-1             .[2]..[3].
|               .[ ]..[ ].
|               ..........
*/
func TestFinalScoreMonasteries(t *testing.T) {
	// ------ create tileset --------
	var tiles []tiles.Tile
	var err error
	tiles = append(tiles, tiletemplates.MonasteryWithSingleRoad())
	tiles = append(tiles, tiletemplates.MonasteryWithoutRoads())
	tiles = append(tiles, tiletemplates.MonasteryWithoutRoads())

	tileset := tilesets.TileSet{
		StartingTile: tiletemplates.SingleCityEdgeStraightRoads(),
		Tiles:        tiles,
	}

	// ------ create game --------
	deckStack := stack.NewOrdered(tileset.Tiles)
	deck := deck.Deck{Stack: &deckStack, StartingTile: tileset.StartingTile}

	game, err := NewFromDeck(deck, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	makeTurn(game, t, position.New(1, 0), 1, elements.NormalMeeple, side.NoSide, feature.Monastery)
	makeTurn(game, t, position.New(0, -1), 0, elements.NormalMeeple, side.NoSide, feature.Monastery)
	makeTurn(game, t, position.New(1, -1), 0, elements.NormalMeeple, side.NoSide, feature.Monastery)

	scores, err := game.Finalize()
	if err != nil {
		t.Fatal(err.Error())
	}
	var expectedScores = []uint32{8, 4}

	for i := range 2 {
		if scores.ReceivedPoints[elements.ID(i+1)] != expectedScores[i] {
			t.Fatalf("Player %d final score incorrect. Expected %d, got: %d", i+1, expectedScores[i], scores.ReceivedPoints[elements.ID(i+1)])
		}
	}
}

/*
|                 0    1

|               .| |.
|               ./!\.
|1              | 1 |
|               |   |
|               |   |
|               |   |
|               .\ /.
|0              --0--
|               .....
|               .....
*/
func TestFinalScoreCity(t *testing.T) {
	// ------ create tileset --------
	var tiles []tiles.Tile
	var err error
	tiles = append(tiles, tiletemplates.TwoCityEdgesUpAndDownConnected())

	tileset := tilesets.TileSet{
		StartingTile: tiletemplates.SingleCityEdgeStraightRoads(),
		Tiles:        tiles,
	}

	// ------ create game --------
	deckStack := stack.NewOrdered(tileset.Tiles)
	deck := deck.Deck{Stack: &deckStack, StartingTile: tileset.StartingTile}

	game, err := NewFromDeck(deck, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	makeTurn(game, t, position.New(0, 1), 0, elements.NormalMeeple, side.Top, feature.City)

	scores, err := game.Finalize()
	if err != nil {
		t.Fatal(err.Error())
	}
	var expectedScores = []uint32{2, 0}

	for i := range 2 {
		if scores.ReceivedPoints[elements.ID(i+1)] != expectedScores[i] {
			t.Fatalf("Player %d final score incorrect. Expected %d, got: %d", i+1, expectedScores[i], scores.ReceivedPoints[elements.ID(i+1)])
		}
	}
}

/*
|                 0    1

|          | ! ||   || @ |
|          .\ /.| ! |.\ /.
|1         ..3..| 1 |..2..
|          .....|   |......
|          .....|   |.....
|               |   |
|               .\ /.
|0              --0--
|               .....
|               .....
*/

func TestFinalScoreCities(t *testing.T) {
	// ------ create tileset --------
	var tiles []tiles.Tile
	var err error
	tiles = append(tiles, tiletemplates.TwoCityEdgesUpAndDownConnected())
	tiles = append(tiles, tiletemplates.SingleCityEdgeNoRoads())
	tiles = append(tiles, tiletemplates.SingleCityEdgeNoRoads())

	tileset := tilesets.TileSet{
		StartingTile: tiletemplates.SingleCityEdgeStraightRoads(),
		Tiles:        tiles,
	}

	// ------ create game --------
	deckStack := stack.NewOrdered(tileset.Tiles)
	deck := deck.Deck{Stack: &deckStack, StartingTile: tileset.StartingTile}

	game, err := NewFromDeck(deck, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	makeTurn(game, t, position.New(0, 1), 0, elements.NormalMeeple, side.Top, feature.City)
	makeTurn(game, t, position.New(1, 1), 0, elements.NormalMeeple, side.Top, feature.City)
	makeTurn(game, t, position.New(-1, 1), 0, elements.NormalMeeple, side.Top, feature.City)

	scores, err := game.Finalize()
	if err != nil {
		t.Fatal(err.Error())
	}
	var expectedScores = []uint32{3, 1}

	for i := range 2 {
		if scores.ReceivedPoints[elements.ID(i+1)] != expectedScores[i] {
			t.Fatalf("Player %d final score incorrect. Expected %d, got: %d", i+1, expectedScores[i], scores.ReceivedPoints[elements.ID(i+1)])
		}
	}
}

/*
|                 0    1

|               ..!..
|               .....
|1              ..1..
|               ./ \.
|               |   |
|               |   |
|               .\ /.
|0              --0--
|               .....
|               .....
*/
func TestFinalScoreField(t *testing.T) {
	// ------ create tileset --------
	var tiles []tiles.Tile
	var err error
	tiles = append(tiles, tiletemplates.SingleCityEdgeNoRoads())

	tileset := tilesets.TileSet{
		StartingTile: tiletemplates.SingleCityEdgeStraightRoads(),
		Tiles:        tiles,
	}

	// ------ create game --------
	deckStack := stack.NewOrdered(tileset.Tiles)
	deck := deck.Deck{Stack: &deckStack, StartingTile: tileset.StartingTile}

	game, err := NewFromDeck(deck, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	makeTurn(game, t, position.New(0, 1), 2, elements.NormalMeeple, side.Top, feature.Field)

	scores, err := game.Finalize()
	if err != nil {
		t.Fatal(err.Error())
	}
	var expectedScores = []uint32{3, 0}

	for i := range 2 {
		if scores.ReceivedPoints[elements.ID(i+1)] != expectedScores[i] {
			t.Fatalf("Player %d final score incorrect. Expected %d, got: %d", i+1, expectedScores[i], scores.ReceivedPoints[elements.ID(i+1)])
		}
	}
}

/*
|                 0    1

|               ..!.......
|               ......[ ].
|1              ..1...[5]-
|               ./ \..[ ].
|               |   |.....
|               |   |.....
|               .\ /..[ ].
|0              --0---[4].
|               ......[ ].
|               ..........
|               ..@..
|               .....
|-1             ..2..
|               ./ \.
|               |   |
|               |   |
|               .\ /.
|-2             ..3..
|               .....
|               .....

	Testing multiple meeples on same field
*/
func TestFinalScoreFields1(t *testing.T) {
	// ------ create tileset --------
	var tiles []tiles.Tile
	var err error
	tiles = append(tiles, tiletemplates.SingleCityEdgeNoRoads())
	tiles = append(tiles, tiletemplates.SingleCityEdgeNoRoads())
	tiles = append(tiles, tiletemplates.SingleCityEdgeNoRoads())
	tiles = append(tiles, tiletemplates.MonasteryWithSingleRoad())
	tiles = append(tiles, tiletemplates.MonasteryWithSingleRoad())

	tileset := tilesets.TileSet{
		StartingTile: tiletemplates.SingleCityEdgeStraightRoads(),
		Tiles:        tiles,
	}

	// ------ create game --------
	deckStack := stack.NewOrdered(tileset.Tiles)
	deck := deck.Deck{Stack: &deckStack, StartingTile: tileset.StartingTile}

	game, err := NewFromDeck(deck, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	makeTurn(game, t, position.New(0, 1), 2, elements.NormalMeeple, side.Top, feature.Field)
	makeTurn(game, t, position.New(0, -1), 2, elements.NormalMeeple, side.Top, feature.Field)
	makeTurn(game, t, position.New(0, -2), 0, elements.NoneMeeple, side.NoSide, feature.NoneType)
	makeTurn(game, t, position.New(1, 0), 1, elements.NoneMeeple, side.NoSide, feature.NoneType)
	makeTurn(game, t, position.New(1, 1), 3, elements.NoneMeeple, side.NoSide, feature.NoneType)

	scores, err := game.Finalize()
	if err != nil {
		t.Fatal(err.Error())
	}
	var expectedScores = []uint32{6, 6}

	for i := range 2 {
		if scores.ReceivedPoints[elements.ID(i+1)] != expectedScores[i] {
			t.Fatalf("Player %d final score incorrect. Expected %d, got: %d", i+1, expectedScores[i], scores.ReceivedPoints[elements.ID(i+1)])
		}
	}
}

/*
|                 0

|               ...@.
|               .....
|2              ..2..
|               ./ \.
|               |   |
|               |   |
|               .\ /.
|1              ..1.!
|               ./ \.
|               |   |
|               |   |
|               .\ /.
|0              --0--
|               .....
|               .....

	Testing multiple meeples on same field
*/
func TestFinalScoreFields2(t *testing.T) {
	// ------ create tileset --------
	var tiles []tiles.Tile
	var err error
	tiles = append(tiles, tiletemplates.TwoCityEdgesUpAndDownNotConnected())
	tiles = append(tiles, tiletemplates.SingleCityEdgeNoRoads())

	tileset := tilesets.TileSet{
		StartingTile: tiletemplates.SingleCityEdgeStraightRoads(),
		Tiles:        tiles,
	}

	// ------ create game --------
	deckStack := stack.NewOrdered(tileset.Tiles)
	deck := deck.Deck{Stack: &deckStack, StartingTile: tileset.StartingTile}

	game, err := NewFromDeck(deck, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	makeTurn(game, t, position.New(0, 1), 2, elements.NormalMeeple, side.Right, feature.Field)
	makeTurn(game, t, position.New(0, 2), 2, elements.NormalMeeple, side.Top, feature.Field)

	scores, err := game.Finalize()
	if err != nil {
		t.Fatal(err.Error())
	}
	var expectedScores = []uint32{6, 3}

	for i := range 2 {
		if scores.ReceivedPoints[elements.ID(i+1)] != expectedScores[i] {
			t.Fatalf("Player %d final score incorrect. Expected %d, got: %d", i+1, expectedScores[i], scores.ReceivedPoints[elements.ID(i+1)])
		}
	}
}

/*
|      -2   -1    0    1    2

|     .........................
|     .[ ].................[ ].
|2    .[K]...6....5....7...[H].
|     .[ ]../ \../ \../ \..[ ].
|     .....|   ||   ||   |.....
|     ....-    ||   ||   |-....
|     .../    /..\ /.\     \...
|1    ..G    X....1....X    9..
|     ...\  /...........\  /...
|     ....--.............--....
|     ....--.............--....
|     .../  \....[ ]..../  \...
|0    ..E    4...[0]...2    8..
|     ...\  /....[ ]....\  /...
|     ....--.............--....
|     ....--......!......--....
|     .../  \.........../  \...
|-1   ..F    X....3....X    A..
|     ...\    \../ \../    /...
|     ....-    ||   ||    -....
|     .....|   ||   ||   |...@.
|     .[ ]..\ /..\ /..\ /..[ ].
|-2   .[J]...D....B....C...[I].
|     .[ ].................[ ].
|     .........................

	Field surrounded be cities! 8 cities in total.
	For nice loops starting tile is changed for monastery field.
	One meeple inside(!) and one meeple outside (@)
*/
func TestFinalScoreFields3(t *testing.T) {
	// ------ create tileset --------
	var tiles []tiles.Tile
	var err error
	for range 4 + 4*3 { // inner and outer single edges
		tiles = append(tiles, tiletemplates.SingleCityEdgeNoRoads())
	}

	for range 4 { // inner corners cities
		tiles = append(tiles, tiletemplates.TwoCityEdgesCornerConnected())
	}

	for range 4 { // outer corner monasteries
		tiles = append(tiles, tiletemplates.MonasteryWithoutRoads())
	}

	tileset := tilesets.TileSet{
		StartingTile: tiletemplates.MonasteryWithoutRoads(),
		Tiles:        tiles,
	}

	// ------ create game --------
	deckStack := stack.NewOrdered(tileset.Tiles)
	deck := deck.Deck{Stack: &deckStack, StartingTile: tileset.StartingTile}

	game, err := NewFromDeck(deck, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	// ------ make turns --------
	rotations := []uint{0, 1, 2, 3}
	offsets := []position.Position{position.New(0, 0), position.New(-1, 0), position.New(1, 0)}

	// start with inner single city edges (starting from top) (tiles 1-4)
	for _, rotation := range rotations {
		if rotation == 2 {
			makeTurn(game, t, position.New(0, 1).Rotate(rotation), rotation, elements.NormalMeeple, side.Top, feature.Field)
		} else {
			makeTurn(game, t, position.New(0, 1).Rotate(rotation), rotation, elements.NoneMeeple, side.NoSide, feature.NoneType)
		}
	}

	// Now outer single city edges (starting from top) (tiles 5-G)
	for _, rotation := range rotations {
		for _, offset := range offsets {
			makeTurn(game, t, position.New(0, 2).Rotate(rotation).Add(offset.Rotate(rotation)), rotation+2, elements.NoneMeeple, side.NoSide, feature.NoneType)
		}
	}

	// Inner corners
	for _, rotation := range rotations {
		makeTurn(game, t, position.New(1, 1).Rotate(rotation), rotation, elements.NoneMeeple, side.NoSide, feature.NoneType)
	}

	// Outer corners
	for _, rotation := range rotations {
		if rotation == 1 {
			makeTurn(game, t, position.New(2, 2).Rotate(rotation), 0, elements.NormalMeeple, side.Top, feature.Field)
		} else {
			makeTurn(game, t, position.New(2, 2).Rotate(rotation), 0, elements.NoneMeeple, side.NoSide, feature.NoneType)
		}
	}

	// ------ Finalize --------
	scores, err := game.Finalize()
	if err != nil {
		t.Fatal(err.Error())
	}
	var expectedScores = []uint32{24, 24}

	for i := range 2 {
		if scores.ReceivedPoints[elements.ID(i+1)] != expectedScores[i] {
			t.Fatalf("Player %d final score incorrect. Expected %d, got: %d", i+1, expectedScores[i], scores.ReceivedPoints[elements.ID(i+1)])
		}
	}
}
