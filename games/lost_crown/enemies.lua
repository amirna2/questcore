Enemy "cave_goblin" {
    name        = "Cave Goblin",
    description = "A snarling goblin clutching a rusty blade. Its beady eyes gleam with malice.",
    location    = "secret_passage",
    stats = {
        hp      = 12,
        max_hp  = 12,
        attack  = 4,
        defense = 1,
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

Item "goblin_blade" {
    name        = "Rusty Goblin Blade",
    description = "A crudely forged blade, chipped and stained. Still sharp enough to cut.",
    location    = "secret_passage",
    takeable    = false,
}

-- Engaging the goblin: attack command triggers combat.
Rule("attack_goblin",
    When { verb = "attack", object = "cave_goblin" },
    { InRoom("secret_passage"), PropIs("cave_goblin", "alive", true) },
    Then {
        Say("You engage the Cave Goblin in combat!"),
        StartCombat("cave_goblin"),
    }
)

-- Auto-engage: goblin attacks when player enters the passage.
On("room_entered", {
    conditions = {
        InRoom("secret_passage"),
        PropIs("cave_goblin", "alive", true),
        Not(InCombat()),
    },
    effects = {
        Say("A Cave Goblin blocks your path!"),
        StartCombat("cave_goblin"),
    }
})

-- After defeating the goblin.
On("enemy_defeated", {
    conditions = { PropIs("cave_goblin", "alive", false) },
    effects = {
        Say("The goblin crumples to the ground."),
    }
})
