package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
	mu     sync.Mutex
	seq    int
}

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewClient(advplsPath string) (*Client, error) {
	cmd := exec.Command(advplsPath, "language-server")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("falha ao iniciar advpls.exe: %w", err)
	}

	c := &Client{
		cmd:    cmd,
		stdin:  stdin,
		reader: bufio.NewReader(stdout),
		seq:    1,
	}

	if err := c.initialize(); err != nil {
		c.Close()
		return nil, err
	}

	return c, nil
}

func (c *Client) initialize() error {
	_, err := c.sendRequest("initialize", map[string]interface{}{
		"processId":    os.Getpid(),
		"capabilities": map[string]interface{}{},
	})
	if err != nil {
		return fmt.Errorf("LSP initialize falhou: %w", err)
	}

	return c.sendNotification("initialized", map[string]interface{}{})
}

func (c *Client) Connect(serverName, address string, port int, build, environment string) (token string, needAuth bool, err error) {
	result, err := c.sendRequest("$totvsserver/connect", map[string]interface{}{
		"connectionInfo": map[string]interface{}{
			"connType":      1,
			"serverName":    serverName,
			"identification": "",
			"serverType":    1,
			"server":        address,
			"port":          port,
			"build":         build,
			"bSecure":       0,
			"environment":   environment,
			"autoReconnect": true,
		},
	})
	if err != nil {
		return "", false, err
	}

	var resp struct {
		ConnectionToken    string `json:"connectionToken"`
		NeedAuthentication bool   `json:"needAuthentication"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", false, err
	}

	return resp.ConnectionToken, resp.NeedAuthentication, nil
}

func (c *Client) Authenticate(token, environment, user, password string) (string, error) {
	result, err := c.sendRequest("$totvsserver/authentication", map[string]interface{}{
		"authenticationInfo": map[string]interface{}{
			"connectionToken": token,
			"environment":     environment,
			"user":            user,
			"password":        password,
			"encoding":        "CP1252",
		},
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		ConnectionToken string `json:"connectionToken"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", err
	}

	return resp.ConnectionToken, nil
}

func (c *Client) Compile(token, environment string, filePaths, includes []string, recompile bool) error {
	fileUris := make([]string, len(filePaths))
	for i, p := range filePaths {
		fileUris[i] = toFileURI(p)
	}

	includeUris := make([]string, len(includes))
	for i, p := range includes {
		includeUris[i] = toFileURI(p)
	}

	result, err := c.sendRequest("$totvsserver/compilation", map[string]interface{}{
		"compilationInfo": map[string]interface{}{
			"connectionToken":     token,
			"authorizationToken":  "",
			"environment":         environment,
			"includeUris":         includeUris,
			"fileUris":            fileUris,
			"compileOptions": map[string]interface{}{
				"recompile":       recompile,
				"generatePpoFile": false,
				"showPreCompiler": false,
			},
			"extensionsAllowed":   []string{".tlpp", ".prw", ".prx", ".prg", ".apw", ".aph", ".ahu", ".apo", ".apt"},
			"includeUrisRequired": len(includes) > 0,
		},
	})
	if err != nil {
		return err
	}

	var resp struct {
		CompileInfos []struct {
			Status string `json:"status"`
			Detail string `json:"detail"`
		} `json:"compileInfos"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return err
	}

	for _, info := range resp.CompileInfos {
		switch info.Status {
		case "FATAL", "ERROR":
			return fmt.Errorf("%s", info.Detail)
		case "WARN":
			fmt.Printf("  aviso: %s\n", info.Detail)
		}
	}

	return nil
}

func (c *Client) DeletePrograms(token, environment string, filePaths []string) error {
	_, err := c.sendRequest("$totvsserver/deletePrograms", map[string]interface{}{
		"deleteProgramsInfo": map[string]interface{}{
			"connectionToken":    token,
			"authorizationToken": "",
			"environment":        environment,
			"programs":           filePaths,
		},
	})
	return err
}

func (c *Client) Disconnect(token, serverName string) {
	_ = c.sendNotification("$totvsserver/disconnect", map[string]interface{}{
		"disconnectInfo": map[string]interface{}{
			"connectionToken": token,
			"serverName":      serverName,
		},
	})
}

func (c *Client) Close() {
	c.stdin.Close()
	_ = c.cmd.Wait()
}

func (c *Client) sendRequest(method string, params interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	id := c.seq
	c.seq++
	c.mu.Unlock()

	if err := c.write(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}); err != nil {
		return nil, err
	}

	for {
		msg, err := c.readMessage()
		if err != nil {
			return nil, err
		}
		if msg.ID != nil && *msg.ID == id {
			if msg.Error != nil {
				return nil, fmt.Errorf("LSP %s: %s (code %d)", method, msg.Error.Message, msg.Error.Code)
			}
			return msg.Result, nil
		}
		// Ignora notificações e respostas de outros ids
	}
}

func (c *Client) sendNotification(method string, params interface{}) error {
	return c.write(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	})
}

func (c *Client) write(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := io.WriteString(c.stdin, header); err != nil {
		return err
	}
	_, err = c.stdin.Write(data)
	return err
}

func (c *Client) readMessage() (*rpcMessage, error) {
	var contentLength int

	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("erro ao ler header LSP: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			parts := strings.SplitN(line, ":", 2)
			n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("Content-Length inválido: %w", err)
			}
			contentLength = n
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("Content-Length ausente na resposta LSP")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(c.reader, body); err != nil {
		return nil, fmt.Errorf("erro ao ler corpo LSP: %w", err)
	}

	var msg rpcMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("JSON inválido na resposta LSP: %w", err)
	}

	return &msg, nil
}

func toFileURI(p string) string {
	p = filepath.ToSlash(filepath.Clean(p))
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return "file://" + p
}
