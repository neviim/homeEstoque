package version

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Running é a versão embutida no binário via ldflags (-X version.Running=X.Y.Z).
var Running = "dev"

const cacheTTL = 5 * time.Second

type entry struct {
	value  string
	readAt time.Time
}

var (
	mu    sync.Mutex
	cache = map[string]entry{}
)

// Available lê a versão do arquivo VERSION em repoRoot, com cache de 5s por diretório.
// Retorna Running se o arquivo não existir.
func Available(repoRoot string) string {
	mu.Lock()
	defer mu.Unlock()
	if e, ok := cache[repoRoot]; ok && time.Since(e.readAt) < cacheTTL {
		return e.value
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, "VERSION"))
	v := Running
	if err == nil {
		v = strings.TrimSpace(string(data))
	}
	cache[repoRoot] = entry{value: v, readAt: time.Now()}
	return v
}

// IsUpdateAvailable retorna true quando a versão no arquivo difere da embutida.
func IsUpdateAvailable(repoRoot string) bool {
	return Available(repoRoot) != Running
}

// ResetCache limpa o cache de versão (usado em testes).
func ResetCache() {
	mu.Lock()
	cache = map[string]entry{}
	mu.Unlock()
}
