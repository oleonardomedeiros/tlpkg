package lockfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LockedDep struct {
	Name    string
	Version string
}

const header = "# Gerado pelo tpm. Não editar manualmente.\n"

func Read(dir string) (map[string]string, error) {
	path := filepath.Join(dir, "libs.lock")
	locked := make(map[string]string)

	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return locked, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao ler libs.lock: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 2 {
			locked[parts[0]] = parts[1]
		}
	}

	return locked, nil
}

func Write(dir string, deps []LockedDep) error {
	path := filepath.Join(dir, "libs.lock")

	var sb strings.Builder
	sb.WriteString(header)
	for _, d := range deps {
		sb.WriteString(fmt.Sprintf("%s %s\n", d.Name, d.Version))
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("erro ao escrever libs.lock: %w", err)
	}

	return nil
}
