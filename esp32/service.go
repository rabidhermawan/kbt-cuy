package esp32

import (
	"fmt"
	"net/http"
	"time"
)

// TriggerLock sends a command to the ESP32 to open the lock
func TriggerLock(stationIP string, action string) error {
	// SIMULATION MODE
	// If using the demo IP from seed data, we just print to console
	if stationIP == "192.168.1.50" {
		fmt.Printf("[SIMULATION] Connecting to %s...\n", stationIP)
		time.Sleep(500 * time.Millisecond) // Simulate network latency

		if action == "open" {
			fmt.Printf("[SIMULATION] Command: OPEN LOCK. Holding for 10 seconds.\n")

			// Simulate the hardware timer logic in the background
			go func() {
				time.Sleep(10 * time.Second)
				fmt.Printf("[SIMULATION] Auto-Closing lock at %s\n", stationIP)
			}()
			return nil
		}
		return fmt.Errorf("unknown command")
	}

	// REAL HARDWARE IMPLEMENTATION
	// This sends a real HTTP GET request to the ESP32's web server

	client := http.Client{
		Timeout: 5 * time.Second, // Don't hang the backend if ESP is offline
	}

	// Construct URL: http://192.168.1.105/open
	url := fmt.Sprintf("http://%s/%s", stationIP, action)
	fmt.Printf("[HARDWARE] Sending Request: %s\n", url)

	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("[HARDWARE] Connection Failed: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("station returned error status: %s", resp.Status)
	}

	return nil
}

// RetryOpen allows the user to trigger the lock one more time
func RetryOpen(stationIP string) error {
	return TriggerLock(stationIP, "open")
}
