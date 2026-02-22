Rule("take_gem",
    When { verb = "take", object = "gem" },
    { HasItem("rusty_key") },
    Then {
        Say("You pry the gem loose with the rusty key."),
        GiveItem("gem"),
        SetFlag("gem_taken", true),
        IncCounter("treasures", 1)
    }
)

Rule("take_gem_fail",
    When { verb = "take", object = "gem" },
    Then {
        Say("The gem is firmly embedded. You need a tool."),
        Stop()
    }
)

Rule("unlock_door",
    When { verb = "use", object = "rusty_key", target = "door" },
    Then {
        Say("You unlock the door."),
        RemoveItem("rusty_key"),
        OpenExit("entrance", "south", "garden"),
        EmitEvent("door_unlocked")
    }
)

On("door_unlocked", {
    conditions = { InRoom("entrance") },
    effects = { Say("You hear a click as the door unlocks.") }
})
