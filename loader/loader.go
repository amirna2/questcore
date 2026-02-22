package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nathoo/questcore/engine/state"
	lua "github.com/yuin/gopher-lua"
)

// collector accumulates Lua definitions during file execution.
type collector struct {
	game     *lua.LTable
	rooms    []rawRoom
	entities []rawEntity
	rules    []rawRule
	handlers []rawHandler
	order    int
}

func (c *collector) nextSourceOrder() int {
	c.order++
	return c.order
}

// Load reads all .lua files from dir, compiles them into game definitions,
// validates references, and returns the immutable Defs. The Lua VM is
// discarded after loading.
func Load(dir string) (*state.Defs, error) {
	// Discover .lua files.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading game directory %s: %w", dir, err)
	}

	var luaFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".lua") {
			luaFiles = append(luaFiles, e.Name())
		}
	}
	if len(luaFiles) == 0 {
		return nil, fmt.Errorf("no .lua files found in %s", dir)
	}

	// Sort: game.lua first, rest alphabetical.
	luaFiles = sortedLuaFiles(luaFiles)

	// Create sandboxed VM.
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer L.Close()

	// Open safe libs only.
	openSafeLibs(L)

	// Sandbox: remove dangerous globals.
	sandbox(L)

	// Register API.
	coll := &collector{}
	registerAPI(L, coll)

	// Execute each file.
	for _, f := range luaFiles {
		path := filepath.Join(dir, f)
		if err := L.DoFile(path); err != nil {
			return nil, fmt.Errorf("executing %s: %w", f, err)
		}
	}

	// Compile.
	defs, err := compile(coll)
	if err != nil {
		return nil, fmt.Errorf("compiling game data: %w", err)
	}

	// Validate.
	if err := validate(defs); err != nil {
		return nil, err
	}

	return defs, nil
}

// openSafeLibs opens only the safe subset of Lua standard libraries.
func openSafeLibs(L *lua.LState) {
	// Base library (print, type, tostring, tonumber, pairs, ipairs, etc.)
	lua.OpenBase(L)
	// Table library (table.insert, table.sort, etc.)
	lua.OpenTable(L)
	// String library (string.format, string.sub, etc.)
	lua.OpenString(L)
	// Math library (math.floor, math.max, etc.)
	lua.OpenMath(L)
}

// sandbox removes dangerous globals and functions.
func sandbox(L *lua.LState) {
	// Remove dangerous base globals.
	dangerous := []string{
		"dofile", "loadfile", "load", "loadstring",
		"rawset", "rawget", "rawequal",
		"collectgarbage",
	}
	for _, name := range dangerous {
		L.SetGlobal(name, lua.LNil)
	}

	// Remove math.randomseed to preserve determinism.
	if mathTbl := L.GetGlobal("math"); mathTbl != lua.LNil {
		if tbl, ok := mathTbl.(*lua.LTable); ok {
			tbl.RawSetString("randomseed", lua.LNil)
		}
	}
}
