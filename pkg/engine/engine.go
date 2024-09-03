package engine

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/deck"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/game"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/logger"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/stack"
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/tilesets"
)

var (
	ErrCommunicatorClosed  = errors.New("communicator is closed")
	ErrGameNotFound        = errors.New("game with the given ID was not found")
	ErrLockAlreadyAcquired = errors.New("lock for game with this ID is already acquired")
)

const (
	inputBufferSize            = 10_000_000
	childrenCleanupWarnMsg     = "WARN: children of the game with ID %v have not been cleaned up"
	childrenCleanupWarnFullMsg = childrenCleanupWarnMsg + ": %#v\n"
)

type ExecutionPanicError struct {
	panicValues []any
	stacks      [][]byte
}

func (err *ExecutionPanicError) Error() string {
	formattedErrors := make([]string, len(err.panicValues))
	for i := range err.panicValues {
		formattedErrors[i] = fmt.Sprintf(
			"%#v\n- stack trace:\n%s", err.panicValues[i], err.stacks[i],
		)
	}
	return fmt.Sprintf(
		"panic during request execution: %v", strings.Join(formattedErrors, "\n"),
	)
}

func processWorkerInput(input *workerInput) (resp Response) {
	defer func() {
		if err := recover(); err != nil {
			resp = &SyncResponse{
				BaseResponse{
					gameID: input.request.gameID(),
					err: &ExecutionPanicError{
						panicValues: []any{err},
						stacks:      [][]byte{debug.Stack()},
					},
				},
			}
		}
	}()

	return input.request.execute(input.game)
}

func worker(comm *communicator) {
	defer comm.workGroup.Done()

	for input := range comm.inputBuffer {
		resp := processWorkerInput(&input)
		input.outputBuffer <- workerOutput{
			requestID: input.requestID,
			resp:      resp,
		}
		input.waitGroup.Done()
	}
}

// Internal struct with communication tools used by the game engine to send requests
// and cooperatively shutdown the workers.
type communicator struct {
	workGroup   sync.WaitGroup
	inputBuffer chan workerInput
	closed      bool
}

func newCommunicator() *communicator {
	return &communicator{
		inputBuffer: make(chan workerInput, inputBufferSize),
	}
}

func (comm *communicator) Close() {
	if comm.closed {
		comm.workGroup.Wait()
		return
	}
	comm.closed = true
	close(comm.inputBuffer)
	comm.workGroup.Wait()
}

type SerializedGameWithID struct {
	ID   int
	Game game.SerializedGame
}

// The entry point for Python side of things - the engine keeps track of created games,
// sends requests to its workers and returns back responses received from them.
type GameEngine struct {
	comm          *communicator
	logDir        string
	games         map[int]*game.Game
	gameMutexes   map[int]*sync.RWMutex
	nextGameID    int
	nextRequestID int
	closed        bool
	childGames    map[int]map[int]struct{}
	parentGames   map[int]int
	appLogger     *log.Logger
}

func StartGameEngine(workerCount int, logDir string) (*GameEngine, error) {
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return nil, err
	}
	comm := newCommunicator()
	engine := &GameEngine{
		comm:          comm,
		logDir:        logDir,
		games:         map[int]*game.Game{},
		gameMutexes:   map[int]*sync.RWMutex{},
		nextGameID:    1,
		nextRequestID: 1,
		childGames:    map[int]map[int]struct{}{},
		parentGames:   map[int]int{},
		appLogger:     log.New(os.Stderr, "", log.LstdFlags),
	}

	for range workerCount {
		comm.workGroup.Add(1)
		go worker(comm)
	}

	return engine, nil
}

func (engine *GameEngine) IsClosed() bool {
	return engine.closed
}

func (engine *GameEngine) Close() {
	if engine.closed {
		return
	}
	engine.closed = true
	engine.comm.Close()
}

