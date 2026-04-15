package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type HomewizardSource struct {
	URL       string
	Interval  time.Duration
	Token     string
	TokenFile string
	client    *http.Client
}

type homewizardV2Response struct {
	EnergyImportT1Kwh float64 `json:"energy_import_t1_kwh"`
	EnergyImportT2Kwh float64 `json:"energy_import_t2_kwh"`
	EnergyExportT1Kwh float64 `json:"energy_export_t1_kwh"`
	EnergyExportT2Kwh float64 `json:"energy_export_t2_kwh"`
	PowerW             float64 `json:"power_w"`
	VoltageL1V         float64 `json:"voltage_l1_v"`
	VoltageL2V         float64 `json:"voltage_l2_v"`
	VoltageL3V         float64 `json:"voltage_l3_v"`
	CurrentL1A         float64 `json:"current_l1_a"`
	CurrentL2A         float64 `json:"current_l2_a"`
	CurrentL3A         float64 `json:"current_l3_a"`
	Tariff             float64 `json:"tariff"`
	External           []struct {
		Type  string  `json:"type"`
		Value float64 `json:"value"`
	} `json:"external"`
}

func (h *HomewizardSource) Start(ctx context.Context) (<-chan Reading, error) {
	h.client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}

	if h.Token == "" && h.TokenFile != "" {
		data, err := os.ReadFile(h.TokenFile)
		if err == nil {
			h.Token = string(bytes.TrimSpace(data))
		}
	}

	if h.Token == "" {
		logrus.Info("No Homewizard token found. Initiating authorization...")
		token, err := h.authorize(ctx)
		if err != nil {
			return nil, fmt.Errorf("authorization failed: %w", err)
		}
		h.Token = token
		if h.TokenFile != "" {
			if err := os.WriteFile(h.TokenFile, []byte(token), 0600); err != nil {
				logrus.Errorf("Failed to save token to %s: %v", h.TokenFile, err)
			}
		}
	}

	readings := make(chan Reading)
	if h.Interval == 0 {
		h.Interval = 5 * time.Second
	}

	go func() {
		defer close(readings)
		ticker := time.NewTicker(h.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				reading, err := h.fetch()
				if err != nil {
					logrus.Errorf("Error fetching from Homewizard: %v", err)
					continue
				}
				readings <- *reading
			}
		}
	}()

	return readings, nil
}

func (h *HomewizardSource) authorize(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/api/user", h.URL)
	payload, _ := json.Marshal(map[string]string{"name": "local/prometheus-p1-exporter"})

	// First attempt
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Version", "2")

	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		logrus.Warn("Action required: Press the physical button on your HomeWizard device NOW to authorize this exporter.")
		
		// Retry loop for 30 seconds
		timeout := time.After(45 * time.Second)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-timeout:
				return "", fmt.Errorf("authorization timed out: button was not pressed in time")
			case <-ticker.C:
				req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Api-Version", "2")
				
				resp, err := h.client.Do(req)
				if err != nil {
					continue
				}
				
				if resp.StatusCode == http.StatusOK {
					var result struct {
						Token string `json:"token"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
						resp.Body.Close()
						logrus.Info("Authorization successful!")
						return result.Token, nil
					}
				}
				resp.Body.Close()
			}
		}
	}

	return "", fmt.Errorf("unexpected status during authorization: %d", resp.StatusCode)
}

func (h *HomewizardSource) fetch() (*Reading, error) {
	req, _ := http.NewRequest("GET", h.URL+"/api/measurement", nil)
	req.Header.Set("Authorization", "Bearer "+h.Token)
	req.Header.Set("X-Api-Version", "2")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var data homewizardV2Response
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	reading := &Reading{
		ElectricityUsageLow:     &data.EnergyImportT1Kwh,
		ElectricityUsageHigh:    &data.EnergyImportT2Kwh,
		ElectricityReturnedLow:  &data.EnergyExportT1Kwh,
		ElectricityReturnedHigh: &data.EnergyExportT2Kwh,
		ActiveTariff:            &data.Tariff,

		ActiveVoltageL1V: &data.VoltageL1V,
		ActiveVoltageL2V: &data.VoltageL2V,
		ActiveVoltageL3V: &data.VoltageL3V,
		ActiveCurrentL1A: &data.CurrentL1A,
		ActiveCurrentL2A: &data.CurrentL2A,
		ActiveCurrentL3A: &data.CurrentL3A,
	}

	// Map power_w to Delivered and Generated
	delivered := 0.0
	generated := 0.0
	if data.PowerW > 0 {
		delivered = data.PowerW / 1000.0
	} else if data.PowerW < 0 {
		generated = -data.PowerW / 1000.0
	}
	reading.ActualElectricityDelivered = &delivered
	reading.ActualElectricityGenerated = &generated

	// Gas usage
	for _, ext := range data.External {
		if ext.Type == "gas_meter" {
			val := ext.Value
			reading.GasUsage = &val
			break
		}
	}

	return reading, nil
}
