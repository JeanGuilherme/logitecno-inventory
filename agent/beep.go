package main

import (
	"fmt"
	"os"
	"time"
)

// Beep emite um sinal sonoro no terminal.
// Usa o caractere BEL (\a) — funciona em qualquer terminal sem dependencia.
// Se o pacote 'beep' estiver instalado, usa ele (mais audivel em hardware antigo).
func beep(times int, duration time.Duration) {
	// Tenta o binario beep primeiro (mais audivel)
	if _, err := os.Stat("/usr/bin/beep"); err == nil {
		for i := 0; i < times; i++ {
			execBeep(duration)
			time.Sleep(120 * time.Millisecond)
		}
		return
	}

	// Fallback: BEL do terminal
	for i := 0; i < times; i++ {
		fmt.Print("\a")
		time.Sleep(120 * time.Millisecond)
	}
}

func execBeep(d time.Duration) {
	// beep -f <freq> -l <ms>
	ms := int(d.Milliseconds())
	cmd := execCommand("beep", "-f", "880", "-l", fmt.Sprintf("%d", ms))
	cmd.Run()
}

// Sinais padrao do agente
func beepSuccess() {
	beep(1, 150*time.Millisecond)
}

func beepQueued() {
	beep(3, 100*time.Millisecond)
}

func beepCritical() {
	beep(2, 500*time.Millisecond)
}