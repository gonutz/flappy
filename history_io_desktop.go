//go:build !js

package main

import (
	"os"
	"path/filepath"
	"runtime"
)

func historyPath() string {
	return filepath.Join(historyDir(), "flappy_go_history")
}

func historyDir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("APPDATA")
	}

	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}

	return "."
}

func saveKillHistory(kills []kill) {
	os.WriteFile(historyPath(), killsToBytes(kills), 0666)
}

func loadKillHistory() []kill {
	data, err := os.ReadFile(historyPath())
	if err != nil {
		return nil
	}

	return bytesToKills(data)
}
