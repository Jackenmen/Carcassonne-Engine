package stack

import (
	"errors"
	"math/rand" //nolint:gosec// Weak number generator is sufficent in our case
	"slices"
	"time"
)

type Comparable[T any] interface {
	Equals(T) bool
}

type Stack[T Comparable[T]] struct {
	seed   int64
	turnNo int32
	tiles  []T
	order  []int32
}

var (
	ErrStackOutOfBounds = errors.New("stack: out of bounds")
	ErrTileNotFound     = errors.New("could not find the given tile")
)

// New creates new Stack and shuffles it using current time as seed.
// NODE: Input slice is not copied.
func New[T Comparable[T]](tiles []T) Stack[T] {
	return NewSeeded(tiles, time.Now().UnixNano())
}

// NewSeeded creates new Stack and shuffles it using the provided seed.
// NODE: Input slice is not copied.
func NewSeeded[T Comparable[T]](tiles []T, seed int64) Stack[T] {
	stack := NewOrdered(tiles)
	stack.seed = seed
	rng := rand.New(rand.NewSource(stack.seed)) //nolint:gosec// Weak number generator is sufficent in our case
	rng.Shuffle(len(stack.order), func(i, j int) {
		stack.order[i], stack.order[j] = stack.order[j], stack.order[i]
	})
	return stack
}

// NewOrdered creates new Stack and maintains original order.
// NODE: Input slice is not copied.
func NewOrdered[T Comparable[T]](tiles []T) Stack[T] {
	stack := Stack[T]{
		seed:   0,
		turnNo: 0,
		tiles:  tiles,
		order:  make([]int32, len(tiles)),
	}
	for i := range len(tiles) {
		stack.order[i] = int32(i)
	}
	return stack
}

func (s Stack[T]) DeepClone() Stack[T] {
	s.order = slices.Clone(s.order)
	return s
}

func (s Stack[T]) GetRemaining() []T {
	tiles := []T{}
	for _, i := range s.order[s.turnNo:] {
		tiles = append(tiles, s.tiles[i])
	}
	return tiles
}

func (s Stack[T]) GetRemainingTileCount() int32 {
	return int32(len(s.tiles)) - s.turnNo
}

func (s Stack[T]) GetTotalTileCount() int32 {
	return int32(len(s.tiles))
}

// returns the original tiles slice (input to constructor), not a shuffled slice
func (s Stack[T]) GetTiles() []T {
	return s.tiles
}

func (s Stack[T]) Get(n int32) (T, error) {
	if n >= int32(len(s.tiles)) {
		return *new(T), ErrStackOutOfBounds
	}
	return s.tiles[s.order[n]], nil
}

func (s *Stack[T]) Next() (T, error) {
	defer func() { s.turnNo++ }()
	return s.Get(s.turnNo)
}

func (s Stack[T]) Peek() (T, error) {
	return s.Get(s.turnNo)
}

func (s *Stack[T]) MoveToTop(tile T) error {
	// TODO: this is O(n) but could be optimized to O(1) with an additional map
	// keyed by the tile
	order := s.order[s.turnNo:]
	for turnIndex, tileIndex := range order {
		if s.tiles[tileIndex].Equals(tile) {
			order[turnIndex], order[0] = order[0], order[turnIndex]
			return nil
		}
	}
	return ErrTileNotFound
}
