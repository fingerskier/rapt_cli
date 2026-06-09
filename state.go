package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// State is persisted to the user's config dir so rapt can remember
// reading speed and where the user left off in each file.
type State struct {
	WPM      int            `json:"wpm"`
	LastFile string         `json:"last_file,omitempty"`
	Files    map[string]int `json:"files"`
}

func statePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "rapt", "state.json"), nil
}

func loadState() *State {
	st := &State{Files: map[string]int{}}
	path, err := statePath()
	if err != nil {
		return st
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return st
	}
	if err := json.Unmarshal(data, st); err != nil {
		return &State{Files: map[string]int{}}
	}
	if st.Files == nil {
		st.Files = map[string]int{}
	}
	return st
}

func (s *State) save() error {
	path, err := statePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