// Generate a random game from the given tileset.
func (engine *GameEngine) GenerateGame(tileSet tilesets.TileSet) (SerializedGameWithID, error) {
	deckStack := stack.New(tileSet.Tiles)
	deck := deck.Deck{Stack: &deckStack, StartingTile: tileSet.StartingTile}
	return engine.generateGameFromDeck(deck)
}

// Generate a game from the given tileset using its defined tile order.
//
// Usage for games played by an agent is ill-advised - the serialized game reveals
// the tileset and the order in it will be consistent with stack's order.
func (engine *GameEngine) GenerateOrderedGame(tileSet tilesets.TileSet) (SerializedGameWithID, error) {
	deckStack := stack.NewOrdered(tileSet.Tiles)
	deck := deck.Deck{Stack: &deckStack, StartingTile: tileSet.StartingTile}
	return engine.generateGameFromDeck(deck)
}

func (engine *GameEngine) GenerateGameSeeded(tileSet tilesets.TileSet, seed int64) (SerializedGameWithID, error) {
	deckStack := stack.NewSeeded(tileSet.Tiles, seed)
	deck := deck.Deck{
		Stack:        &deckStack,
		StartingTile: tileSet.StartingTile,
	}

	return engine.generateGameFromDeck(deck)
}

func (engine *GameEngine) generateGameFromDeck(deck deck.Deck) (SerializedGameWithID, error) {
	id := engine.nextGameID
	engine.nextGameID++

	logFile := path.Join(engine.logDir, fmt.Sprintf("%v.jsonl", id))
	logger, err := logger.NewFromFile(logFile)
	if err != nil {
		return SerializedGameWithID{}, err
	}

	g, err := game.NewFromDeck(deck, &logger)
	if err != nil {
		return SerializedGameWithID{}, err
	}

	engine.games[id] = g
	engine.gameMutexes[id] = &sync.RWMutex{}
	return SerializedGameWithID{id, g.Serialized()}, nil
}

// *Fully* clone the game (including its log) with the given ID `count` times
// returning the IDs of the cloned games.
// Intended use: Allowing multiple agents to play the same game scenario.
func (engine *GameEngine) CloneGame(gameID int, count int) ([]int, error) {
	return engine.cloneGame(gameID, count, true)
}

// Clone the game with the given ID `count` times and track the clones as the children
// to the given game. All of the game's children should be cleaned up before a turn is
// played on the parent game, otherwise a warning is issued.
//
// Returns the IDs of the cloned games.
//
// Intended use: Allowing a single agent to expand the game tree through child games
// until it's ready to make its turn.
func (engine *GameEngine) SubCloneGame(gameID int, count int) ([]int, error) {
	ret, err := engine.cloneGame(gameID, count, false)
	if err != nil {
		return ret, err
	}

	childGames, ok := engine.childGames[gameID]
	if !ok {
		childGames = map[int]struct{}{}
		engine.childGames[gameID] = childGames
	}

	for _, childID := range ret {
		childGames[childID] = struct{}{}
		engine.parentGames[childID] = gameID
	}
	return ret, nil
}

// Delete games with the given IDs.
func (engine *GameEngine) DeleteGames(gameIDs []int) {
	for _, gameID := range gameIDs {
		if len(engine.childGames[gameID]) != 0 {
			// since we don't know whether the agent isn't actually using these,
			// just settle on a warning
			engine.appLogger.Printf(
				childrenCleanupWarnFullMsg,
				gameID,
				engine.childGames[gameID],
			)
		}
		delete(engine.games, gameID)
		delete(engine.gameMutexes, gameID)
		delete(engine.childGames, gameID)
		parentID := engine.parentGames[gameID]
		if parentID != 0 {
			delete(engine.childGames[parentID], gameID)
		}
	}
}

