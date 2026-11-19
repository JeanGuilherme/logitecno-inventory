package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// queuePath é onde a fila local é gravada.
// No live environment, aponta para a partição gravável do pendrive.
func queuePath() string {
	if v := os.Getenv("LOGITECNO_QUEUE_PATH"); v != "" {
		return v
	}
	return "/var/lib/logitecno/queue.jsonl"
}

// queueAppend adiciona o inventário como uma linha JSON (JSONL).
// Append-only: nunca reescreve o arquivo, então não corrompe em queda de energia.
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

// queueCount retorna quantos registros existem na fila.
// Útil para o resumo na tela do operador.
func queueCount() int {
	path := queuePath()
	b, err := os.ReadFile(path)
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

// queueFlush tenta enviar tudo da fila. Chamado quando há rede.
// Retorna quantos foram enviados com sucesso.
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

	// Reescreve só o que sobrou
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

// touch para evitar import não usado em versões futuras
var _ = time.Now