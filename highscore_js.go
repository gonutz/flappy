//go:build js

package main

import (
	"strconv"
	"syscall/js"
)

func saveHighscore(score int) {
	text := strconv.Itoa(score)
	js.Global().Get("localStorage").Call("setItem", "flappy_go_highscore", text)
}

func loadHighscore() int {
	text := js.Global().Get("localStorage").Call("getItem", "flappy_go_highscore").String()
	score, _ := strconv.Atoi(text)
	return score
}
