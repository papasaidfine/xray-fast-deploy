package link

import "testing"

func TestGenerateVLESSLink(t *testing.T) {
	got := GenerateVLESS(Link{
		UUID:      "11111111-1111-4111-8111-111111111111",
		Address:   "vpn.example.com",
		Port:      443,
		PublicKey: "public-key",
		SNI:       "www.apple.com",
		Name:      `phone "alpha"`,
		ShortID:   "short-id",
	})

	want := `vless://11111111-1111-4111-8111-111111111111@vpn.example.com:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=www.apple.com&fp=chrome&pbk=public-key&sid=short-id&type=tcp&headerType=none#phone%20%22alpha%22`
	if got != want {
		t.Fatalf("link = %q, want %q", got, want)
	}
}
