package esp32

import (
	"fmt"
	"time"
)

// TriggerLock simulates sending a command to the ESP32
// In production, this would make an HTTP request to the station's IP
func TriggerLock(stationIP string, action string) error {
	// Simulate network latency
	fmt.Printf("[ESP32] Connecting to %s...\n", stationIP)
	time.Sleep(500 * time.Millisecond)

	// Simulate command execution
	if action == "open" {
		fmt.Printf("[ESP32] Command: OPEN LOCK. Holding for 10 seconds.\n")

		// In a real app, this might be a goroutine that sends a close signal later,
		// or the ESP32 handles the timer itself.
		go func() {
			time.Sleep(10 * time.Second)
			fmt.Printf("[ESP32] Auto-Closing lock at %s\n", stationIP)
		}()

		return nil
	}

	return fmt.Errorf("unknown command")
}

// RetryOpen allows the user to trigger the lock one more time
func RetryOpen(stationIP string) error {
	fmt.Println("[ESP32] Retrying Open Command...")
	return TriggerLock(stationIP, "open")
}
