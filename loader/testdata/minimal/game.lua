Game {
    title = "Minimal Test Game",
    author = "Test",
    version = "1.0",
    start = "hall",
    intro = "Welcome!"
}

Room "hall" {
    description = "A grand hall.",
    exits = { north = "hall" }
}
