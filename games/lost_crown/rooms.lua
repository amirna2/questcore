Room "castle_gates" {
    description = "You stand before the imposing castle gates. Tall stone walls stretch in both directions, and iron-studded oak doors stand open ahead. A weathered guard post sits to one side.",
    exits = {
        north = "great_hall"
    },
    fallbacks = {
        open = "The gates are already open.",
        close = "You have no authority to close the castle gates."
    }
}

Room "great_hall" {
    description = "The great hall stretches before you, its vaulted ceiling lost in shadow. Faded tapestries line the walls, depicting battles long forgotten. A massive fireplace dominates the north wall.",
    exits = {
        south = "castle_gates",
        east  = "library",
        north = "throne_room",
        west  = "armory"
    }
}

Room "throne_room" {
    description = "The throne room is grand but somber. The gilded throne sits empty on a raised dais. A velvet cushion where the crown once rested shows only a circular imprint in the dust.",
    exits = {
        south = "great_hall",
        east  = "tower_stairs"
    },
    fallbacks = {
        take = "Everything in the throne room belongs to the king."
    }
}

Room "library" {
    description = "Floor-to-ceiling shelves overflow with leather-bound books and scrolls. A reading desk sits beneath a narrow window. Dust motes dance in the thin beam of light.",
    exits = {
        west = "great_hall"
    }
}

Room "armory" {
    description = "Racks of weapons line the walls — swords, maces, and crossbows, all well-maintained. A locked display case in the corner catches your eye.",
    exits = {
        east = "great_hall"
    }
}

Room "tower_stairs" {
    description = "A narrow spiral staircase winds upward, the stone steps worn smooth by centuries of footsteps. A cold draft seeps down from above.",
    exits = {
        west  = "throne_room",
        up    = "tower_top"
    }
}

Room "tower_top" {
    description = "You emerge onto the tower parapet. The kingdom spreads below — rolling hills, a dark forest to the east, and the glimmer of a river to the south. The wind whips at your cloak.",
    exits = {
        down = "tower_stairs"
    }
}

Room "secret_passage" {
    description = "A narrow, torch-lit passage stretches ahead. The air is damp and smells of earth. Cobwebs brush your face as you move forward. At the far end, a small chamber holds a stone pedestal.",
    exits = {
        south = "library"
    }
}
