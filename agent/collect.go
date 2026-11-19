package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Inventory struct {
	SchemaVersion string    `json:"schema_version"`
	CollectedAt   time.Time `json:"collected_at"`
	Hostname      string    `json:"hostname"`
	Manufacturer  string    `json:"manufacturer"`
	Model         string    `json:"model"`
	Serial        string    `json:"serial"`
	BoardSerial   string    `json:"board_serial"`
	CPU           string    `json:"cpu"`
	RAMBytes      uint64    `json:"ram_bytes"`
	GPU           string    `json:"gpu"`
	MACs          []string  `json:"macs"`
	Network       NetStatus `json:"network"`
	Errors        []string  `json:"errors,omitempty"`
}

func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func collectInventory() Inventory {
	inv := Inventory{
		SchemaVersion: schemaVersion,
		CollectedAt:   time.Now().UTC(),
		MACs:          []string{},
	}

	inv.Hostname, _ = os.Hostname()
	inv.Manufacturer = readFile("/sys/class/dmi/id/sys_vendor")
	inv.Model = readFile("/sys/class/dmi/id/product_name")
	inv.Serial = readFile("/sys/class/dmi/id/product_serial")
	inv.BoardSerial = readFile("/sys/class/dmi/id/board_serial")

	if inv.Serial == "" || strings.Contains(strings.ToLower(inv.Serial), "to be filled") {
		inv.Errors = append(inv.Errors, "serial ausente ou invalido")
	}

	// CPU
	if b, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					inv.CPU = strings.TrimSpace(parts[1])
				}
				break
			}
		}
	}

	// RAM
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				var kb uint64
				if _, err := fmt.Sscanf(line, "MemTotal: %d kB", &kb); err == nil {
					inv.RAMBytes = kb * 1024
				}
				break
			}
		}
	}

	// GPU
	if out, err := exec.Command("lspci").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "vga") || strings.Contains(lower, "3d controller") {
				parts := strings.SplitN(line, ": ", 2)
				if len(parts) == 2 {
					inv.GPU = strings.TrimSpace(parts[1])
				}
				break
			}
		}
	} else {
		inv.Errors = append(inv.Errors, "lspci indisponivel")
	}

	// MACs
	if entries, err := os.ReadDir("/sys/class/net"); err == nil {
		for _, e := range entries {
			name := e.Name()
			if name == "lo" {
				continue
			}
			mac := readFile(filepath.Join("/sys/class/net", name, "address"))
			if mac != "" && mac != "00:00:00:00:00:00" {
				inv.MACs = append(inv.MACs, mac)
			}
		}
	}

	return inv
}