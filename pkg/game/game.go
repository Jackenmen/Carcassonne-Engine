package game

import (
	"errors"
	"fmt"
	"io"

	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/deck"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/game/elements"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/logger"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/player"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/stack"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tiles"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tilesets"
)

var ErrCannotSwapTiles = errors.New(
	"swapping tiles is only allowed in game clones created with DeepCloneWithSwappableTiles()",
)

type SerializedGame struct {
	CurrentTile         tiles.Tile
	ValidTilePlacements []elements.PlacedTile
	CurrentPlayerID     elements.ID
	Players             []elements.Player
	PlayerCount         int
	Tiles               []elements.PlacedTile
	TileSet             tilesets.TileSet
}

type Game struct {
	board   elements.Board
	deck    deck.Deck
	players []elements.Player
	// index in the `players` field, not the Player ID
	currentPlayer int
	log           logger.Logger
	canSwapTiles  bool
}

func NewFromTileSet(tileSet tilesets.TileSet, log logger.Logger) (*Game, error) {
	deckStack := stack.New(tileSet.Tiles)
	deck := deck.Deck{
		Stack:        &deckStack,
		StartingTile: tileSet.StartingTile,
	}
	return NewFromDeck(deck, log)
}

func NewFromDeck(
	deck deck.Deck, log logger.Logger,
) (*Game, error) {
	if log == nil {
		nullLogger := logger.New(io.Discard)
		log = &nullLogger
	}
	game := &Game{
		board:         NewBoard(deck.TileSet()),
		deck:          deck,
		players:       []elements.Player{player.New(1), player.New(2)},
		currentPlayer: 0,
		log:           log,
	}

	// All tiles in base game can be placed on the first move but let's just check this
	// in case this isn't true for tiles from all of the expansions.
	err := game.ensureCurrentTileHasValidPlacement()
	if err != nil {
		return nil, err
	}
	if err := log.LogEvent(
		logger.StartEvent, logger.NewStartEntryContent(game.deck.StartingTile, game.deck.GetRemaining(), len(game.players)),
	); err != nil {
		return nil, err
	}

	return game, nil
}

func (game Game) DeepClone() *Game {
	game.board = game.board.DeepClone()
	game.deck = game.deck.DeepClone()

	players := make([]elements.Player, len(game.players))
	for i, player := range game.players {
		players[i] = player.DeepClone()
	}
	game.players = players

	nullLogger := logger.New(io.Discard)
	game.log = &nullLogger

	return &game
}

func (game *Game) DeepCloneWithSwappableTiles() *Game {
	clone := game.DeepClone()
	clone.canSwapTiles = true
	return clone
}

func (game *Game) DeepCloneWithLog(log logger.Logger) (*Game, error) {
	clone := game.DeepClone()
	if err := game.log.CopyTo(log); err != nil {
		return nil, err
	}
	clone.log = log
	return clone, nil
}

func (game *Game) Serialized() SerializedGame {
	serialized := SerializedGame{
		CurrentPlayerID: game.CurrentPlayer().ID(),
		Players:         game.players,
		PlayerCount:     game.PlayerCount(),
		Tiles:           game.board.Tiles(),
		TileSet:         game.deck.TileSet(),
	}

	// prevent leakage of future state of the CurrentTile
	if game.CanSwapTiles() {
		return serialized
	}

	if tile, err := game.GetCurrentTile(); err == nil {
		serialized.CurrentTile = tile
		serialized.ValidTilePlacements = game.board.GetTilePlacementsFor(tile)
	}
	return serialized
}

func (game *Game) CanSwapTiles() bool {
	return game.canSwapTiles
}

func (game *Game) GetCurrentTile() (tiles.Tile, error) {
	return game.deck.Peek()
}

func (game *Game) GetRemainingTiles() []tiles.Tile {
	return game.deck.GetRemaining()
}

func (game *Game) CurrentPlayer() elements.Player {
	return game.players[game.currentPlayer]
}

func (game *Game) PlayerCount() int {
	return len(game.players)
}

func (game *Game) GetTilePlacementsFor(tile tiles.Tile) []elements.PlacedTile {
	return game.board.GetTilePlacementsFor(tile)
}

func (game *Game) GetLegalMovesFor(placement elements.PlacedTile) []elements.PlacedTile {
	moves := []elements.PlacedTile{}
	player := game.CurrentPlayer()

outer:
	for _, move := range game.board.GetLegalMovesFor(placement) {
		for i, feat := range move.Features {
			if feat.Meeple.Type != elements.NoneMeeple {
				if player.MeepleCount(feat.Meeple.Type) == 0 {
					// filter out moves that the current player cannot perform
					continue outer
				}
				feat.Meeple.PlayerID = game.CurrentPlayer().ID()
				move.Features[i] = feat
			}
		}

		moves = append(moves, move)
	}

	return moves
}

