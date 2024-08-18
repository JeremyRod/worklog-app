package main

// create logger implementation.
// Probably just a os.Stdout logger to show when entries are saved.
// This will be issue if tui is gonna be used for entering logs.
// Look at using Zerolog and zerologr log sink
// Dont think I would need lumberjack since no file logging?
// Unless traceablity is wanted about entries and timing
