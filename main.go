package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/nginx/nginx-plus-go-client/v2/client"
)

type Config struct {
	Nginx struct {
		IPAddress string `json:"ipAddress"`
		Port      int    `json:"port"`
	} `json:"nginx"`
	LogLevel string `json:"logLevel"`
}

const expiryThreshold = 30 * (time.Hour * 24)

func licenseExpiring(licenseData *client.NginxLicense) (bool, int64) {
	expiry := time.Unix(int64(licenseData.ActiveTill), 0) //nolint:gosec
	now := time.Now()
	timeUntilLicenseExpiry := expiry.Sub(now)
	daysUntilLicenseExpiry := int64(timeUntilLicenseExpiry.Hours() / 24)
	expiryDays := int64(expiryThreshold.Hours() / 24)
	return daysUntilLicenseExpiry < expiryDays, daysUntilLicenseExpiry
}

func usageGraceEnding(licenseData *client.NginxLicense) (bool, int64) {
	grace := time.Second * time.Duration(licenseData.Reporting.Grace) //nolint:gosec
	daysUntilUsageGraceEnds := int64(grace.Hours() / 24)
	expiryDays := int64(expiryThreshold.Hours() / 24)
	return daysUntilUsageGraceEnds < expiryDays, daysUntilUsageGraceEnds
}

func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func main() {

	duration := "1440h"
	t, err := time.ParseDuration(duration)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Duration is %f\n", t.Seconds())

	// Read config file
	cfg, err := loadConfig("config.json")
	if err != nil {
		fmt.Println(err)
	}
	// logLevel := cfg.LogLevel
	ipAddress := cfg.Nginx.IPAddress
	port := cfg.Nginx.Port
	fmt.Printf("Nginx IP Address: %s, Port: %d\n", ipAddress, port)
	httpClient := &http.Client{}
	c, err := client.NewNginxClient(fmt.Sprintf("http://%s:%d/api", ipAddress, port), client.WithHTTPClient(httpClient))
	if err != nil {
		fmt.Println(err)
	}
	// fmt.Printf("Nginx client created: %+v\n", c)
	upstreams, err := c.GetUpstreams(context.Background())
	if err != nil {
		panic(err)
	}
	for name, upstream := range *upstreams {
		fmt.Printf("Upstream name: %s, Upstream: %+v\n", name, upstream.Zone)
	}

	licenseData, err := c.GetNginxLicense(context.Background())
	if err != nil {
		fmt.Printf("could not get license data, %v\n", err)
	} else {
		fmt.Printf("License data: %+v\n", licenseData)
		fmt.Printf("License active till: %v\n", time.Unix(int64(licenseData.ActiveTill), 0)) //nolint:gosec

		if expiring, days := licenseExpiring(licenseData); expiring {
			fmt.Printf("License expiring in %d day(s)\n", days)
		}

		if ending, days := usageGraceEnding(licenseData); ending {
			fmt.Printf("Usage reporting grace period ending in %d day(s)\n", days)
		}
	}
}
