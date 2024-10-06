package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Defina suas variáveis
const (
	apiToken      = "x-6IBY2gHGb11vr60F13YznNFXmEgATFJpWwMu3g"
	domain        = "sshtproject.com"
	subdomain     = "ssht.telks.sshtproject.com"
	interfaceName = "eth0" // Nome da interface de rede que possui o endereço IPv6
	logFile       = "ip_update.log"
)

// Struct para resposta JSON da API
type ApiResponse struct {
	Success bool `json:"success"`
	Result  []struct {
		ID string `json:"id"`
	} `json:"result"`
}

func main() {
	// Verifique o token
	if !verifyToken(apiToken) {
		fmt.Println("Token inválido ou sem permissões adequadas.")
		return
	}
	fmt.Println("Token verificado com sucesso.")

	// Obter o ZONE_ID
	zoneID, err := getZoneID(apiToken, domain)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("ZONE_ID: %s\n", zoneID)

	// Início do loop infinito
	for {
		// Obter o RECORD_ID do subdomínio
		recordID, err := getRecordID(apiToken, zoneID, subdomain)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Obter o endereço IPv6 atual da interface de rede
		ipv6, err := getCurrentIPv6(interfaceName)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Endereço IPv6 atual: %s\n", ipv6)

		var oldIP string
		if recordID != "" {
			// Obter o IP atual do registro existente
			oldIP, err = getCurrentDNSRecord(apiToken, zoneID, recordID)
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		if oldIP != ipv6 {
			fmt.Printf("Atualizando o IP: antigo = %s, novo = %s\n", oldIP, ipv6)
			data := fmt.Sprintf(`{
				"type": "AAAA",
				"name": "%s",
				"content": "%s",
				"ttl": 120,
				"proxied": false
			}`, subdomain, ipv6)

			if recordID == "" {
				fmt.Printf("Registro AAAA para %s não encontrado. Criando um novo registro.\n", subdomain)
				err = createDNSRecord(apiToken, zoneID, data)
			} else {
				fmt.Printf("Registro AAAA para %s encontrado. Atualizando o registro existente.\n", subdomain)
				err = updateDNSRecord(apiToken, zoneID, recordID, data)
			}

			if err != nil {
				fmt.Println(err)
				return
			}

			// Log da atualização
			logIPUpdate(oldIP, ipv6)
		} else {
			fmt.Println("Nenhuma atualização necessária. O IP está o mesmo.")
		}

		// Aguardar 5 minutos antes de verificar novamente
		time.Sleep(5 * time.Minute)
	}
}

func verifyToken(token string) bool {
	url := "https://api.cloudflare.com/client/v4/user/tokens/verify"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var result ApiResponse
	json.Unmarshal(body, &result)

	return result.Success
}

func getZoneID(token, domain string) (string, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", domain)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var result ApiResponse
	json.Unmarshal(body, &result)

	if len(result.Result) == 0 {
		return "", fmt.Errorf("não foi possível obter o ZONE_ID.")
	}

	return result.Result[0].ID, nil
}

func getRecordID(token, zoneID, subdomain string) (string, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=AAAA&name=%s", zoneID, subdomain)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var result ApiResponse
	json.Unmarshal(body, &result)

	if len(result.Result) == 0 {
		return "", nil // Registro não encontrado
	}

	return result.Result[0].ID, nil
}

func getCurrentIPv6(interfaceName string) (string, error) {
	cmd := exec.Command("ip", "-6", "addr", "show", "dev", interfaceName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("não foi possível obter o endereço IPv6 da interface %s: %v", interfaceName, err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "inet6") && strings.Contains(line, "global") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				return strings.Split(parts[1], "/")[0], nil
			}
		}
	}
	return "", fmt.Errorf("endereço IPv6 não encontrado na interface %s", interfaceName)
}

func getCurrentDNSRecord(token, zoneID, recordID string) (string, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var result struct {
		Success bool `json:"success"`
		Result  struct {
			Content string `json:"content"`
		} `json:"result"`
	}
	json.Unmarshal(body, &result)

	if !result.Success {
		return "", fmt.Errorf("falha ao obter o registro DNS: %s", string(body))
	}

	return result.Result.Content, nil
}

func createDNSRecord(token, zoneID, data string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(data)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var result ApiResponse
	json.Unmarshal(body, &result)

	if !result.Success {
		return fmt.Errorf("falha ao criar o registro DNS: %s", string(body))
	}

	return nil
}

func updateDNSRecord(token, zoneID, recordID, data string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordID)
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer([]byte(data)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var result ApiResponse
	json.Unmarshal(body, &result)

	if !result.Success {
		return fmt.Errorf("falha ao atualizar o registro DNS: %s", string(body))
	}

	return nil
}

func logIPUpdate(oldIP, newIP string) {
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Erro ao abrir o arquivo de log:", err)
		return
	}
	defer file.Close()

	logEntry := fmt.Sprintf("Atualização de IP: antigo = %s, novo = %s, data = %s\n", oldIP, newIP, time.Now().Format(time.RFC3339))
	if _, err := file.WriteString(logEntry); err != nil {
		fmt.Println("Erro ao escrever no arquivo de log:", err)
	}
}
