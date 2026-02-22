Rule("same_id",
    When { verb = "look" },
    Then { Say("First rule.") }
)

Rule("same_id",
    When { verb = "examine" },
    Then { Say("Second rule.") }
)