func (engine *GameEngine) cloneGame(gameID int, count int, full bool) ([]int, error) {
	reservedIDs := make([]int, count)
	for i := range count {
		reservedIDs[i] = engine.nextGameID
		engine.nextGameID++
	}

	logDir := ""
	if full {
		logDir = engine.logDir
	}
	req := &cloneGameRequest{GameID: gameID, ReservedIDs: reservedIDs, LogDir: logDir}
	responses := engine.sendBatch([]Request{req})
	if err := responses[0].Err(); err != nil {
		return nil, err
	}

	resp := responses[0].(*cloneGameResponse)
	for i, game := range resp.Clones {
		engine.games[reservedIDs[i]] = game
		engine.gameMutexes[reservedIDs[i]] = &sync.RWMutex{}
	}

	return reservedIDs, nil
}

// Due to limitations of Python bindings generator with []interface return type,
// this wraps sendBatch() and limits the return type to only one Response type.
func (engine *GameEngine) SendPlayTurnBatch(concreteRequests []*PlayTurnRequest) []*PlayTurnResponse {
	requests := make([]Request, len(concreteRequests))
	for i := range concreteRequests {
		requests[i] = concreteRequests[i]
	}
	responses := engine.sendBatch(requests)
	concreteResponses := make([]*PlayTurnResponse, len(responses))
	for i := range responses {
		var ok bool
		concreteResponses[i], ok = responses[i].(*PlayTurnResponse)
		if !ok {
			// we can get a SyncResponse here, if the request didn't reach
			// a worker due to failure during prepareWorkerInput
			// this *is* stupid but it's what we have to deal with due to
			// a limitation with auto-generated bindings breaking on
			// a `[]Interface` return:
			// https://github.com/go-python/gopy/issues/357
			concreteResponses[i] = &PlayTurnResponse{
				BaseResponse: responses[i].(*SyncResponse).BaseResponse,
			}
		}
	}
	return concreteResponses
}

// Due to limitations of Python bindings generator with []interface return type,
// this wraps sendBatch() and limits the return type to only one Response type.
func (engine *GameEngine) SendGetRemainingTilesBatch(concreteRequests []*GetRemainingTilesRequest) []*GetRemainingTilesResponse {
	requests := make([]Request, len(concreteRequests))
	for i := range concreteRequests {
		requests[i] = concreteRequests[i]
	}
	responses := engine.sendBatch(requests)
	concreteResponses := make([]*GetRemainingTilesResponse, len(responses))
	for i := range responses {
		var ok bool
		concreteResponses[i], ok = responses[i].(*GetRemainingTilesResponse)
		if !ok {
			// we can get a SyncResponse here, if the request didn't reach
			// a worker due to failure during prepareWorkerInput
			// this *is* stupid but it's what we have to deal with due to
			// a limitation with auto-generated bindings breaking on
			// a `[]Interface` return:
			// https://github.com/go-python/gopy/issues/357
			concreteResponses[i] = &GetRemainingTilesResponse{
				BaseResponse: responses[i].(*SyncResponse).BaseResponse,
			}
		}
	}
	return concreteResponses
}

// Due to limitations of Python bindings generator with []interface return type,
// this wraps sendBatch() and limits the return type to only one Response type.
func (engine *GameEngine) SendGetLegalMovesBatch(concreteRequests []*GetLegalMovesRequest) []*GetLegalMovesResponse {
	requests := make([]Request, len(concreteRequests))
	for i := range concreteRequests {
		requests[i] = concreteRequests[i]
	}
	responses := engine.sendBatch(requests)
	concreteResponses := make([]*GetLegalMovesResponse, len(responses))
	for i := range responses {
		var ok bool
		concreteResponses[i], ok = responses[i].(*GetLegalMovesResponse)
		if !ok {
			// we can get a SyncResponse here, if the request didn't reach
			// a worker due to failure during prepareWorkerInput
			// this *is* stupid but it's what we have to deal with due to
			// a limitation with auto-generated bindings breaking on
			// a `[]Interface` return:
			// https://github.com/go-python/gopy/issues/357
			concreteResponses[i] = &GetLegalMovesResponse{
				BaseResponse: responses[i].(*SyncResponse).BaseResponse,
			}
		}
	}
	return concreteResponses
}

