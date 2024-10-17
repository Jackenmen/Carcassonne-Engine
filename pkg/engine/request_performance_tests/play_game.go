package requestperformancetests

import (
	"testing"

	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/engine"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tilesets"
)

type TestedRequest func(games *[]engine.SerializedGameWithID, eng *engine.GameEngine, b *testing.B)

func PlayGame(gameCount int, testedRequest TestedRequest, b *testing.B) {
	b.StopTimer()

	eng, err := engine.StartGameEngine(4, "")
	if err != nil {
		b.Fatal()
	}

	var games = []engine.SerializedGameWithID{}
	for seed := range gameCount {
		game, err := eng.GenerateSeededGame(tilesets.StandardTileSet(), int64(seed+1000))
		if err != nil {
			b.Fatal()
		}
		games = append(games, game)
	}

	// for each turn
	for range len(tilesets.StandardTileSet().Tiles) {

		// at the start before making any turn
		// test desired requests
		for range b.N {
			testedRequest(&games, eng, b)
		}

		// get moves
		legalMovesRequests := []*engine.GetLegalMovesRequest{}
		for _, game := range games {
			legalMovesRequests = append(legalMovesRequests,
				&engine.GetLegalMovesRequest{
					BaseGameID:  game.ID,
					TileToPlace: game.Game.CurrentTile,
				},
			)
		}
		legalMoves := eng.SendGetLegalMovesBatch(legalMovesRequests)
		for i, legalMove := range legalMoves {
			if legalMove.Err() != nil {
				b.Fatalf("%#v legal move failed. Reason: %#v", i, legalMove.Err().Error())
			}
		}

		// make move
		makeTurnRequests := []*engine.PlayTurnRequest{}
		for i := range gameCount {
			makeTurnRequests = append(makeTurnRequests,
				&engine.PlayTurnRequest{
					GameID: games[i].ID,
					Move:   legalMoves[i].Moves[0].Move,
				},
			)
		}
		playTurnResp := eng.SendPlayTurnBatch(makeTurnRequests)

		// update games
		for i := range gameCount {
			if playTurnResp[i].Err() != nil {
				b.Fatalf("%#v play turn failed. Reason: %#v", i, playTurnResp[i].Err().Error())
			}
			games[i].Game = playTurnResp[i].Game
		}
	}
	eng.Close()
}
