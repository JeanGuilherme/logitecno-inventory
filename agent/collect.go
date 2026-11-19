package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Disk struct {
	Name       string `json:"name"`
	Model      string `json:"model"`
	Serial     string `json:"serial"`
	SizeBytes  uint64 `json:"size_bytes"`
	Type       string `json:"type"`        // "SSD", "HDD", "NVMe"
	Rotational bool   `json:"rotational"`
	Transport  string `json:"transport"`   // "sata", "nvme", "usb", ""
}

type MemoryModule struct {
	Slot       string `json:"slot"`
	SizeBytes  uint64 `json:"size_bytes"`
	Type       string `json:"type"`      // DDR3, DDR4, DDR5
	SpeedMHz   int    `json:"speed_mhz"`
	Manufacturer string `json:"manufacturer,omitempty"`
	Serial     string `json:"serial,omitempty"`
}

type Inventory struct {
	SchemaVersion string    `json:"schema_version"`
	CollectedAt   time.Time `json:"collected_at"`

	// Identificação
	Hostname     string `json:"hostname"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Serial       string `json:"serial"`
	BoardSerial  string `json:"board_serial"`
	BoardName    string `json:"board_name"`

	// Chassi
	ChassisType  string `json:"chassis_type"`
	ChassisCode  int    `json:"chassis_code"`

	// BIOS
	BIOSVendor   string `json:"bios_vendor"`
	BIOSVersion  string `json:"bios_version"`
	BIOSDate     string `json:"bios_date"`
	FirmwareType string `json:"firmware_type"` // "UEFI" ou "Legacy"

	// CPU
	CPU string `json:"cpu"`

	// RAM
	RAMBytes      uint64         `json:"ram_bytes"`
	RAMModules    []MemoryModule `json:"ram_modules"`
	RAMSlotsTotal int            `json:"ram_slots_total"`
	RAMSlotsFree  int            `json:"ram_slots_free"`

	// Discos
	Disks []Disk `json:"disks"`

	// Rede
	MACs []string `json:"macs"`

	// Input manual
	IDMSCM       string `json:"id_mscm,omitempty"`
	IDFornecedor string `json:"id_fornecedor,omitempty"`

	// Estado
	Network NetStatus `json:"network"`
	Errors  []string  `json:"errors,omitempty"`
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
		Disks:         []Disk{},
		RAMModules:    []MemoryModule{},
	}

	// --- Identificação ---
	inv.Hostname, _ = os.Hostname()
	inv.Manufacturer = readFile("/sys/class/dmi/id/sys_vendor")
	inv.Model = readFile("/sys/class/dmi/id/product_name")
	inv.Serial = readFile("/sys/class/dmi/id/product_serial")
	inv.BoardSerial = readFile("/sys/class/dmi/id/board_serial")
	inv.BoardName = readFile("/sys/class/dmi/id/board_name")

	if inv.Serial == "" || strings.Contains(strings.ToLower(inv.Serial), "to be filled") {
		inv.Errors = append(inv.Errors, "serial ausente ou invalido")
	}

	// --- Chassi ---
	chassisRaw := readFile("/sys/class/dmi/id/chassis_type")
	if chassisRaw != "" {
		if code, err := strconv.Atoi(chassisRaw); err == nil {
			inv.ChassisCode = code
			inv.ChassisType = chassisTypeName(code)
		}
	}

	// --- BIOS ---
	inv.BIOSVendor = readFile("/sys/class/dmi/id/bios_vendor")
	inv.BIOSVersion = readFile("/sys/class/dmi/id/bios_version")
	inv.BIOSDate = readFile("/sys/class/dmi/id/bios_date")

	if _, err := os.Stat("/sys/firmware/efi"); err == nil {
		inv.FirmwareType = "UEFI"
	} else {
		inv.FirmwareType = "Legacy"
	}

	// --- CPU ---
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

	// --- RAM total ---
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if kb, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
						inv.RAMBytes = kb * 1024
					}
				}
				break
			}
		}
	}

	// --- RAM detalhada (dmidecode) ---
	collectMemoryModules(&inv)

	// --- Discos (lsblk) ---
	collectDisks(&inv)

	// --- MACs ---
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

func chassisTypeName(code int) string {
	switch code {
	case 3:
		return "Desktop"
	case 4:
		return "Low Profile Desktop"
	case 5:
		return "Pizza Box"
	case 6:
		return "Mini Tower"
	case 7:
		return "Tower"
	case 8:
		return "Portable"
	case 9:
		return "Laptop"
	case 10:
		return "Notebook"
	case 11:
		return "Hand Held"
	case 13:
		return "All in One"
	case 14:
		return "Sub Notebook"
	case 15:
		return "Space-saving"
	case 16:
		return "Lunch Box"
	case 17:
		return "Main Server Chassis"
	case 23:
		return "Rack Mount Chassis"
	case 30:
		return "Tablet"
	case 31:
		return "Convertible"
	case 32:
		return "Detachable"
	default:
		return fmt.Sprintf("Desconhecido (%d)", code)
	}
}

// collectDisks usa lsblk -J (JSON nativo, sem parsing manual).
// lsblk já resolve SATA, NVMe e USB sem dependencia extra.
func collectDisks(inv *Inventory) {
	out, err := exec.Command(
		"lsblk", "-J", "-b", "-d",
		"-o", "NAME,MODEL,SERIAL,SIZE,ROTA,TYPE,TRAN",
	).Output()
	if err != nil {
		inv.Errors = append(inv.Errors, "lsblk indisponivel: "+err.Error())
		return
	}

	var parsed struct {
		Blockdevices []struct {
			Name   string      `json:"name"`
			Model  string      `json:"model"`
			Serial string      `json:"serial"`
			Size   json.Number `json:"size"`
			Rota   json.Number `json:"rota"`
			Type   string      `json:"type"`
			Tran   string      `json:"tran"`
		} `json:"blockdevices"`
	}

	if err := json.Unmarshal(out, &parsed); err != nil {
		inv.Errors = append(inv.Errors, "lsblk parse: "+err.Error())
		return
	}

	for _, d := range parsed.Blockdevices {
		if d.Type != "disk" {
			continue
		}

		size, _ := d.Size.Int64()
		rota, _ := d.Rota.Int64()

		diskType := "HDD"
		if rota == 0 {
			diskType = "SSD"
		}
		if strings.HasPrefix(d.Name, "nvme") {
			diskType = "NVMe"
		}

		inv.Disks = append(inv.Disks, Disk{
			Name:       d.Name,
			Model:      strings.TrimSpace(d.Model),
			Serial:     strings.TrimSpace(d.Serial),
			SizeBytes:  uint64(size),
			Type:       diskType,
			Rotational: rota == 1,
			Transport:  strings.TrimSpace(d.Tran),
		})
	}
}

// collectMemoryModules usa dmidecode -t memory.
// dmidecode precisa de root. Se falhar, o JSON sai sem os modulos.
func collectMemoryModules(inv *Inventory) {
	out, err := exec.Command("dmidecode", "-t", "memory").Output()
	if err != nil {
		inv.Errors = append(inv.Errors, "dmidecode indisponivel (precisa root)")
		return
	}

	var current *MemoryModule
	slotsTotal := 0

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Memory Device") {
			if current != nil {
				inv.RAMModules = append(inv.RAMModules, *current)
			}
			current = &MemoryModule{}
			slotsTotal++
			continue
		}

		if current == nil {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Size":
			if val != "No Module Installed" && val != "Unknown" {
				current.SizeBytes = parseSizeToBytes(val)
			}
		case "Type":
			if val != "Unknown" && val != "" {
				current.Type = val
			}
		case "Speed":
			fields := strings.Fields(val)
			if len(fields) > 0 {
				if mhz, err := strconv.Atoi(fields[0]); err == nil {
					current.SpeedMHz = mhz
				}
			}
		case "Manufacturer":
			if val != "Unknown" && val != "" && val != "Not Specified" {
				current.Manufacturer = val
			}
		case "Serial Number":
			if val != "Unknown" && val != "" && val != "Not Specified" {
				current.Serial = val
			}
		case "Locator":
			if val != "" {
				current.Slot = val
			}
		}
	}

	if current != nil {
		inv.RAMModules = append(inv.RAMModules, *current)
	}

	inv.RAMSlotsTotal = slotsTotal
	occupied := 0
	for _, m := range inv.RAMModules {
		if m.SizeBytes > 0 {
			occupied++
		}
	}
	inv.RAMSlotsFree = slotsTotal - occupied
}

func parseSizeToBytes(s string) uint64 {
	// Formatos: "8192 MB", "16 GB", "4096 kB"
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return 0
	}
	val, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return 0
	}
	switch strings.ToUpper(fields[1]) {
	case "KB":
		return val * 1024
	case "MB":
		return val * 1024 * 1024
	case "GB":
		return val * 1024 * 1024 * 1024
	case "TB":
		return val * 1024 * 1024 * 1024 * 1024
	}
	return 0
}