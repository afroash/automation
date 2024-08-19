package main

import (
	"fmt"
	"os"
	"time"

	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/platform"
	"gopkg.in/yaml.v2"
)

type Device struct {
	IP       string `yaml:"ip"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Model    string `yaml:"model"`
}

type Config struct {
	Devices []Device `yaml:"devices"`
}

// readConfig reads the configuration from a file and returns a Config struct
func readConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}

func getRunningConfig(d Device) (string, error) {
	p, err := platform.NewPlatform(
		d.Model,
		d.IP,
		options.WithAuthNoStrictKey(),
		options.WithAuthUsername(d.Username),
		options.WithAuthPassword(d.Password),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create platform: %w", err)
	}

	driver, err := p.GetNetworkDriver()
	if err != nil {
		return "", fmt.Errorf("failed to fetch network driver from the platform: %w", err)
	}

	err = driver.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open driver: %w", err)
	}

	defer driver.Close()

	//Using show version as a replacement for show running-config
	r, err := driver.SendCommand("show version")
	if err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	return r.Result, nil
}

func processDevice(d Device, ch chan<- error) {
	fmt.Printf("Getting running config for %s...\n", d.IP)
	r, err := getRunningConfig(d)
	if err != nil {
		ch <- fmt.Errorf("failed to get running config: %v", err)
		return
	}
	//fmt.Println(r)
	f := fmt.Sprintf("%s-running-config.txt", d.IP)
	err = os.WriteFile(f, []byte(r), 0644)
	if err != nil {
		ch <- fmt.Errorf("failed to save running config for %s: %v", d.IP, err)
	} else {
		fmt.Printf("Running config saved for %s to %s\n", d.IP, f)
	}
	ch <- nil
}

func main() {
	//time how long it takes to run the program
	start := time.Now()
	config, err := readConfig("devices.yaml")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	numDevices := len(config.Devices)
	ch := make(chan error, numDevices)

	for _, d := range config.Devices {
		go processDevice(d, ch)
	}

	for i := 0; i < numDevices; i++ {
		if err := <-ch; err != nil {
			fmt.Println(err)
		}
	}

	elasped := time.Since(start)
	fmt.Printf("Time taken: %s\n", elasped)
}
