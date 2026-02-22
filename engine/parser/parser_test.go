package parser

import (
	"testing"

	"github.com/nathoo/questcore/types"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  types.Intent
	}{
		// Empty / whitespace
		{
			name:  "empty string",
			input: "",
			want:  types.Intent{},
		},
		{
			name:  "whitespace only",
			input: "   ",
			want:  types.Intent{},
		},

		// Basic verbs (no object)
		{
			name:  "look",
			input: "look",
			want:  types.Intent{Verb: "look"},
		},
		{
			name:  "inventory",
			input: "inventory",
			want:  types.Intent{Verb: "inventory"},
		},

		// Verb aliases
		{
			name:  "l → look",
			input: "l",
			want:  types.Intent{Verb: "look"},
		},
		{
			name:  "i → inventory",
			input: "i",
			want:  types.Intent{Verb: "inventory"},
		},
		{
			name:  "x sword → examine sword",
			input: "x sword",
			want:  types.Intent{Verb: "examine", Object: "sword"},
		},
		{
			name:  "get key → take key",
			input: "get key",
			want:  types.Intent{Verb: "take", Object: "key"},
		},
		{
			name:  "hit goblin → attack goblin",
			input: "hit goblin",
			want:  types.Intent{Verb: "attack", Object: "goblin"},
		},

		// Direction shortcuts
		{
			name:  "n → go north",
			input: "n",
			want:  types.Intent{Verb: "go", Object: "north"},
		},
		{
			name:  "s → go south",
			input: "s",
			want:  types.Intent{Verb: "go", Object: "south"},
		},
		{
			name:  "e → go east",
			input: "e",
			want:  types.Intent{Verb: "go", Object: "east"},
		},
		{
			name:  "w → go west",
			input: "w",
			want:  types.Intent{Verb: "go", Object: "west"},
		},
		{
			name:  "ne → go northeast",
			input: "ne",
			want:  types.Intent{Verb: "go", Object: "northeast"},
		},
		{
			name:  "se → go southeast",
			input: "se",
			want:  types.Intent{Verb: "go", Object: "southeast"},
		},
		{
			name:  "nw → go northwest",
			input: "nw",
			want:  types.Intent{Verb: "go", Object: "northwest"},
		},
		{
			name:  "sw → go southwest",
			input: "sw",
			want:  types.Intent{Verb: "go", Object: "southwest"},
		},
		{
			name:  "u → go up",
			input: "u",
			want:  types.Intent{Verb: "go", Object: "up"},
		},
		{
			name:  "d → go down",
			input: "d",
			want:  types.Intent{Verb: "go", Object: "down"},
		},
		{
			name:  "north → go north",
			input: "north",
			want:  types.Intent{Verb: "go", Object: "north"},
		},
		{
			name:  "up → go up",
			input: "up",
			want:  types.Intent{Verb: "go", Object: "up"},
		},

		// Explicit go
		{
			name:  "go north",
			input: "go north",
			want:  types.Intent{Verb: "go", Object: "north"},
		},

		// Verb + object
		{
			name:  "take key",
			input: "take key",
			want:  types.Intent{Verb: "take", Object: "key"},
		},
		{
			name:  "drop sword",
			input: "drop sword",
			want:  types.Intent{Verb: "drop", Object: "sword"},
		},
		{
			name:  "open door",
			input: "open door",
			want:  types.Intent{Verb: "open", Object: "door"},
		},
		{
			name:  "use lamp",
			input: "use lamp",
			want:  types.Intent{Verb: "use", Object: "lamp"},
		},

		// Preposition as delimiter
		{
			name:  "use key on door",
			input: "use key on door",
			want:  types.Intent{Verb: "use", Object: "key", Target: "door"},
		},
		{
			name:  "use sword on goblin",
			input: "use sword on goblin",
			want:  types.Intent{Verb: "use", Object: "sword", Target: "goblin"},
		},
		{
			name:  "attack goblin with sword",
			input: "attack goblin with sword",
			want:  types.Intent{Verb: "attack", Object: "goblin", Target: "sword"},
		},

		// Multi-word objects (the core fix)
		{
			name:  "take rusty key → multi-word object",
			input: "take rusty key",
			want:  types.Intent{Verb: "take", Object: "rusty key"},
		},
		{
			name:  "use rusty key on iron door → multi-word object and target",
			input: "use rusty key on iron door",
			want:  types.Intent{Verb: "use", Object: "rusty key", Target: "iron door"},
		},
		{
			name:  "examine old book → multi-word object",
			input: "examine old book",
			want:  types.Intent{Verb: "examine", Object: "old book"},
		},

		// Article stripping
		{
			name:  "take the key → article stripped",
			input: "take the key",
			want:  types.Intent{Verb: "take", Object: "key"},
		},
		{
			name:  "take a sword → article stripped",
			input: "take a sword",
			want:  types.Intent{Verb: "take", Object: "sword"},
		},
		{
			name:  "use the rusty key on the iron door → articles stripped",
			input: "use the rusty key on the iron door",
			want:  types.Intent{Verb: "use", Object: "rusty key", Target: "iron door"},
		},

		// Ask alias
		{
			name:  "ask captain about quest → talk with topic",
			input: "ask captain about quest",
			want:  types.Intent{Verb: "talk", Object: "captain", Target: "quest"},
		},

		// Movement aliases
		{
			name:  "walk north → go north",
			input: "walk north",
			want:  types.Intent{Verb: "go", Object: "north"},
		},
		{
			name:  "run east → go east",
			input: "run east",
			want:  types.Intent{Verb: "go", Object: "east"},
		},
		{
			name:  "move south → go south",
			input: "move south",
			want:  types.Intent{Verb: "go", Object: "south"},
		},

		// Talk aliases
		{
			name:  "speak guard → talk guard",
			input: "speak guard",
			want:  types.Intent{Verb: "talk", Object: "guard"},
		},
		{
			name:  "chat guard → talk guard",
			input: "chat guard",
			want:  types.Intent{Verb: "talk", Object: "guard"},
		},

		// Combat aliases
		{
			name:  "kill dragon → attack dragon",
			input: "kill dragon",
			want:  types.Intent{Verb: "attack", Object: "dragon"},
		},
		{
			name:  "fight troll → attack troll",
			input: "fight troll",
			want:  types.Intent{Verb: "attack", Object: "troll"},
		},

		// Read passes through (not aliased — rules may handle it)
		{
			name:  "read book → read book",
			input: "read book",
			want:  types.Intent{Verb: "read", Object: "book"},
		},

		// Wait alias
		{
			name:  "z → wait",
			input: "z",
			want:  types.Intent{Verb: "wait"},
		},

		// Multi-word verbs
		{
			name:  "look at painting",
			input: "look at painting",
			want:  types.Intent{Verb: "examine", Object: "painting"},
		},
		{
			name:  "look in chest → examine chest",
			input: "look in chest",
			want:  types.Intent{Verb: "examine", Object: "chest"},
		},
		{
			name:  "pick up key",
			input: "pick up key",
			want:  types.Intent{Verb: "take", Object: "key"},
		},
		{
			name:  "talk to guard",
			input: "talk to guard",
			want:  types.Intent{Verb: "talk", Object: "guard"},
		},
		{
			name:  "speak to guard → talk guard",
			input: "speak to guard",
			want:  types.Intent{Verb: "talk", Object: "guard"},
		},
		{
			name:  "speak with guard → talk guard",
			input: "speak with guard",
			want:  types.Intent{Verb: "talk", Object: "guard"},
		},
		{
			name:  "chat with guard → talk guard",
			input: "chat with guard",
			want:  types.Intent{Verb: "talk", Object: "guard"},
		},
		{
			name:  "talk guard (no preposition)",
			input: "talk guard",
			want:  types.Intent{Verb: "talk", Object: "guard"},
		},

		// Case insensitivity
		{
			name:  "LOOK AT PAINTING",
			input: "LOOK AT PAINTING",
			want:  types.Intent{Verb: "examine", Object: "painting"},
		},
		{
			name:  "Take Key",
			input: "Take Key",
			want:  types.Intent{Verb: "take", Object: "key"},
		},

		// Unknown verb passes through
		{
			name:  "unknown verb",
			input: "dance",
			want:  types.Intent{Verb: "dance"},
		},
		{
			name:  "unknown verb with object",
			input: "push boulder",
			want:  types.Intent{Verb: "push", Object: "boulder"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if got != tt.want {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}
