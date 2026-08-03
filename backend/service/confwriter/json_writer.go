package confwriter

import (
	"backend/model"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

type JsonWriter struct {
	filename             string
	skipDhcpNotification bool
}

func New(filename string, skipDhcpNotification bool) JsonWriter {
	return JsonWriter{
		filename:             filename,
		skipDhcpNotification: skipDhcpNotification,
	}
}

type reservation struct {
	IP  string `json:"ip-address"`
	Mac string `json:"hw-address"`
}

func (jw JsonWriter) WhitelistMacs(netConfigs []model.NetConfig) error {
	reservations := mapToReservationUser(netConfigs)

	f, err := os.OpenFile(filepath.Clean(jw.filename), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return model.Error(http.StatusInternalServerError, err.Error(), "could not open output file")
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(reservations)
	if err != nil {
		return model.Error(http.StatusInternalServerError, err.Error(), "error on writing output file")
	}

	return jw.reloadConfig()
}

func mapToReservationUser(configs []model.NetConfig) []reservation {
	reservations := make([]reservation, 0, len(configs))
	for _, config := range configs {
		reservations = append(reservations, reservation{
			IP:  config.IP,
			Mac: config.Mac,
		})
	}
	return reservations
}
