package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/engine"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/game"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tilesets"
)

const (
	CONTEXT_BREADCRUMBS          = "breadcrumbs"
	CONTEXT_BREADCRUMBS_MESSAGES = "messages"
	CONTEXT_GAME_INFO            = "gameInfo"
	CONTEXT_GAME_INFO_ID         = "gameID"
	CONTEXT_GAME_INFO_SEED       = "seed"
)

type ExpectedNoTileErr struct {
	Game game.SerializedGame
}

func (err *ExpectedNoTileErr) Error() string {
	return fmt.Sprintf(
		"expected current tile to be nil, got %#v instead", err.Game.CurrentTile,
	)
}

type Breadcrumbs struct {
	scope    *sentry.Scope
	messages []string
}

func (breadcrumbs *Breadcrumbs) Add(msg string) {
	breadcrumbs.messages = append(breadcrumbs.messages, msg)
	breadcrumbs.scope.SetContext(CONTEXT_BREADCRUMBS, map[string]interface{}{
		CONTEXT_BREADCRUMBS_MESSAGES: breadcrumbs.messages,
	})
	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "game_engine",
		Message:  msg,
		Level:    sentry.LevelInfo,
	})
}

func singleGameEngineRun(scope *sentry.Scope, eng *engine.GameEngine) {
	defer func() {
		err := recover()
		if err != nil {
			sentry.CurrentHub().Recover(err)
			sentry.Flush(time.Second * 5)
		}
	}()

	tileSet := tilesets.StandardTileSet()

	breadcrumbs := Breadcrumbs{scope: scope}
	seed := time.Now().UnixNano()
	gameWithID, err := eng.GenerateGameSeeded(tileSet, seed)
	if err != nil {
		sentry.CaptureException(err)
		return
	}
	game, gameID := gameWithID.Game, gameWithID.ID
	scope.SetContext(CONTEXT_GAME_INFO, map[string]interface{}{
		CONTEXT_GAME_INFO_ID:   strconv.Itoa(gameID),
		CONTEXT_GAME_INFO_SEED: strconv.FormatInt(seed, 10),
	})

	msg := fmt.Sprintf("Game(%v): created with seed %v", gameID, seed)
	log.Printf("%v\n", msg)
	breadcrumbs.Add(msg)

	rng := rand.New(rand.NewSource(seed))

	for i := range len(tileSet.Tiles) {
		legalMovesReq := &engine.GetLegalMovesRequest{
			BaseGameID: gameID, TileToPlace: game.CurrentTile,
		}
		legalMovesResp := eng.SendGetLegalMovesBatch(
			[]*engine.GetLegalMovesRequest{legalMovesReq},
		)[0]
		if legalMovesResp.Err() != nil {
			sentry.CaptureException(legalMovesResp.Err())
			return
		}

		rng.Shuffle(len(legalMovesResp.Moves), func(i, j int) {
			legalMovesResp.Moves[i], legalMovesResp.Moves[j] = legalMovesResp.Moves[j], legalMovesResp.Moves[i]
		})
		move := legalMovesResp.Moves[0].Move
		breadcrumbs.Add(fmt.Sprintf(
			"Game(%v): iteration %v got moves, selecting:\n%#v\nat position %v",
			gameID,
			i,
			move,
			move.Position,
		))
		playTurnReq := &engine.PlayTurnRequest{GameID: gameID, Move: move}
		playTurnResp := eng.SendPlayTurnBatch([]*engine.PlayTurnRequest{playTurnReq})[0]
		if playTurnResp.Err() != nil {
			sentry.CaptureException(playTurnResp.Err())
			return
		}
		breadcrumbs.Add(fmt.Sprintf("Game(%v): iteration %v played turn", gameID, i))

		game = playTurnResp.Game
		gameID = playTurnResp.GameID()

		if len(game.CurrentTile.Features) == 0 {
			// number of tiles in the tile set and number of tiles that you actually
			// get to place can differ, if a tile that's next in the stack happens to
			// not have any position to place available
			break
		}
	}

	if len(game.CurrentTile.Features) != 0 {
		sentry.CaptureException(&ExpectedNoTileErr{game})
	}
}

