package engine

import "testing"

func TestRNG_Deterministic(t *testing.T) {
	rng1 := NewRNG(42)
	rng2 := NewRNG(42)

	for i := 0; i < 20; i++ {
		a := rng1.Roll(6)
		b := rng2.Roll(6)
		if a != b {
			t.Fatalf("roll %d: got %d and %d from same seed", i, a, b)
		}
	}
}

func TestRNG_Roll_Range(t *testing.T) {
	rng := NewRNG(99)

	for i := 0; i < 1000; i++ {
		r := rng.Roll(6)
		if r < 1 || r > 6 {
			t.Fatalf("roll out of range [1,6]: got %d", r)
		}
	}
}

func TestRNG_Roll_OneSided(t *testing.T) {
	rng := NewRNG(1)

	for i := 0; i < 10; i++ {
		if r := rng.Roll(1); r != 1 {
			t.Fatalf("1-sided die should always be 1, got %d", r)
		}
	}
}

func TestRNG_WeightedSelect_Deterministic(t *testing.T) {
	rng1 := NewRNG(42)
	rng2 := NewRNG(42)
	weights := []int{70, 20, 10}

	for i := 0; i < 20; i++ {
		a := rng1.WeightedSelect(weights)
		b := rng2.WeightedSelect(weights)
		if a != b {
			t.Fatalf("selection %d: got %d and %d from same seed", i, a, b)
		}
	}
}

func TestRNG_WeightedSelect_Distribution(t *testing.T) {
	rng := NewRNG(12345)
	weights := []int{70, 20, 10}
	counts := [3]int{}

	const trials = 10000
	for i := 0; i < trials; i++ {
		idx := rng.WeightedSelect(weights)
		if idx < 0 || idx > 2 {
			t.Fatalf("index out of range: %d", idx)
		}
		counts[idx]++
	}

	// With 10k trials, expect roughly 70%/20%/10% ± some margin.
	if counts[0] < 6000 || counts[0] > 8000 {
		t.Errorf("expected ~7000 for weight 70, got %d", counts[0])
	}
	if counts[1] < 1000 || counts[1] > 3000 {
		t.Errorf("expected ~2000 for weight 20, got %d", counts[1])
	}
	if counts[2] < 200 || counts[2] > 1800 {
		t.Errorf("expected ~1000 for weight 10, got %d", counts[2])
	}
}

func TestRNG_WeightedSelect_SingleOption(t *testing.T) {
	rng := NewRNG(1)

	for i := 0; i < 10; i++ {
		if idx := rng.WeightedSelect([]int{100}); idx != 0 {
			t.Fatalf("single option should always be 0, got %d", idx)
		}
	}
}

func TestRNG_Position_Tracks(t *testing.T) {
	rng := NewRNG(42)

	if rng.Position() != 0 {
		t.Fatalf("expected position 0, got %d", rng.Position())
	}

	rng.Roll(6)
	if rng.Position() != 1 {
		t.Fatalf("expected position 1, got %d", rng.Position())
	}

	rng.WeightedSelect([]int{50, 50})
	if rng.Position() != 2 {
		t.Fatalf("expected position 2, got %d", rng.Position())
	}

	rng.Roll(20)
	rng.Roll(20)
	if rng.Position() != 4 {
		t.Fatalf("expected position 4, got %d", rng.Position())
	}
}

func TestRNG_Restore_MatchesPosition(t *testing.T) {
	// Advance an RNG to position 10 and record the next 5 rolls.
	rng := NewRNG(42)
	for i := 0; i < 10; i++ {
		rng.Roll(6)
	}

	var expected [5]int
	for i := range expected {
		expected[i] = rng.Roll(6)
	}

	// Restore to position 10 and verify same rolls.
	restored := RestoreRNG(42, 10)
	if restored.Position() != 10 {
		t.Fatalf("expected position 10, got %d", restored.Position())
	}

	for i, want := range expected {
		got := restored.Roll(6)
		if got != want {
			t.Fatalf("roll %d: expected %d, got %d", i, want, got)
		}
	}
}

func TestRNG_DifferentSeeds_DifferentResults(t *testing.T) {
	rng1 := NewRNG(1)
	rng2 := NewRNG(2)

	// With different seeds, at least some rolls should differ.
	differs := false
	for i := 0; i < 20; i++ {
		if rng1.Roll(100) != rng2.Roll(100) {
			differs = true
			break
		}
	}
	if !differs {
		t.Error("expected different seeds to produce different results")
	}
}
