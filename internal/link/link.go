package link

import (
	"fmt"
	"net/url"
)

type Link struct {
	UUID      string
	Address   string
	Port      int
	PublicKey string
	SNI       string
	Name      string
	ShortID   string
}

func GenerateVLESS(link Link) string {
	return fmt.Sprintf(
		"vless://%s@%s:%d?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=chrome&pbk=%s&sid=%s&type=tcp&headerType=none#%s",
		link.UUID,
		link.Address,
		link.Port,
		url.QueryEscape(link.SNI),
		url.QueryEscape(link.PublicKey),
		url.QueryEscape(link.ShortID),
		url.PathEscape(link.Name),
	)
}
