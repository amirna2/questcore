-- Reading the old book reveals a clue about the passage.
Rule("read_old_book",
    When { verb = "read", object = "old_book" },
    { HasItem("old_book") },
    Then {
        Say("You read the dog-eared page carefully. It describes a hidden passage behind the north wall of the library, activated by pressing a specific stone."),
        SetFlag("found_book_clue", true)
    }
)

Rule("read_old_book_no_have",
    When { verb = "read", object = "old_book" },
    Then {
        Say("You'd need to pick it up first.")
    }
)

-- Reading the quest scroll.
Rule("read_quest_scroll",
    When { verb = "read", object = "quest_scroll" },
    { HasItem("quest_scroll") },
    Then {
        Say("The scroll reads: 'Find my crown. The thief was last seen near the library. Trust no one.' It bears the king's seal.")
    }
)

Rule("read_quest_scroll_no_have",
    When { verb = "read", object = "quest_scroll" },
    Then {
        Say("You'd need to pick it up first.")
    }
)

-- Examining the diagram/page mentioned in the old book.
Rule("examine_diagram",
    When { verb = "examine", object = "diagram" },
    { HasItem("old_book") },
    Then {
        Say("The diagram in the old book shows a detailed layout of the library's north wall. One stone is circled and marked with an arrow — it looks like a pressure mechanism.")
    }
)

Rule("examine_page",
    When { verb = "examine", object = "page" },
    { HasItem("old_book") },
    Then {
        Say("The dog-eared page shows a hand-drawn diagram of the library's north wall, with one stone carefully marked. Below it, someone has written: 'Third from the left.'")
    }
)

-- Opening the secret passage in the library.
Rule("push_wall_library",
    When { verb = "push", object = "wall" },
    { InRoom("library"), FlagSet("knows_passage") },
    Then {
        Say("You press the third stone from the left. With a grinding rumble, a section of the wall slides away, revealing a dark passage leading north!"),
        OpenExit("library", "north", "secret_passage"),
        SetFlag("passage_open", true),
        EmitEvent("passage_opened")
    }
)

Rule("push_wall_no_knowledge",
    When { verb = "push", object = "wall" },
    { InRoom("library") },
    Then {
        Say("You push against the wall, but nothing happens. Perhaps you need to know exactly where to push.")
    }
)

-- Taking the silver dagger requires the rusty key (to unlock the case).
Rule("take_dagger_with_key",
    When { verb = "use", object = "rusty_key", target = "silver_dagger" },
    { HasItem("rusty_key"), InRoom("armory") },
    Then {
        Say("You insert the rusty key into the display case lock. It turns with a click! You carefully lift the silver dagger from its velvet rest."),
        RemoveItem("rusty_key"),
        SetProp("silver_dagger", "takeable", true),
        GiveItem("silver_dagger"),
        SetFlag("case_unlocked", true)
    }
)

-- Unlock/open the display case directly.
Rule("unlock_case",
    When { verb = "unlock", object = "case" },
    { HasItem("rusty_key"), InRoom("armory") },
    Then {
        Say("You insert the rusty key into the display case lock. It turns with a click! You carefully lift the silver dagger from its velvet rest."),
        RemoveItem("rusty_key"),
        SetProp("silver_dagger", "takeable", true),
        GiveItem("silver_dagger"),
        SetFlag("case_unlocked", true)
    }
)

Rule("open_case_locked",
    When { verb = "open", object = "case" },
    { InRoom("armory") },
    Then {
        Say("The display case is locked. You'd need a key.")
    }
)

Rule("examine_case",
    When { verb = "examine", object = "case" },
    { InRoom("armory"), FlagNot("case_unlocked") },
    Then {
        Say("A locked glass display case. Inside, a silver dagger gleams on a velvet cushion. The lock looks like it takes a small key.")
    }
)

Rule("examine_case_unlocked",
    When { verb = "examine", object = "case" },
    { InRoom("armory"), FlagSet("case_unlocked") },
    Then {
        Say("The display case stands open, its lock hanging loose. The velvet cushion inside is empty.")
    }
)

-- Taking the crown — the climax.
Rule("take_crown",
    When { verb = "take", object = "lost_crown" },
    { InRoom("secret_passage") },
    Then {
        Say("You lift the Lost Crown from the stone pedestal. It pulses with a warm golden light in your hands. The kingdom is saved!"),
        GiveItem("lost_crown"),
        SetFlag("crown_found", true),
        IncCounter("score", 100),
        EmitEvent("crown_recovered")
    }
)

-- Using the spyglass at the tower top.
Rule("use_spyglass",
    When { verb = "use", object = "spyglass" },
    { HasItem("spyglass"), InRoom("tower_top") },
    Then {
        Say("Through the spyglass, you survey the kingdom. To the east, you notice a faint light flickering in the forest — perhaps another adventure for another day. Below, in the castle courtyard, guards patrol in orderly lines."),
        IncCounter("score", 10)
    }
)

-- Examining the throne reveals the missing crown's imprint.
Rule("examine_throne",
    When { verb = "examine", object = "throne" },
    { InRoom("throne_room") },
    Then {
        Say("The gilded throne is magnificent but empty. On the velvet cushion, a circular imprint marks where the crown once sat. A faint scratch on the armrest suggests someone pried it away in haste.")
    }
)

