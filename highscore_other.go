//go:build !windows && !js

package main

import (
	"os"
	"path/filepath"
	"strconv"
)

func highscorePath() string {
	dir := "."

	if exe, err := os.Executable(); err == nil {
		dir = filepath.Dir(exe)
	}

	return filepath.Join(dir, "flappy_go_highscore")
}

func saveHighscore(score int) {
	text := strconv.Itoa(score)
	os.WriteFile(highscorePath(), []byte(text), 0666)
}

func loadHighscore() int {
	data, err := os.ReadFile(highscorePath())
	if err != nil {
		return 0
	}

	score, _ := strconv.Atoi(string(data))
	return score
}