func (game *Game) ensureCurrentTileHasValidPlacement() error {
	for {
		// Peek at the tile that will be returned by GetCurrentTile() next time
		// to see, if it can actually be placed anywhere.
		nextTile, err := game.deck.Peek()
		if err != nil {
			if errors.Is(err, stack.ErrStackOutOfBounds) {
				// out of bounds - not our concern at the end of a turn
				return nil
			}

			panic("Stack.Peek() returned error that we do not know how to handle")
		}

		if game.board.TileHasValidPlacement(nextTile) {
			break
		}

		if _, err := game.deck.Next(); err != nil {
			// We already peeked and checked for out of bounds so that's unexpected...
			return err
		}
	}

	return nil
}

func (game *Game) SwapCurrentTile(tile tiles.Tile) error {
	if !game.CanSwapTiles() {
		return ErrCannotSwapTiles
	}
	return game.deck.MoveToTop(tile)
}

func (game *Game) PlayTurn(move elements.PlacedTile) error {
	// This is guaranteed to return a tile that has at least one valid placement
	// or `OutOfBounds` error, if there's no tiles left in the deck and this turn
	// shouldn't be happening.
	currentTile, err := game.GetCurrentTile()
	if err != nil {
		return err
	}

	if !currentTile.Equals(elements.ToTile(move)) {
		return fmt.Errorf("%w: %#v", elements.ErrWrongTile, currentTile)
	}
	player := game.CurrentPlayer()
	// prevent reusing underlying Features slice
	move = move.DeepClone()

	// In the class diagram, the `scoreReport` would be returned by
	// separate `CheckCompleted()` method but it's been abstracted by PlaceTile instead.
	scoreReport, err := player.PlaceTile(game.board, move)
	if err != nil {
		return err
	}
	// if placing a tile hasn't failed, the board has already been modified
	// and we can update the current player as well
	game.currentPlayer = (game.currentPlayer + 1) % game.PlayerCount()

	if err = game.log.LogEvent(
		logger.PlaceTileEvent, logger.NewPlaceTileEntryContent(player.ID(), move),
	); err != nil {
		return err
	}

	// Score features and update meeple counts
	for playerID, receivedPoints := range scoreReport.ReceivedPoints {
		player := game.players[playerID-1]
		player.SetScore(player.Score() + receivedPoints)
	}

	for playerID, returnedMeeples := range scoreReport.ReturnedMeeples {
		player := game.players[playerID-1]
		for _, meeple := range returnedMeeples {
			// return meeple to player
			player.SetMeepleCount(
				meeple.Type,
				player.MeepleCount(meeple.Type)+1,
			)

			// remove meeple from board
			game.board.RemoveMeeple(meeple.Position)
		}
	}

	if !scoreReport.IsEmpty() {
		if err := game.log.LogEvent(
			logger.ScoreEvent, logger.NewScoreEntryContent(scoreReport),
		); err != nil {
			return err
		}
	}

	// Pop from the stack after the move.
	if _, err = game.deck.Next(); err != nil {
		return err
	}

	err = game.ensureCurrentTileHasValidPlacement()
	if err != nil {
		return err
	}

	return nil
}

func (game *Game) Finalize() (elements.ScoreReport, error) {
	playerScores := elements.NewScoreReport()

	if _, err := game.GetCurrentTile(); !errors.Is(err, stack.ErrStackOutOfBounds) {
		return playerScores, elements.ErrGameIsNotFinished
	}

	// load scores
	for _, player := range game.players {
		playerScores.ReceivedPoints[player.ID()] = player.Score()
	}

	// add final score report
	meeplesReport := game.board.ScoreMeeples(true)
	playerScores.Join(meeplesReport)

	if err := game.log.LogEvent(logger.ScoreEvent, logger.NewScoreEntryContent(playerScores)); err != nil {
		return playerScores, err
	}

	return playerScores, nil
}

/*
Calculate points as if game has just finished
*/
func (game *Game) GetMidGameScore() elements.ScoreReport {
	playerScores := elements.NewScoreReport()

	// load scores
	for _, player := range game.players {
		playerScores.ReceivedPoints[player.ID()] = player.Score()
	}

	// add final score report
	meeplesReport := game.board.ScoreMeeples(false)
	playerScores.Join(meeplesReport)

	return playerScores
}
