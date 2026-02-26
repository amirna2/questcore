package engine

import "math/rand"

// RNG wraps math/rand.Rand with deterministic position tracking.
// Position increments with every call, enabling save/restore.
type RNG struct {
	seed int64
	src  *rand.Rand
	pos  int64
}

// NewRNG creates a new deterministic RNG from a seed.
func NewRNG(seed int64) *RNG {
	return &RNG{
		seed: seed,
		src:  rand.New(rand.NewSource(seed)),
	}
}

// Roll returns a random integer in [1, sides].
func (r *RNG) Roll(sides int) int {
	r.pos++
	return r.src.Intn(sides) + 1
}

// WeightedSelect returns an index chosen by weighted random selection.
// weights must be non-empty with all positive values.
func (r *RNG) WeightedSelect(weights []int) int {
	total := 0
	for _, w := range weights {
		total += w
	}
	r.pos++
	roll := r.src.Intn(total)
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if roll < cumulative {
			return i
		}
	}
	return len(weights) - 1
}

// Position returns the number of RNG calls made since creation.
func (r *RNG) Position() int64 {
	return r.pos
}

// RestoreRNG creates an RNG and advances it to the given position.
// This reproduces the exact RNG state for save/load.
func RestoreRNG(seed int64, position int64) *RNG {
	rng := NewRNG(seed)
	for i := int64(0); i < position; i++ {
		rng.src.Int63()
	}
	rng.pos = position
	return rng
}
