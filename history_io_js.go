//go:build js

package main

import "syscall/js"

const historyName = "flappy_go_history"

func saveKillHistory(kills []kill) {
	text := string(killsToBytes(kills))
	js.Global().Get("localStorage").Call("setItem", historyName, text)
}

func loadKillHistory() []kill {
	text := js.Global().Get("localStorage").Call("getItem", historyName).String()
	return bytesToKills([]byte(text))
}
