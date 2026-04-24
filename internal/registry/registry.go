package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type PackageInfo struct {
	Description string   `json:"description"`
	Versions    []string `json:"versions"`
	Latest      string   `json:"latest"`
}

type Index struct {
	Packages map[string]PackageInfo `json:"packages"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(sourceURL string) *Client {
	return &Client{
		baseURL:    strings.TrimSuffix(sourceURL, "/"),
		httpClient: &http.Client{},
	}
}

func (c *Client) FetchIndex() (*Index, error) {
	url := c.baseURL + "/index.json"
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro ao acessar registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry retornou status %d\nURL: %s", resp.StatusCode, url)
	}

	var index Index
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("erro ao interpretar index.json: %w", err)
	}

	return &index, nil
}

func (c *Client) ResolveVersion(index *Index, name, constraint string) (string, error) {
	pkg, ok := index.Packages[name]
	if !ok {
		return "", fmt.Errorf("pacote '%s' não encontrado no registry", name)
	}

	if constraint == "latest" {
		return pkg.Latest, nil
	}

	// Versão exata
	if !strings.HasPrefix(constraint, "^") && !strings.HasPrefix(constraint, ">=") {
		for _, v := range pkg.Versions {
			if v == constraint {
				return v, nil
			}
		}
		return "", fmt.Errorf("versão '%s' do pacote '%s' não encontrada\n  versões disponíveis: %s",
			constraint, name, strings.Join(pkg.Versions, ", "))
	}

	// Compatível com major (^1.2.0 → >= 1.2.0 e < 2.0.0)
	if strings.HasPrefix(constraint, "^") {
		base := strings.TrimPrefix(constraint, "^")
		resolved, err := resolveCompatible(pkg.Versions, base)
		if err != nil {
			return "", fmt.Errorf("pacote '%s': %w", name, err)
		}
		return resolved, nil
	}

	// Mínimo (>=1.0.0)
	if strings.HasPrefix(constraint, ">=") {
		base := strings.TrimPrefix(constraint, ">=")
		resolved, err := resolveMinimum(pkg.Versions, base)
		if err != nil {
			return "", fmt.Errorf("pacote '%s': %w", name, err)
		}
		return resolved, nil
	}

	return "", fmt.Errorf("constraint '%s' não suportado", constraint)
}

func (c *Client) Download(name, version, destDir string) (string, error) {
	url := fmt.Sprintf("%s/packages/%s/%s/%s.tlpp", c.baseURL, name, version, name)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("erro ao baixar '%s@%s': %w", name, version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("erro ao baixar '%s@%s': status %d", name, version, resp.StatusCode)
	}

	destPath := filepath.Join(destDir, name+".tlpp")
	out, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("erro ao criar arquivo '%s': %w", destPath, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("erro ao salvar '%s': %w", destPath, err)
	}

	return destPath, nil
}

func resolveCompatible(versions []string, base string) (string, error) {
	baseParts := splitVersion(base)
	if len(baseParts) < 1 {
		return "", fmt.Errorf("versão base inválida: %s", base)
	}
	major := baseParts[0]

	var best []int
	var bestStr string

	for _, v := range versions {
		parts := splitVersion(v)
		if len(parts) < 3 {
			continue
		}
		if parts[0] != major {
			continue
		}
		if compareVersions(parts, splitVersion(base)) >= 0 {
			if best == nil || compareVersions(parts, best) > 0 {
				best = parts
				bestStr = v
			}
		}
	}

	if bestStr == "" {
		return "", fmt.Errorf("nenhuma versão compatível com ^%s encontrada", base)
	}
	return bestStr, nil
}

func resolveMinimum(versions []string, base string) (string, error) {
	baseParts := splitVersion(base)
	var best []int
	var bestStr string

	for _, v := range versions {
		parts := splitVersion(v)
		if compareVersions(parts, baseParts) >= 0 {
			if best == nil || compareVersions(parts, best) > 0 {
				best = parts
				bestStr = v
			}
		}
	}

	if bestStr == "" {
		return "", fmt.Errorf("nenhuma versão >= %s encontrada", base)
	}
	return bestStr, nil
}

func splitVersion(v string) []int {
	parts := strings.Split(v, ".")
	nums := make([]int, len(parts))
	for i, p := range parts {
		fmt.Sscanf(p, "%d", &nums[i])
	}
	return nums
}

func compareVersions(a, b []int) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] - b[i]
		}
	}
	return len(a) - len(b)
}
