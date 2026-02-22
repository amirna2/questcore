// Package parser converts command strings into Intent structs.
// Intentionally dumb: no NLP, just pattern matching.
package parser

import (
	"strings"

	"github.com/nathoo/questcore/types"
)

var directionExpansions = map[string]string{
	"n":    "north",
	"s":    "south",
	"e":    "east",
	"w":    "west",
	"ne":   "northeast",
	"nw":   "northwest",
	"se":   "southeast",
	"sw":   "southwest",
	"up":   "up",
	"down": "down",
	"u":    "up",
	"d":    "down",
}

// Full direction names that are standalone shortcuts for "go <dir>".
var directionNames = map[string]bool{
	"north": true, "south": true, "east": true, "west": true,
	"northeast": true, "northwest": true, "southeast": true, "southwest": true,
	"up": true, "down": true,
}

var verbAliases = map[string]string{
	// Look / Examine
	"l":        "look",
	"x":        "examine",
	"inspect":  "examine",
	"check":    "examine",
	"study":    "examine",
	"observe":  "examine",
	"describe": "examine",
	"search":   "examine",

	// Movement
	"walk":    "go",
	"run":     "go",
	"move":    "go",
	"head":    "go",
	"proceed": "go",
	"enter":   "go",
	"travel":  "go",

	// Take / Get
	"get":   "take",
	"grab":  "take",
	"hold":  "take",
	"carry": "take",
	"catch": "take",

	// Drop
	"discard": "drop",

	// Attack / Combat
	"hit":     "attack",
	"fight":   "attack",
	"strike":  "attack",
	"kill":    "attack",
	"punch":   "attack",
	"kick":    "attack",
	"smash":   "attack",
	"destroy": "attack",
	"break":   "attack",

	// Talk / Dialogue
	"ask":      "talk",
	"speak":    "talk",
	"chat":     "talk",
	"converse": "talk",
	"say":      "talk",
	"tell":     "talk",

	// Open / Close
	"shut": "close",

	// Push / Pull
	"press": "push",
	"shove": "push",
	"shift": "push",
	"drag":  "pull",
	"tug":   "pull",
	"yank":  "pull",

	// Give
	"offer": "give",
	"hand":  "give",
	"feed":  "give",

	// Throw
	"toss": "throw",
	"hurl": "throw",
	"lob":  "throw",

	// Eat / Drink
	"consume": "eat",
	"taste":   "eat",
	"bite":    "eat",
	"devour":  "eat",
	"sip":     "drink",
	"swallow": "drink",
	"quaff":   "drink",

	// Miscellaneous
	"inv":     "inventory",
	"i":       "inventory",
	"z":       "wait",
	"smell":   "smell",
	"sniff":   "smell",
	"listen":  "listen",
	"hear":    "listen",
	"touch":   "touch",
	"feel":    "touch",
	"rub":     "touch",
	"climb":   "climb",
	"scale":   "climb",
	"jump":    "jump",
	"leap":    "jump",
	"hop":     "jump",
	"unlock":  "unlock",
	"tie":     "tie",
	"fasten":  "tie",
	"attach":  "tie",
	"untie":   "untie",
	"detach":  "untie",
	"release": "untie",
	"wear":    "wear",
	"don":     "wear",
	"wave":    "wave",
	"sing":    "sing",
	"pray":    "pray",
	"sleep":   "sleep",
	"nap":     "sleep",
	"rest":    "sleep",
	"knock":   "knock",
	"rap":     "knock",
	"yell":    "yell",
	"scream":  "yell",
	"shout":   "yell",
	"swim":    "swim",
	"dive":    "swim",
	"buy":     "buy",
	"purchase": "buy",
}

var prepositions = map[string]bool{
	"on": true, "at": true, "to": true,
	"with": true, "in": true, "from": true,
	"about": true,
}

var articles = map[string]bool{
	"the": true, "a": true, "an": true,
}

// Parse converts a raw command string into an Intent.
func Parse(input string) types.Intent {
	input = strings.TrimSpace(input)
	if input == "" {
		return types.Intent{}
	}

	words := strings.Fields(strings.ToLower(input))

	// Direction shortcut: bare "n", "south", etc. → go <direction>
	if len(words) == 1 {
		if dir, ok := directionExpansions[words[0]]; ok {
			return types.Intent{Verb: "go", Object: dir}
		}
		if directionNames[words[0]] {
			return types.Intent{Verb: "go", Object: words[0]}
		}
	}

	// Handle multi-word verb phrases before general parsing.
	words = expandMultiWordVerbs(words)

	// Apply verb aliases.
	if alias, ok := verbAliases[words[0]]; ok {
		words[0] = alias
	}

	verb := words[0]
	rest := words[1:]

	// Strip articles ("the", "a", "an").
	rest = stripArticles(rest)

	// Use the first preposition as a delimiter between object and target.
	object, target := splitOnPreposition(rest)

	return types.Intent{
		Verb:   verb,
		Object: object,
		Target: target,
	}
}

// expandMultiWordVerbs handles "look at", "pick up", "talk to" etc.
func expandMultiWordVerbs(words []string) []string {
	if len(words) < 2 {
		return words
	}

	switch words[0] {
	case "look":
		if words[1] == "at" || words[1] == "in" || words[1] == "under" {
			return append([]string{"examine"}, words[2:]...)
		}
	case "pick":
		if words[1] == "up" {
			return append([]string{"take"}, words[2:]...)
		}
	case "talk", "speak", "chat":
		if words[1] == "to" || words[1] == "with" {
			return append([]string{"talk"}, words[2:]...)
		}
	case "put":
		if words[1] == "on" {
			return append([]string{"wear"}, words[2:]...)
		}
		if words[1] == "down" {
			return append([]string{"drop"}, words[2:]...)
		}
	case "take":
		if words[1] == "off" {
			return append([]string{"remove"}, words[2:]...)
		}
	case "turn", "switch":
		if words[1] == "on" {
			return append([]string{"activate"}, words[2:]...)
		}
		if words[1] == "off" {
			return append([]string{"deactivate"}, words[2:]...)
		}
	}

	return words
}

// stripArticles removes articles ("the", "a", "an") from the word list.
func stripArticles(words []string) []string {
	result := make([]string, 0, len(words))
	for _, w := range words {
		if !articles[w] {
			result = append(result, w)
		}
	}
	return result
}

// splitOnPreposition splits words on the first preposition.
// Words before the preposition become the object, words after become the target.
// If no preposition is found, all words become the object.
func splitOnPreposition(words []string) (object, target string) {
	for i, w := range words {
		if prepositions[w] {
			object = strings.Join(words[:i], " ")
			target = strings.Join(words[i+1:], " ")
			return object, target
		}
	}
	return strings.Join(words, " "), ""
}
