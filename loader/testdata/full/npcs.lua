NPC "guard" {
    name = "castle guard",
    description = "A stern-looking guard.",
    location = "throne_room",
    topics = {
        greet = {
            text = "Hello, traveler.",
            effects = { SetFlag("met_guard", true) }
        },
        quest = {
            text = "Find the lost crown in the garden.",
            requires = { FlagSet("met_guard") },
            effects = { SetFlag("quest_given", true) }
        }
    }
}

Entity "painting" {
    name = "painting",
    description = "A large oil painting.",
    location = "entrance"
}
