package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

const inputTimeout = 30 * time.Second

// readWithTimeout le uma linha do stdin com timeout.
// Mostra contador regressivo na tela.
// Retorna o texto digitado (trim) e se houve timeout.
func readWithTimeout(label string, timeout time.Duration) (string, bool) {
	fmt.Printf("\n  %s\n", label)
	fmt.Printf("  (Enter para confirmar, ou aguarde %ds para pular)\n", int(timeout.Seconds()))

	resultCh := make(chan string, 1)

	go func() {
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		resultCh <- strings.TrimSpace(line)
	}()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Contador regressivo na mesma linha
	fmt.Print("  Tempo: ")
	for {
		select {
		case val := <-resultCh:
			fmt.Print("\r                              \r")
			return val, false
		case <-ticker.C:
			remaining := int(time.Until(deadline).Seconds())
			if remaining <= 0 {
				fmt.Print("\r                              \r")
				return "", true
			}
			fmt.Printf("\r  Tempo: %2ds restantes ", remaining)
		}
	}
}

// collectManualInput coleta os dois IDs com timeout individual.
// Ambos sao opcionais — se o operador nao digitar, o campo fica vazio.
func collectManualInput(inv *Inventory) {
	printHeader("IDENTIFICACAO MSCM")
	fmt.Println("  Os campos abaixo sao OPCIONAIS.")
	fmt.Println("  Se nao souber, apenas aguarde o tempo esgotar.")

	// Campo 1 — ID MSCM
	idMSCM, timedOut := readWithTimeout("ID MSCM:", inputTimeout)
	if timedOut {
		fmt.Println("  [tempo esgotado — ID MSCM nao informado]")
	} else if idMSCM != "" {
		inv.IDMSCM = idMSCM
		fmt.Printf("  [ID MSCM: %s]\n", idMSCM)
	} else {
		fmt.Println("  [pulado]")
	}

	// Campo 2 — ID Fornecedor
	idForn, timedOut := readWithTimeout("ID Fornecedor:", inputTimeout)
	if timedOut {
		fmt.Println("  [tempo esgotado — ID Fornecedor nao informado]")
	} else if idForn != "" {
		inv.IDFornecedor = idForn
		fmt.Printf("  [ID Fornecedor: %s]\n", idForn)
	} else {
		fmt.Println("  [pulado]")
	}
}

// printHeader imprime um cabecalho com box drawing.
func printHeader(title string) {
	line := strings.Repeat("=", 60)
	fmt.Printf("\n%s\n  %s\n%s\n", line, title, line)
}

// printSummary mostra o resumo final na tela do operador.
// printSummary mostra o resumo final na tela do operador.
func printSummary(inv *Inventory, sent bool) {
	printHeader("RESUMO DA COLETA")

	fmt.Printf("  Serial:       %s\n", orNA(inv.Serial))
	fmt.Printf("  Fabricante:   %s\n", orNA(inv.Manufacturer))
	fmt.Printf("  Modelo:       %s\n", orNA(inv.Model))
	fmt.Printf("  Chassi:       %s\n", orNA(inv.ChassisType))
	fmt.Printf("  CPU:          %s\n", truncate(inv.CPU, 50))
	fmt.Printf("  RAM:          %s\n", formatBytes(inv.RAMBytes))

	if len(inv.Disks) > 0 {
		fmt.Printf("  Discos:       %d\n", len(inv.Disks))
		for _, d := range inv.Disks {
			fmt.Printf("    - %s: %s %s (%s)\n",
				d.Name, d.Type, formatBytes(d.SizeBytes), orNA(d.Model))
		}
	} else {
		fmt.Printf("  Discos:       nenhum detectado\n")
	}

	if inv.IDMSCM != "" {
		fmt.Printf("  ID MSCM:      %s\n", inv.IDMSCM)
	}
	if inv.IDFornecedor != "" {
		fmt.Printf("  ID Fornecedor: %s\n", inv.IDFornecedor)
	}

	fmt.Println()

	if sent {
		fmt.Println("  [OK] ENVIADO COM SUCESSO")
	} else {
		fmt.Println("  [!] SEM REDE — USE O QR CODE OU O JSON ACIMA")
	}

	if len(inv.Errors) > 0 {
		fmt.Println("\n  Avisos:")
		for _, e := range inv.Errors {
			fmt.Printf("    - %s\n", e)
		}
	}

	fmt.Println()
}

func orNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func formatBytes(b uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.0f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.0f KB", float64(b)/float64(KB))
	}
	return fmt.Sprintf("%d B", b)
}