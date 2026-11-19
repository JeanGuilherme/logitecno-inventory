package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Endpoint configurável por variável de ambiente.
// Defina LOGITECNO_WEBHOOK_URL no ambiente ou no /etc/environment da ISO.
const defaultWebhook = "https://n8n.vps1092.panel.speedfy.host/webhook/logitecno-inventory-mscm"

func webhookURL() string {
	if v := os.Getenv("LOGITECNO_WEBHOOK_URL"); v != "" {
		return v
	}
	return defaultWebhook
}

// sendInventory envia o JSON com retry exponencial.
// 3 tentativas: 2s, 4s, 8s.
func sendInventory(inv Inventory) error {
	url := webhookURL()
	if url == "" {
		return fmt.Errorf("LOGITECNO_WEBHOOK_URL nao definida")
	}

	body, err := json.Marshal(inv)
	if err != nil {
		return fmt.Errorf("serializar: %w", err)
	}

	client := http.Client{Timeout: 10 * time.Second}
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			wait := time.Duration(1<<attempt) * time.Second
			time.Sleep(wait)
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("criar request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Schema-Version", schemaVersion)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("status %d", resp.StatusCode)
	}

	return fmt.Errorf("3 tentativas falharam: %w", lastErr)
}