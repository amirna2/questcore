// Package save implements JSON serialization and deserialization of game state.
package save

import (
	"encoding/json"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// SaveData is the JSON-serializable save format.
type SaveData struct {
	Version     string                       `json:"version"`
	Game        string                       `json:"game"`
	Turn        int                          `json:"turn"`
	Player      types.Player                 `json:"player"`
	Flags       map[string]bool              `json:"flags"`
	Counters    map[string]int               `json:"counters"`
	EntityState map[string]types.EntityState `json:"entity_state"`
	RNGSeed     int64                        `json:"rng_seed"`
	CommandLog  []string                     `json:"command_log"`
}

// Save serializes game state to JSON bytes.
func Save(s *types.State, defs *state.Defs) ([]byte, error) {
	data := SaveData{
		Version:     defs.Game.Version,
		Game:        defs.Game.Title,
		Turn:        s.TurnCount,
		Player:      s.Player,
		Flags:       s.Flags,
		Counters:    s.Counters,
		EntityState: s.Entities,
		RNGSeed:     s.RNGSeed,
		CommandLog:  s.CommandLog,
	}
	return json.MarshalIndent(data, "", "  ")
}

// Load deserializes JSON bytes into SaveData.
func Load(data []byte) (*SaveData, error) {
	var sd SaveData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, err
	}
	// Ensure maps are never nil after load.
	if sd.Flags == nil {
		sd.Flags = map[string]bool{}
	}
	if sd.Counters == nil {
		sd.Counters = map[string]int{}
	}
	if sd.EntityState == nil {
		sd.EntityState = map[string]types.EntityState{}
	}
	if sd.Player.Inventory == nil {
		sd.Player.Inventory = []string{}
	}
	if sd.Player.Stats == nil {
		sd.Player.Stats = map[string]int{}
	}
	if sd.CommandLog == nil {
		sd.CommandLog = []string{}
	}
	return &sd, nil
}

// ApplySave applies loaded save data onto a state.
func ApplySave(s *types.State, sd *SaveData) {
	s.Player = sd.Player
	s.Flags = sd.Flags
	s.Counters = sd.Counters
	s.Entities = sd.EntityState
	s.TurnCount = sd.Turn
	s.RNGSeed = sd.RNGSeed
	s.CommandLog = sd.CommandLog
}
