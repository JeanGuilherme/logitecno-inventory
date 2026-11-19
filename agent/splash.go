package main

import (
	"fmt"
	"time"
)

const (
	cReset  = "\033[0m"
	cBold   = "\033[1m"
	cCyan   = "\033[36m"
	cGreen  = "\033[32m"
	cYellow = "\033[33m"
	cRed    = "\033[31m"
	cGray   = "\033[90m"
	cWhite  = "\033[97m"
)

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func showSplash() {
	clearScreen()

	fmt.Println()
	fmt.Println()
	fmt.Printf("%s%s", cCyan, cBold)
	fmt.Println("        ██╗      ██████╗  ██████╗ ██╗████████╗███████╗ ██████╗███╗   ██╗ ██████╗ ")
	fmt.Println("        ██║     ██╔═══██╗██╔════╝ ██║╚══██╔══╝██╔════╝██╔════╝████╗  ██║██╔═══██╗")
	fmt.Println("        ██║     ██║   ██║██║  ███╗██║   ██║   █████╗  ██║     ██╔██╗ ██║██║   ██║")
	fmt.Println("        ██║     ██║   ██║██║   ██║██║   ██║   ██╔══╝  ██║     ██║╚██╗██║██║   ██║")
	fmt.Println("        ███████╗╚██████╔╝╚██████╔╝██║   ██║   ███████╗╚██████╗██║ ╚████║╚██████╔╝")
	fmt.Println("        ╚══════╝ ╚═════╝  ╚═════╝ ╚═╝   ╚═╝   ╚══════╝ ╚═════╝╚═╝  ╚═══╝ ╚═════╝ ")
	fmt.Printf("%s", cReset)

	fmt.Println()
	fmt.Printf("%s%s              S O L U T I O N S%s\n", cGray, cBold, cReset)
	fmt.Println()
	fmt.Printf("%s          ────────────────────────────────────────%s\n", cGray, cReset)
	fmt.Println()
	fmt.Printf("%s%s              Inventario de Hardware%s\n", cWhite, cBold, cReset)
	fmt.Printf("%s                  versao %s%s\n", cGray, schemaVersion, cReset)
	fmt.Println()
	fmt.Println()

	fmt.Printf("%s          Carregando", cGray)
	for i := 0; i < 3; i++ {
		time.Sleep(400 * time.Millisecond)
		fmt.Print(".")
	}
	fmt.Printf("%s\n", cReset)

	time.Sleep(300 * time.Millisecond)
}

func printStatus(label string, ok bool, detail string) {
	icon := cGreen + "[OK]" + cReset
	if !ok {
		icon = cRed + "[!!]" + cReset
	}
	if detail != "" {
		fmt.Printf("  %s %-20s %s%s%s\n", icon, label, cGray, detail, cReset)
	} else {
		fmt.Printf("  %s %s\n", icon, label)
	}
}

func printWarn(msg string) {
	fmt.Printf("  %s[AVISO]%s %s\n", cYellow, cReset, msg)
}

func printInfo(msg string) {
	fmt.Printf("  %s%s%s\n", cGray, msg, cReset)
}