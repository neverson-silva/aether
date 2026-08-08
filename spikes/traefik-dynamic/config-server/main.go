// SPIKE-TRAEFIK: servidor de dynamic config EM MEMÓRIA.
// Representa o "Networking Engine" servindo o ProxyConfig ao provider HTTP do
// Traefik. O spike prova que alterações de rota entram em vigor sem:
//   - escrita de arquivo em disco (fora do traefik.yml estático)
//   - restart do Traefik
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

var (
	mu     sync.RWMutex
	config = initialConfig()
)

func initialConfig() map[string]interface{} {
	return map[string]interface{}{
		"http": map[string]interface{}{
			"routers": map[string]interface{}{
				"router-a": map[string]interface{}{
					"rule":        "Host(`spike.local`)",
					"service":     "svc-a",
					"entryPoints": []string{"web"},
				},
			},
			"services": map[string]interface{}{
				"svc-a": map[string]interface{}{
					"loadBalancer": map[string]interface{}{
						"servers": []map[string]string{{"url": "http://127.0.0.1:18099"}},
					},
				},
			},
		},
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		log.Printf("encode: %v", err)
	}
}

func main() {
	// rota extra adicionada "ao vivo" (simula deploy de novo domínio sem file write)
	mu.Lock()
	routers := config["http"].(map[string]interface{})["routers"].(map[string]interface{})
	routers["router-b"] = map[string]interface{}{
		"rule":        "Host(`spike2.local`)",
		"service":     "svc-a",
		"entryPoints": []string{"web"},
	}
	mu.Unlock()

	http.HandleFunc("/traefik", handler)
	port := "18090"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	fmt.Println("config-server listen :" + port)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+port, nil))
}
