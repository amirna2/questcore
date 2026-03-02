Game {
    title   = "Combat Test",
    author  = "Test",
    version = "0.1.0",
    start   = "cave",
    player_stats = {
        hp = 20, max_hp = 20,
        attack = 5, defense = 2,
    },
}

Room "cave" {
    description = "A dark cave.",
    exits = { south = "entrance" },
}

Room "entrance" {
    description = "The entrance.",
    exits = { north = "cave" },
}

Item "goblin_blade" {
    name        = "Rusty Goblin Blade",
    description = "A crudely forged blade.",
    location    = "cave",
    takeable    = true,
}

Enemy "cave_goblin" {
    name        = "Cave Goblin",
    description = "A snarling goblin clutching a rusty blade.",
    location    = "cave",
    stats = {
        hp = 12, max_hp = 12,
        attack = 4, defense = 1,
    },
    behavior = {
        { action = "attack", weight = 70 },
        { action = "defend", weight = 20 },
        { action = "flee",   weight = 10 },
    },
    loot = {
        items = { { id = "goblin_blade", chance = 50 } },
        gold  = 5,
    },
}

Rule("attack_goblin",
    When { verb = "attack", object = "cave_goblin" },
    Then {
        Say("You engage the Cave Goblin in combat!"),
        StartCombat("cave_goblin"),
    }
)
