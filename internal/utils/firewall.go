package utils

import (
	"fmt"
	"os/exec"
)

// AllowPort opens the specified port for both TCP and UDP in UFW
func AllowPort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port number must be between 1 and 65535")
	}

	// Command to allow TCP
	tcpCmd := exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", port))
	if err := tcpCmd.Run(); err != nil {
		return fmt.Errorf("failed to allow TCP port %d: %v", port, err)
	}

	// Command to allow UDP
	udpCmd := exec.Command("ufw", "allow", fmt.Sprintf("%d/udp", port))
	if err := udpCmd.Run(); err != nil {
		return fmt.Errorf("failed to allow UDP port %d: %v", port, err)
	}

	fmt.Printf("Successfully allowed port %d for TCP and UDP\n", port)
	return nil
}

// DeletePort removes the specified port's rules from UFW
func DeletePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port number must be between 1 and 65535")
	}

	// Command to delete TCP rule
	tcpCmd := exec.Command("ufw", "delete", "allow", fmt.Sprintf("%d/tcp", port))
	if err := tcpCmd.Run(); err != nil {
		return fmt.Errorf("failed to delete TCP port %d: %v", port, err)
	}

	// Command to delete UDP rule
	udpCmd := exec.Command("ufw", "delete", "allow", fmt.Sprintf("%d/udp", port))
	if err := udpCmd.Run(); err != nil {
		return fmt.Errorf("failed to delete UDP port %d: %v", port, err)
	}

	fmt.Printf("Successfully deleted port %d rules for TCP and UDP\n", port)
	return nil
}
