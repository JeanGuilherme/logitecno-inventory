package main

import (
	"fmt"

	qrcode "github.com/skip2/go-qrcode"
)

// showQRCode renderiza o payload como QR Code no terminal.
// Usa ToSmallString para caber em terminal de 80 colunas.
func showQRCode(data string) error {
	qr, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		return err
	}

	qr.DisableBorder = false
	fmt.Print(qr.ToSmallString(false))
	return nil
}

// buildQRPayload monta um payload compacto com os campos de identificacao.
// O JSON completo nao cabe em QR legivel de terminal — o QR serve para
// identificar a maquina rapidamente, e o JSON completo vai impresso abaixo.
func buildQRPayload(inv Inventory) string {
	return fmt.Sprintf(
		"LOGITECNO|v=%s|serial=%s|mscm=%s|forn=%s|ts=%s",
		inv.SchemaVersion,
		inv.Serial,
		inv.IDMSCM,
		inv.IDFornecedor,
		inv.CollectedAt.Format("20060102T150405"),
	)
}