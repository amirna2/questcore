NPC "captain" {
    name = "Captain Aldric",
    description = "The captain of the guard. His weathered face is creased with worry, and his hand rests on the pommel of his sword.",
    location = "castle_gates",
    topics = {
        greet = {
            text = "Captain Aldric nods gravely. 'Adventurer. The king awaits you in the throne room. The crown's disappearance has shaken us all.'",
            effects = { SetFlag("met_captain", true) }
        },
        crown = {
            text = "'The crown vanished three nights ago. No signs of forced entry. I've doubled the guard, but... I fear the thief knows the castle well.'",
            requires = { FlagSet("met_captain") }
        },
        passage = {
            text = "'A secret passage? Hmm. There are old rumors about hidden tunnels, built during the siege wars. If anyone would know, it'd be the court scholar.'",
            requires = { FlagSet("found_book_clue") }
        }
    }
}

NPC "scholar" {
    name = "Scholar Elara",
    description = "An elderly woman in ink-stained robes, surrounded by stacks of parchment. Her sharp eyes miss nothing.",
    location = "library",
    topics = {
        greet = {
            text = "'Ah, the adventurer. I wondered when they'd send someone competent. The answer lies in the books, as it always does.'",
            effects = { SetFlag("met_scholar", true) }
        },
        crown = {
            text = "'The crown is protected by old enchantments. Whoever took it must have known how to bypass them. Look for clues in the histories.'",
            requires = { FlagSet("met_scholar") }
        },
        book = {
            text = "'That old book? It's been here for centuries. A record of the castle's construction — including its hidden passages. Read it carefully, then come talk to me.'",
            requires = { FlagSet("met_scholar") }
        },
        secret = {
            text = "'Secrets? This castle is full of them. The old histories speak of hidden passages built during the siege wars. You might find clues in the books on these shelves.'",
            requires = { FlagSet("met_scholar") }
        },
        passage = {
            text = "'You found the book! Yes, there is a hidden passage behind the north wall. Push the third stone from the left, and the wall will open.'",
            requires = { HasItem("old_book"), FlagSet("met_scholar") },
            effects = {
                SetFlag("knows_passage", true),
                Say("Scholar Elara marks the location on your quest scroll.")
            }
        }
    }
}
