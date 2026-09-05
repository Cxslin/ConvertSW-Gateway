package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type ComboModel struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Strategy    string   `json:"strategy"` // "fallback" or "round_robin"
	Targets     []string `json:"targets"`  // format: "provider:model", e.g. "deepseek:deepseek-reasoner"
	Enabled     bool     `json:"enabled"`
}

type ComboStore struct {
	filePath string
	mu       sync.RWMutex
	combos   map[string]ComboModel
	order    []string
	rrIndex  map[string]int
}

func defaultCombos() []ComboModel {
	return []ComboModel{
		{
			ID:          "combo-auto",
			Name:        "Combo Auto (Smart Fallback)",
			Description: "Rute otomatis cerdas: DeepSeek R1 -> ChatGPT -> Qwen Max -> Mistral Large",
			Strategy:    "fallback",
			Targets: []string{
				"deepseek:deepseek-reasoner",
				"chatgpt:gpt-5-6",
				"qwen:qwen-max",
				"mistral:mistral-large",
			},
			Enabled: true,
		},
		{
			ID:          "combo-coding",
			Name:        "Combo Coding & Agent",
			Description: "Optimal untuk AI Agent (Cline, Roo Code, Cursor): DeepSeek R1 -> ChatGPT -> Qwen 2.5 72B -> Mistral",
			Strategy:    "fallback",
			Targets: []string{
				"deepseek:deepseek-reasoner",
				"chatgpt:gpt-5-6",
				"qwen:qwen2.5-72b-instruct",
				"mistral:mistral-large",
			},
			Enabled: true,
		},
		{
			ID:          "combo-fast",
			Name:        "Combo Fast (High Speed)",
			Description: "Respons secepat kilat: Mistral Chat -> Qwen Turbo -> ChatGPT -> DeepSeek Chat",
			Strategy:    "fallback",
			Targets: []string{
				"mistral:mistral-chat",
				"qwen:qwen-turbo",
				"chatgpt:gpt-4o-mini",
				"deepseek:deepseek-chat",
			},
			Enabled: true,
		},
		{
			ID:          "combo-reasoning",
			Name:        "Combo Deep Reasoning",
			Description: "Penalaran matematika & logika mendalam: DeepSeek Reasoner -> ChatGPT -> Qwen Max",
			Strategy:    "fallback",
			Targets: []string{
				"deepseek:deepseek-reasoner",
				"chatgpt:gpt-5-6",
				"qwen:qwen-max",
			},
			Enabled: true,
		},
	}
}

func NewComboStore(filePath string) *ComboStore {
	cs := &ComboStore{
		filePath: filePath,
		combos:   make(map[string]ComboModel),
		order:    make([]string, 0),
		rrIndex:  make(map[string]int),
	}
	_ = cs.Load()
	return cs
}

func (cs *ComboStore) Load() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, err := os.Stat(cs.filePath); os.IsNotExist(err) {
		// Initialize with default combos
		_ = os.MkdirAll(filepath.Dir(cs.filePath), 0755)
		defaults := defaultCombos()
		cs.combos = make(map[string]ComboModel)
		cs.order = make([]string, 0, len(defaults))
		for _, c := range defaults {
			cs.combos[c.ID] = c
			cs.order = append(cs.order, c.ID)
		}
		return cs.saveLocked()
	}

	data, err := os.ReadFile(cs.filePath)
	if err != nil {
		return err
	}

	var list []ComboModel
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	cs.combos = make(map[string]ComboModel)
	cs.order = make([]string, 0, len(list))
	for _, c := range list {
		cs.combos[c.ID] = c
		cs.order = append(cs.order, c.ID)
	}

	// Ensure default combos exist if not present
	for _, def := range defaultCombos() {
		if _, exists := cs.combos[def.ID]; !exists {
			cs.combos[def.ID] = def
			cs.order = append(cs.order, def.ID)
		}
	}

	return nil
}

func (cs *ComboStore) Save() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.saveLocked()
}

func (cs *ComboStore) saveLocked() error {
	list := make([]ComboModel, 0, len(cs.order))
	for _, id := range cs.order {
		if c, ok := cs.combos[id]; ok {
			list = append(list, c)
		}
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cs.filePath, data, 0644)
}

func (cs *ComboStore) List() []ComboModel {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	list := make([]ComboModel, 0, len(cs.order))
	for _, id := range cs.order {
		if c, ok := cs.combos[id]; ok {
			list = append(list, c)
		}
	}
	return list
}

func (cs *ComboStore) Get(id string) (ComboModel, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	c, ok := cs.combos[id]
	return c, ok
}

func (cs *ComboStore) Set(c ComboModel) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if strings.TrimSpace(c.ID) == "" {
		return fmt.Errorf("combo model ID cannot be empty")
	}

	if _, exists := cs.combos[c.ID]; !exists {
		cs.order = append(cs.order, c.ID)
	}
	cs.combos[c.ID] = c
	return cs.saveLocked()
}

func (cs *ComboStore) Delete(id string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	delete(cs.combos, id)
	newOrder := make([]string, 0, len(cs.order))
	for _, oid := range cs.order {
		if oid != id {
			newOrder = append(newOrder, oid)
		}
	}
	cs.order = newOrder
	return cs.saveLocked()
}

// ResolveTargets resolves an ordered list of targets to attempt for a given model.
// Returns: targets (slice of "provider:model"), isCombo (bool)
func (cs *ComboStore) ResolveTargets(modelName string) ([]string, bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	combo, ok := cs.combos[modelName]
	if !ok || !combo.Enabled || len(combo.Targets) == 0 {
		return nil, false
	}

	if combo.Strategy == "round_robin" && len(combo.Targets) > 1 {
		idx := cs.rrIndex[modelName] % len(combo.Targets)
		cs.rrIndex[modelName] = (idx + 1) % len(combo.Targets)
		// Return with rotated priority
		rotated := make([]string, len(combo.Targets))
		for i := 0; i < len(combo.Targets); i++ {
			rotated[i] = combo.Targets[(idx+i)%len(combo.Targets)]
		}
		return rotated, true
	}

	// Default fallback: return in defined order
	out := make([]string, len(combo.Targets))
	copy(out, combo.Targets)
	return out, true
}
