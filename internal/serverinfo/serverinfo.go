package serverinfo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Info struct {
	PublicKey string
	Port      int
	SNI       string
	Address   string
	Created   string
}

func Load(path string) (Info, error) {
	file, err := os.Open(path)
	if err != nil {
		return Info{}, err
	}
	defer file.Close()

	var info Info
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if !ok {
			continue
		}
		value = strings.Trim(value, `"'`)
		switch key {
		case "PUBLIC_KEY":
			info.PublicKey = value
		case "PORT":
			port, _ := strconv.Atoi(value)
			info.Port = port
		case "SNI":
			info.SNI = value
		case "SERVER_IP", "SERVER_ADDRESS":
			info.Address = value
		case "CREATED":
			info.Created = value
		}
	}
	return info, scanner.Err()
}

func Save(path string, info Info) error {
	if info.Created == "" {
		info.Created = time.Now().Format("2006-01-02 15:04:05")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data := fmt.Sprintf(
		"PUBLIC_KEY=%q\nPORT=%q\nSNI=%q\nSERVER_IP=%q\nCREATED=%q\n",
		info.PublicKey,
		strconv.Itoa(info.Port),
		info.SNI,
		info.Address,
		info.Created,
	)
	return os.WriteFile(path, []byte(data), 0o600)
}

func ResolveAddress(info Info, detect func() (string, error)) string {
	if info.Address != "" {
		return info.Address
	}
	address, err := detect()
	if err != nil || address == "" {
		return "UNKNOWN"
	}
	return address
}