// Due to limitations of Python bindings generator with []interface return type,
// this wraps sendBatch() and limits the return type to only one Response type.
func (engine *GameEngine) SendGetMidGameScoreBatch(concreteRequests []*GetMidGameScoreRequest) []*GetMidGameScoreResponse {
	requests := make([]Request, len(concreteRequests))
	for i := range concreteRequests {
		requests[i] = concreteRequests[i]
	}
	responses := engine.sendBatch(requests)
	concreteResponses := make([]*GetMidGameScoreResponse, len(responses))
	for i := range responses {
		var ok bool
		concreteResponses[i], ok = responses[i].(*GetMidGameScoreResponse)
		if !ok {
			// we can get a SyncResponse here, if the request didn't reach
			// a worker due to failure during prepareWorkerInput
			// this *is* stupid but it's what we have to deal with due to
			// a limitation with auto-generated bindings breaking on
			// a `[]Interface` return:
			// https://github.com/go-python/gopy/issues/357
			concreteResponses[i] = &GetMidGameScoreResponse{
				BaseResponse: responses[i].(*SyncResponse).BaseResponse,
			}
		}
	}
	return concreteResponses
}

// API for handling the sent requests using background workers.
// The order and types of returned responses correspond to the requests slice.
//
// If a request fails before reaching a worker, a `SyncResponse` type will
// be returned in place of the worker response.
// If a panic occurs during execution of this method, it will be captured
// and a SyncResponse with ExecutionPanicError will be returned for relevant
// (or all) requests.
//
// Concurrent calls to this function can be made but no more than
// one request for a *single* game (ID) can be performed at the same time
// to avoid concurrent writes by the workers on different threads.
// You will receive `ErrGameNotFound` error, if you try doing so.
func (engine *GameEngine) sendBatch(requests []Request) (responses []Response) {
	batch := newRequestBatch(engine, requests)
	batch.Process()
	return batch.responses
}

// Prepare the input that will be sent through engine's input buffer
// to a worker. This mutates engine's games map (deleting the game from
// the request) and updates the next free request ID.
// If this function did not return an error, the caller needs to readd
// the game from the request after the worker is done with it.
func (engine *GameEngine) prepareWorkerInput(
	waitGroup *sync.WaitGroup,
	outputBuffer chan workerOutput,
	req Request,
) (workerInput, error) {
	if engine.comm.closed {
		return workerInput{}, ErrCommunicatorClosed
	}
	gameID := req.gameID()
	game, ok := engine.games[gameID]
	if !ok {
		return workerInput{}, ErrGameNotFound
	}
	mutex := engine.gameMutexes[gameID]
	canWrite := req.requiresWrite()
	if canWrite {
		if !mutex.TryLock() {
			return workerInput{}, ErrLockAlreadyAcquired
		}
	} else if !mutex.TryRLock() {
		return workerInput{}, ErrLockAlreadyAcquired
	}

	requestID := engine.nextRequestID
	engine.nextRequestID++
	return workerInput{
		requestID:    requestID,
		waitGroup:    waitGroup,
		outputBuffer: outputBuffer,
		game:         game,
		request:      req,
		canWrite:     canWrite,
	}, nil
}

func (engine *GameEngine) send(input workerInput) {
	input.waitGroup.Add(1)
	engine.comm.inputBuffer <- input
}

type requestBatch struct {
	engine                       *GameEngine
	requests                     []Request
	games                        map[int]*game.Game
	removableGames               map[int]struct{}
	parentsWithRemovableChildren map[int]struct{}
	responses                    []Response
	outputBuffer                 chan workerOutput
	waitGroup                    sync.WaitGroup
	outputWaitGroup              sync.WaitGroup
	panicErr                     ExecutionPanicError
}

func newRequestBatch(engine *GameEngine, requests []Request) requestBatch {
	return requestBatch{
		engine:                       engine,
		requests:                     requests,
		games:                        map[int]*game.Game{},
		removableGames:               map[int]struct{}{},
		parentsWithRemovableChildren: map[int]struct{}{},
		responses:                    make([]Response, len(requests)),
		outputBuffer:                 make(chan workerOutput, len(requests)),
	}
}