func formatStackTrace(stacktrace *sentry.Stacktrace) string {
	parts := make([]string, len(stacktrace.Frames))
	for i, frame := range stacktrace.Frames {
		parts[i] = fmt.Sprintf(
			"%v:%v:%v\n%v",
			frame.AbsPath,
			frame.Lineno,
			frame.Colno,
			strings.Replace(frame.ContextLine, "\t", "    ", -1),
		)
	}
	return strings.Join(parts, "\n")
}

func main() {
	sentryDebugRaw := os.Getenv("SENTRY_DEBUG")
	if sentryDebugRaw == "" {
		sentryDebugRaw = "0"
	}
	sentryDebug, err := strconv.ParseBool(sentryDebugRaw)
	if err != nil {
		panic(err)
	}
	logsDir := path.Join("logs", fmt.Sprintf("%v", time.Now().UnixNano()))

	err = sentry.Init(sentry.ClientOptions{
		Dsn:   os.Getenv("SENTRY_DSN"),
		Debug: sentryDebug,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			gameInfo := event.Contexts[CONTEXT_GAME_INFO]
			gameId := gameInfo[CONTEXT_GAME_INFO_ID]
			logFilename := fmt.Sprintf("%v.jsonl", gameId)
			logPath := path.Join(logsDir, logFilename)
			payload, err := os.ReadFile(logPath)
			if err != nil {
				payload = []byte(fmt.Sprintf(
					"ERROR: could not read the log file at %v:\n%v",
					logPath,
					err.Error(),
				))
			}
			event.Attachments = append(event.Attachments, &sentry.Attachment{
				Filename:    logFilename,
				ContentType: "text/plain",
				Payload:     payload,
			})

			breadcrumbs := event.Contexts[CONTEXT_BREADCRUMBS][CONTEXT_BREADCRUMBS_MESSAGES].([]string)
			breadcrumbSb := strings.Builder{}
			for _, breadcrumb := range breadcrumbs {
				breadcrumbSb.WriteString("  - ")
				breadcrumbSb.WriteString(breadcrumb)
				breadcrumbSb.WriteString("\n")
			}
			event.Attachments = append(event.Attachments, &sentry.Attachment{
				Filename:    "breadcrumbs.log",
				ContentType: "text/plain",
				Payload:     []byte(breadcrumbSb.String()),
			})
			delete(event.Contexts, CONTEXT_BREADCRUMBS)

			sb := strings.Builder{}
			sb.WriteString(event.Message)
			sb.WriteString("\n")
			if len(breadcrumbs) != 0 {
				sb.WriteString("- breadcrumbs:\n")
			}
			sb.WriteString(breadcrumbSb.String())
			if len(event.Threads) != 0 {
				sb.WriteString("- threads:\n")
			}
			for i, thread := range event.Threads {
				sb.WriteString("  - thread ")
				sb.WriteString(strconv.Itoa(i))
				sb.WriteString(":\n")
				sb.WriteString("    - stack trace:\n")
				sb.WriteString(formatStackTrace(thread.Stacktrace))
				sb.WriteString("\n")
			}
			if len(event.Exception) != 0 {
				sb.WriteString("- exceptions:\n")
			}
			for i, exc := range event.Exception {
				sb.WriteString("  - exception ")
				sb.WriteString(strconv.Itoa(i))
				sb.WriteString(":\n")
				sb.WriteString(exc.Type)
				sb.WriteString(": ")
				sb.WriteString(exc.Value)
				sb.WriteString("\n")
				sb.WriteString("    - stack trace:\n")
				sb.WriteString(formatStackTrace(exc.Stacktrace))
				sb.WriteString("\n")
			}
			log.Print(sb.String())
			return event
		},
		MaxBreadcrumbs:   100,
		AttachStacktrace: true,
	})
	if err != nil {
		panic(err)
	}
	defer sentry.Flush(15 * time.Second)

	eng, err := engine.StartGameEngine(4, logsDir)
	if err != nil {
		sentry.CaptureException(err)
		log.Fatal("failed to start game engine\n")
	}

	for {
		sentry.WithScope(func(scope *sentry.Scope) {
			singleGameEngineRun(scope, eng)
		})
	}
}