-- Throne room scenery.
Rule("examine_cushion",
    When { verb = "examine", object = "cushion" },
    { InRoom("throne_room") },
    Then {
        Say("The velvet cushion bears a circular imprint in the dust where the crown once rested. Someone removed it recently — the dust hasn't had time to settle back.")
    }
)

Rule("examine_dais",
    When { verb = "examine", object = "dais" },
    { InRoom("throne_room") },
    Then {
        Say("The raised stone dais elevates the throne above the rest of the room. Its edges are worn smooth by centuries of supplicants kneeling before the king.")
    }
)

-- Great hall scenery.
Rule("examine_fireplace",
    When { verb = "examine", object = "fireplace" },
    { InRoom("great_hall") },
    Then {
        Say("The massive stone fireplace is cold and dark. Ashes from a long-dead fire sit in the grate. Above the mantel, a faded coat of arms is carved into the stone.")
    }
)

Rule("examine_tapestries",
    When { verb = "examine", object = "tapestries" },
    { InRoom("great_hall") },
    Then {
        Say("The faded tapestries depict ancient battles — knights charging on horseback, a castle under siege, and a crowned king raising a sword in victory. The colors have dimmed with age, but the craftsmanship is remarkable.")
    }
)

-- Library scenery.
Rule("examine_shelves",
    When { verb = "examine", object = "shelves" },
    { InRoom("library") },
    Then {
        Say("The shelves are packed with leather-bound volumes, loose scrolls, and the occasional curiosity — a brass compass, a dried flower pressed between pages, a cracked magnifying glass. Most titles are too faded to read.")
    }
)

Rule("examine_desk",
    When { verb = "examine", object = "desk" },
    { InRoom("library") },
    Then {
        Say("A sturdy reading desk beneath the window. Its surface is scarred with ink stains and the scratches of countless quills. A few loose pages lie scattered across it, but nothing of interest.")
    }
)

Rule("examine_window_library",
    When { verb = "examine", object = "window" },
    { InRoom("library") },
    Then {
        Say("A narrow window letting in a thin beam of light. Through it, you can see the castle courtyard far below.")
    }
)

-- Castle gates scenery.
Rule("examine_gates",
    When { verb = "examine", object = "gates" },
    { InRoom("castle_gates") },
    Then {
        Say("Massive iron-studded oak doors, standing open. The iron is pitted with age but the hinges are well-oiled. These gates have held against many a siege.")
    }
)

Rule("examine_guard_post",
    When { verb = "examine", object = "guard post" },
    { InRoom("castle_gates") },
    Then {
        Say("A small stone shelter for the gate watch. Inside, a stool, a lantern, and a half-eaten loaf of bread. The captain must have been here recently.")
    }
)

-- Armory scenery.
Rule("examine_weapons",
    When { verb = "examine", object = "weapons" },
    { InRoom("armory") },
    Then {
        Say("Racks of swords, maces, and crossbows line the walls, all polished and ready for battle. These are not for adventurers to take — they belong to the castle guard.")
    }
)

Rule("examine_swords",
    When { verb = "examine", object = "swords" },
    { InRoom("armory") },
    Then {
        Say("Fine steel swords, each one engraved with the royal crest. They're racked securely and meant for the castle guard.")
    }
)

-- Tower scenery.
Rule("examine_stairs",
    When { verb = "examine", object = "staircase" },
    { InRoom("tower_stairs") },
    Then {
        Say("The spiral staircase is ancient, its stone steps worn into shallow grooves by countless feet over the centuries. The walls are bare stone, cold to the touch.")
    }
)

Rule("examine_parapet",
    When { verb = "examine", object = "parapet" },
    { InRoom("tower_top") },
    Then {
        Say("The stone parapet is chest-high, with crenellations offering a commanding view in every direction. The wind is fierce up here.")
    }
)

-- Secret passage scenery.
Rule("examine_pedestal",
    When { verb = "examine", object = "pedestal" },
    { InRoom("secret_passage"), FlagNot("crown_found") },
    Then {
        Say("A weathered stone pedestal, carved with ancient symbols. Atop it, the Lost Crown gleams with a faint golden light.")
    }
)

Rule("examine_pedestal_empty",
    When { verb = "examine", object = "pedestal" },
    { InRoom("secret_passage"), FlagSet("crown_found") },
    Then {
        Say("The stone pedestal stands empty. The ancient symbols carved into its surface seem to have dimmed.")
    }
)

Rule("examine_cobwebs",
    When { verb = "examine", object = "cobwebs" },
    { InRoom("secret_passage") },
    Then {
        Say("Thick cobwebs hang from the ceiling and walls. No one has been down here in a very long time.")
    }
)

-- Winning the game.
On("crown_recovered", {
    effects = {
        Say(""),
        Say("=== CONGRATULATIONS ==="),
        Say("You have recovered the Lost Crown! The kingdom rejoices."),
        Say("Final score: {score} points.")
    }
})

-- Passage opening sound effect.
On("passage_opened", {
    conditions = { InRoom("library") },
    effects = { Say("A cold draft rushes out from the darkness beyond.") }
})
