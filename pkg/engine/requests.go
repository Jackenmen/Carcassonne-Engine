package engine

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"sort"

	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/game"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/game/elements"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/logger"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/stack"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tiles"
)

// internal worker request used by GameEngine.CloneGame()
type cloneGameResponse struct {
	BaseResponse
	Clones []*game.Game
}
type cloneGameRequest struct {
	GameID      int
	ReservedIDs []int
	LogDir      string
}

func (req *cloneGameRequest) gameID() int {
	return req.GameID
}

func (req *cloneGameRequest) requiresWrite() bool {
	return false
}

func (req *cloneGameRequest) execute(g *game.Game) Response {
	count := len(req.ReservedIDs)
	clones := make([]*game.Game, count)
	resp := &cloneGameResponse{BaseResponse: BaseResponse{gameID: req.GameID}}

	if req.LogDir == "" {
		for i := range req.ReservedIDs {
			clones[i] = g.DeepClone()
		}
		resp.Clones = clones
		return resp
	}

	for i, id := range req.ReservedIDs {
		logFile := path.Join(req.LogDir, fmt.Sprintf("%v.jsonl", id))
		logger, err := logger.NewFromFile(logFile)
		if err != nil {
			resp.err = err
			return resp
		}

		clone, err := g.DeepCloneWithLog(&logger)
		if err != nil {
			resp.err = err
			return resp
		}

		clones[i] = clone
	}

	resp.Clones = clones
	return resp
}

type PlayTurnResponse struct {
	BaseResponse
	Game game.SerializedGame
}
type PlayTurnRequest struct {
	GameID int
	Move   elements.PlacedTile
}

func (resp *PlayTurnResponse) canRemoveGame() bool {
	err := resp.Err()
	return err != nil && errors.Is(err, stack.ErrStackOutOfBounds)
}

func (resp *PlayTurnResponse) canRemoveChildGames() bool {
	return resp.Err() == nil
}

func (req *PlayTurnRequest) gameID() int {
	return req.GameID
}

func (req *PlayTurnRequest) requiresWrite() bool {
	return true
}

func (req *PlayTurnRequest) execute(game *game.Game) Response {
	err := game.PlayTurn(req.Move)
	resp := &PlayTurnResponse{
		BaseResponse: BaseResponse{
			gameID: req.gameID(),
			err:    err,
		},
	}
	if err != nil {
		return resp
	}

	resp.Game = game.Serialized()
	return resp
}

// State of the game the request is made for.
// Currently, this is internally represented as a slice of moves to make on the base game
// which can be resolved to a Game by passing it that game.
// Eventually this should probably move to some kind of handles to cached state.
type GameState struct {
	serializedGame game.SerializedGame
	simulatedMoves []elements.PlacedTile
}

func (state *GameState) Serialized() game.SerializedGame {
	return state.serializedGame
}

func (state *GameState) resolve(baseGame *game.Game) (*game.Game, error) {
	if state == nil {
		return baseGame, nil
	}
	game := baseGame.DeepClone()
	for _, move := range state.simulatedMoves {
		if err := game.SwapCurrentTile(elements.ToTile(move)); err != nil {
			return nil, err
		}
		if err := game.PlayTurn(move); err != nil {
			return nil, err
		}
	}
	return game, nil
}

func (state *GameState) with(
	serializedGame game.SerializedGame,
	move elements.PlacedTile,
) *GameState {
	var simulatedMoves []elements.PlacedTile
	if state == nil {
		simulatedMoves = []elements.PlacedTile{move}
	} else {
		simulatedMoves = make([]elements.PlacedTile, 0, len(state.simulatedMoves)+1)
		simulatedMoves = append(simulatedMoves, state.simulatedMoves...)
		simulatedMoves = append(simulatedMoves, move)
	}
	return &GameState{
		serializedGame: serializedGame,
		simulatedMoves: simulatedMoves,
	}
}

// represents a tile and its probability to be drawn from the deck
type TileProbability struct {
	Tile        tiles.Tile
	Probability float32
}

// tiles have probabilities represented as a 32-bit float
type GetRemainingTilesResponse struct {
	BaseResponse
	TileProbabilities []TileProbability
}
type GetRemainingTilesRequest struct {
	BaseGameID   int
	StateToCheck *GameState
}

func (req *GetRemainingTilesRequest) gameID() int {
	return req.BaseGameID
}

func (req *GetRemainingTilesRequest) requiresWrite() bool {
	return false
}

func (req *GetRemainingTilesRequest) execute(baseGame *game.Game) Response {
	resp := &GetRemainingTilesResponse{BaseResponse: BaseResponse{gameID: req.gameID()}}
	game, err := req.StateToCheck.resolve(baseGame)
	if err != nil {
		resp.err = err
		return resp
	}

	remaining := game.GetRemainingTiles()
	total := float32(len(remaining))
	probabilities := []TileProbability{}
	for _, tile := range remaining {
		n := len(probabilities)
		i := sort.Search(n, func(i int) bool {
			if len(probabilities[i].Tile.Features) != len(tile.Features) {
				return len(probabilities[i].Tile.Features) > len(tile.Features)
			}
			for j := range tile.Features {
				a := probabilities[i].Tile.Features[j]
				b := tile.Features[j]
				if a.FeatureType != b.FeatureType {
					return a.FeatureType > b.FeatureType
				}
				if a.ModifierType != b.ModifierType {
					return a.ModifierType > b.ModifierType
				}
				if a.Sides != b.Sides {
					return a.Sides > b.Sides
				}
			}
			return true
		})
		if i < n && probabilities[i].Tile.ExactEquals(tile) {
			probabilities[i].Probability++
		} else {
			probabilities = slices.Insert(
				probabilities,
				i,
				TileProbability{Tile: tile, Probability: 1.0},
			)
		}
	}
	for i := range probabilities {
		probabilities[i].Probability /= total
	}

	resp.TileProbabilities = probabilities

	return resp
}

type MoveWithState struct {
	Move  elements.PlacedTile
	State *GameState
}

type GetLegalMovesResponse struct {
	BaseResponse
	Moves []MoveWithState
}
type GetLegalMovesRequest struct {
	BaseGameID   int
	StateToCheck *GameState
	TileToPlace  tiles.Tile
}

func (req *GetLegalMovesRequest) gameID() int {
	return req.BaseGameID
}

func (req *GetLegalMovesRequest) requiresWrite() bool {
	return false
}

func (req *GetLegalMovesRequest) execute(baseGame *game.Game) Response {
	resp := &GetLegalMovesResponse{BaseResponse: BaseResponse{gameID: req.gameID()}}
	baseGame, err := req.StateToCheck.resolve(baseGame)
	if err != nil {
		resp.err = err
		return resp
	}

	placements := baseGame.GetTilePlacementsFor(req.TileToPlace)
	resp.Moves = []MoveWithState{}
	for _, placement := range placements {
		for _, move := range baseGame.GetLegalMovesFor(placement) {
			game := baseGame.DeepClone()
			if err := game.SwapCurrentTile(elements.ToTile(move)); err != nil {
				resp.err = err
				return resp
			}
			if err := game.PlayTurn(move); err != nil {
				resp.err = err
				return resp
			}
			moveState := MoveWithState{
				Move:  move,
				State: req.StateToCheck.with(game.Serialized(), move),
			}
			resp.Moves = append(resp.Moves, moveState)
		}
	}

	return resp
}
