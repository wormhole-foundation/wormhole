package main

import "sync"

type TestStruct struct {
	mu   sync.Mutex
	data map[string]string
}

// Good: Function with proper locking
func (t *TestStruct) goodFunction() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// This should NOT trigger the rule because we have proper locking
	t.processDataAlreadyLocked("key")
}

// Bad: Function without proper locking
func (t *TestStruct) badFunction() {
	// This SHOULD trigger the rule because no locking
	t.processDataAlreadyLocked("key")
}

// Good: Function with explicit lock/unlock
func (t *TestStruct) anotherGoodFunction() {
	t.mu.Lock()
	t.updateCacheAlreadyLocked("value")
	t.mu.Unlock()
}

// Helper functions that should be called with locks
func (t *TestStruct) processDataAlreadyLocked(key string) {
	t.data[key] = "processed"
}

func (t *TestStruct) updateCacheAlreadyLocked(value string) {
	t.data["cache"] = value
}
