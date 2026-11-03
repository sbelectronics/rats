package main

// platform_hooks.go - hooks that other parts set.
//
// GameMainFn is the game's entry point (RatsA's Main).  game_*.go sets it in an init():
//
//	func init() { GameMainFn = GameMain }
//
// main.go runs GameMainFn in its own goroutine.  If it is nil, main.go shows a
// test pattern (all 256 glyphs with every attribute) instead.
var GameMainFn func()
