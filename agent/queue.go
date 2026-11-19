package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// queuePath resolve onde a fila sera gravada.
// Ordem de preferencia:
//   1. LOGITECNO_QUEUE_PATH (variavel de ambiente)
//   2. /mnt/dados/queue.jsonl (particao gravavel do pendrive)
//   3. /tmp/logitecno/queue.jsonl (fallback volatil, ultimo recurso)
func queuePath() string {
	if v := os.Getenv("LOGITECNO_QUEUE_PATH"); v != "" {
		return v
	}

	// Particao de dados do pendrive
	dataDir := "/mnt/dados"
	if st, err := os.Stat(dataDir); err == nil && st.IsDir() {
		// Testa se e gravavel de verdade
		testFile := filepath.Join(dataDir, ".write-test")
		if err := os.WriteFile(testFile, []byte("ok"), 0644); err == nil {
			os.Remove(testFile)
			return filepath.Join(dataDir, "queue.jsonl")
		}
	}

	// Fallback volatil — avisa que vai perder
	return "/tmp/logitecno/queue.jsonl"
}

func queueAppend(inv Inventory) {
	path := queuePath()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		inv.Errors = append(inv.Errors, "queue mkdir: "+err.Error())
		return
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		inv.Errors = append(inv.Errors, "queue open: "+err.Error())
		return
	}
	defer f.Close()

	line, err := json.Marshal(inv)
	if err != nil {
		return
	}
	line = append(line, '\n')

	if _, err := f.Write(line); err != nil {
		return
	}
	f.Sync()
}

func queueCount() int {
	b, err := os.ReadFile(queuePath())
	if err != nil {
		return 0
	}
	count := 0
	for _, c := range b {
		if c == '\n' {
			count++
		}
	}
	return count
}

func queueFlush() int {
	path := queuePath()
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	lines := splitLines(b)
	sent := 0
	var remaining []byte

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var inv Inventory
		if err := json.Unmarshal(line, &inv); err != nil {
			continue
		}
		if err := sendInventory(inv); err != nil {
			remaining = append(remaining, line...)
			remaining = append(remaining, '\n')
		} else {
			sent++
		}
	}

	if len(remaining) == 0 {
		os.Remove(path)
	} else {
		os.WriteFile(path, remaining, 0644)
	}

	return sent
}

func splitLines(b []byte) [][]byte {
	var out [][]byte
	start := 0
	for i, c := range b {
		if c == '\n' {
			out = append(out, b[start:i])
			start = i + 1
		}
	}
	if start < len(b) {
		out = append(out, b[start:])
	}
	return out
}

// queueIsPersistent informa se a fila esta em local que sobrevive ao reboot.
func queueIsPersistent() bool {
	return filepath.Dir(queuePath()) == "/mnt/dados"
}

var _ = time.Now