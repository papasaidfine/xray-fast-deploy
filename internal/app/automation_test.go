package app

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseX25519Output(t *testing.T) {
	cases := []struct {
		name string
		in   string
		priv string
		pub  string
	}{
		{
			name: "legacy format",
			in:   "Private key: PRIV_LEGACY\nPublic key: PUB_LEGACY\n",
			priv: "PRIV_LEGACY",
			pub:  "PUB_LEGACY",
		},
		{
			name: "password format",
			in:   "PrivateKey: PRIV_MID\nPassword: PUB_MID\n",
			priv: "PRIV_MID",
			pub:  "PUB_MID",
		},
		{
			name: "26.x format with hash line",
			in: "PrivateKey: 0J0WzS6tWfMjs6uVRqfztxyNpsKM-d9ru00FEp_I5Eg\n" +
				"Password (PublicKey): 0lLaHIp__WkwJtzLrv-mCv_8-WG-mmtRtMDLic9uQA0\n" +
				"Hash32: 6Md3xh2myFA7K4E7t6rmbyeTq4TRJsNVn4F45oNxqb4\n",
			priv: "0J0WzS6tWfMjs6uVRqfztxyNpsKM-d9ru00FEp_I5Eg",
			pub:  "0lLaHIp__WkwJtzLrv-mCv_8-WG-mmtRtMDLic9uQA0",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pub, priv, err := parseX25519Output(tc.in)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if priv != tc.priv {
				t.Errorf("priv = %q, want %q", priv, tc.priv)
			}
			if pub != tc.pub {
				t.Errorf("pub = %q, want %q", pub, tc.pub)
			}
		})
	}
}

func TestParseX25519OutputRejectsUnknownFormat(t *testing.T) {
	const raw = "some future format nobody expected"
	_, _, err := parseX25519Output(raw)
	if err == nil {
		t.Fatal("parse of unknown format succeeded, want error")
	}
	if !strings.Contains(err.Error(), raw) {
		t.Fatalf("error %q does not include the raw output", err)
	}
}

func TestInstallerCurlArgs(t *testing.T) {
	t.Run("no proxy", func(t *testing.T) {
		got := installerCurlArgs("")
		want := []string{"-L", xrayInstallURL}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("installerCurlArgs(\"\") = %q, want %q", got, want)
		}
	})
	t.Run("with proxy", func(t *testing.T) {
		got := installerCurlArgs("socks5://127.0.0.1:1080")
		want := []string{"-L", "-x", "socks5://127.0.0.1:1080", xrayInstallURL}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("installerCurlArgs(proxy) = %q, want %q", got, want)
		}
	})
}

func TestInstallerBashArgs(t *testing.T) {
	t.Run("no proxy", func(t *testing.T) {
		got := installerBashArgs("")
		want := []string{"-s", "install"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("installerBashArgs(\"\") = %q, want %q", got, want)
		}
	})
	t.Run("with proxy", func(t *testing.T) {
		got := installerBashArgs("http://127.0.0.1:8118")
		want := []string{"-s", "install", "--proxy", "http://127.0.0.1:8118"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("installerBashArgs(proxy) = %q, want %q", got, want)
		}
	})
}

func TestPickFallbackUser(t *testing.T) {
	set := func(names ...string) func(string) bool {
		m := map[string]bool{}
		for _, n := range names {
			m[n] = true
		}
		return func(s string) bool { return m[s] }
	}
	cases := []struct {
		name      string
		users     []string
		groups    []string
		wantUser  string
		wantGroup string
	}{
		{"xray user and group", []string{"xray"}, []string{"xray"}, "xray", "xray"},
		{"nobody with nogroup", []string{"nobody"}, []string{"nogroup"}, "nobody", "nogroup"},
		{"nobody with nobody group", []string{"nobody"}, []string{"nobody"}, "nobody", "nobody"},
		{"nobody but no matching group", []string{"nobody"}, nil, "nobody", ""},
		{"xray preferred over nobody", []string{"xray", "nobody"}, []string{"xray", "nogroup"}, "xray", "xray"},
		{"no service user exists", nil, nil, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, g := pickFallbackUser(set(tc.users...), set(tc.groups...))
			if u != tc.wantUser || g != tc.wantGroup {
				t.Errorf("pickFallbackUser = (%q, %q), want (%q, %q)", u, g, tc.wantUser, tc.wantGroup)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v0.1.0", "v0.1.0", 0},
		{"v0.1.0", "v0.1.1", -1},
		{"v0.1.1", "v0.1.0", 1},
		{"v0.1.0-beta.1", "v0.1.0-beta.2", -1},
		{"v0.1.0-beta.2", "v0.1.0-beta.10", -1},
		{"v0.1.0-beta.1", "v0.1.0", -1},
		{"v0.1.0", "v0.1.0-beta.1", 1},
		{"dev", "v0.1.0-beta.6", -1},
		{"v0.1.0-beta.6", "dev", 1},
		{"v0.2.0-beta.1", "v0.1.0", 1},
	}
	for _, c := range cases {
		got := compareVersions(c.a, c.b)
		if got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
