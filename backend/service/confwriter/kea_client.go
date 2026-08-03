package confwriter

import (
	"backend/model"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	baseURL = "http://dhcp4-server"
	port    = "8000"
)

func (jw JsonWriter) reloadConfig() error {
	if jw.skipDhcpNotification {
		log.Println("[DEBUG] Skipping notification to dhcp server")
		return nil
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Post(
		baseURL+":"+port,
		"application/json",
		bytes.NewBufferString(`{"command":"config-reload", "service": ["dhcp4"]}`),
	)

	if err != nil {
		return model.Error(http.StatusInternalServerError, err.Error(), "sending update signal to dhcp-server failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return model.Error(
			http.StatusInternalServerError,
			fmt.Sprintf("received unexpected status %d", resp.StatusCode),
			"sending update signal to dhcp-server failed",
		)
	}

	return nil
}
