package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Dependency struct {
	Name    string
	Version string
}

type PackagesFile struct {
	Source       string
	Dependencies []Dependency
}

var (
	sourcePattern  = regexp.MustCompile(`^source\s+'([^']+)'`)
	packagePattern = regexp.MustCompile(`^package\s+'([^']+)',\s+'([^']+)'`)
)

func ParsePackagesFile(path string) (*PackagesFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("arquivo 'packages' não encontrado: %w", err)
	}
	defer file.Close()

	result := &PackagesFile{}
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if matches := sourcePattern.FindStringSubmatch(line); matches != nil {
			result.Source = matches[1]
			continue
		}

		if matches := packagePattern.FindStringSubmatch(line); matches != nil {
			result.Dependencies = append(result.Dependencies, Dependency{
				Name:    matches[1],
				Version: matches[2],
			})
			continue
		}

		return nil, fmt.Errorf("linha %d inválida: %q\n  esperado: source 'url' ou package 'nome', 'versão'", lineNum, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo packages: %w", err)
	}

	if result.Source == "" {
		return nil, fmt.Errorf("diretiva 'source' não encontrada no arquivo packages\n  adicione: source 'https://cdn.jsdelivr.net/gh/sua-org/tpm-registry@main'")
	}

	return result, nil
}