func (batch *requestBatch) Process() {
	outputItems := map[int]outputItemInfo{}
	outputItemsLock := sync.RWMutex{}

	defer batch.recoverWithCleanup()

	batch.outputWaitGroup.Add(1)
	go func() {
		defer func() {
			// one cannot recover from a panic that occurred in another goroutine
			// so we want to re-panic in the "parent" goroutine instead
			batch.recover(recover())
			batch.outputWaitGroup.Done()
		}()

		for output := range batch.outputBuffer {
			outputItemsLock.RLock()
			outputInfo := outputItems[output.requestID]
			outputItemsLock.RUnlock()
			batch.responses[outputInfo.RequestIndex] = output.resp

			if outputInfo.AcquiredWrite {
				batch.engine.gameMutexes[outputInfo.GameID].Unlock()
			} else {
				batch.engine.gameMutexes[outputInfo.GameID].RUnlock()
			}

			if respGameRemovable, ok := output.resp.(ResponseGameRemovable); ok {
				if respGameRemovable.canRemoveGame() {
					batch.removableGames[outputInfo.GameID] = struct{}{}
				}
			}

			if respChildGamesRemovable, ok := output.resp.(ResponseChildGamesRemovable); ok {
				if respChildGamesRemovable.canRemoveChildGames() {
					batch.parentsWithRemovableChildren[outputInfo.GameID] = struct{}{}
				}
			}
		}
	}()

	for i, req := range batch.requests {
		gameID := req.gameID()
		input, err := batch.engine.prepareWorkerInput(&batch.waitGroup, batch.outputBuffer, req)
		if err != nil {
			batch.responses[i] = &SyncResponse{BaseResponse{gameID: gameID, err: err}}
			continue
		}
		batch.games[gameID] = input.game
		outputItemsLock.Lock()
		outputItems[input.requestID] = outputItemInfo{
			GameID: gameID, RequestIndex: i, AcquiredWrite: input.canWrite,
		}
		outputItemsLock.Unlock()
		batch.engine.send(input)
	}
}

func (batch *requestBatch) cleanupGames() {
	// remove games for which we got information that we can remove them
	for gameID := range batch.games {
		_, canRemove := batch.removableGames[gameID]
		_, canRemoveChildren := batch.parentsWithRemovableChildren[gameID]

		if (canRemove || canRemoveChildren) && len(batch.engine.childGames[gameID]) != 0 {
			// since we don't know whether the agent isn't actually using these,
			// just settle on a warning
			batch.engine.appLogger.Printf(
				childrenCleanupWarnFullMsg,
				gameID,
				batch.engine.childGames[gameID],
			)
		}

		if canRemove {
			delete(batch.engine.games, gameID)
			delete(batch.engine.gameMutexes, gameID)
			delete(batch.engine.childGames, gameID)
			parentID := batch.engine.parentGames[gameID]
			if parentID != 0 {
				delete(batch.engine.childGames[parentID], gameID)
			}
		}
	}
}

func (batch *requestBatch) recover(panicValue any) {
	if panicValue != nil {
		batch.panicErr.panicValues = append(batch.panicErr.panicValues, panicValue)
		batch.panicErr.stacks = append(batch.panicErr.stacks, debug.Stack())
	}
}

func (batch *requestBatch) recoverWithCleanup() {
	// Wait for all workers request to finish running on the workers.
	batch.waitGroup.Wait()
	// Close the output buffer to let the response-handling goroutine know
	// that there will be no more requests and wait for it to finish.
	close(batch.outputBuffer)
	batch.outputWaitGroup.Wait()

	batch.recover(recover())

	if len(batch.panicErr.panicValues) != 0 {
		for i, req := range batch.requests {
			gameID := req.gameID()
			batch.responses[i] = &SyncResponse{
				BaseResponse{gameID: gameID, err: &batch.panicErr},
			}
		}
	}
	batch.cleanupGames()
}
