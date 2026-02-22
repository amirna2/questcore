local examine_painting = Rule("examine_painting",
    When { verb = "examine", object = "painting" },
    Then { Say("A beautiful landscape.") }
)

Room "entrance" {
    description = "The castle entrance. A painting hangs on the wall.",
    exits = {
        north = "throne_room",
        east = "garden"
    },
    fallbacks = {
        push = "Nothing here to push."
    },
    rules = { examine_painting }
}

Room "throne_room" {
    description = "A grand throne room.",
    exits = {
        south = "entrance"
    }
}

Room "garden" {
    description = "A peaceful garden.",
    exits = {
        west = "entrance"
    }
}
