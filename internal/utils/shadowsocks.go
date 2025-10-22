package utils

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/htetmyatthar/lothone.delivery/internal/config"
)

const ShadowsocksPrefix = "ss://"

// ShadowsocksSettings represents the settings object in Shadowsocks inbounds
type ShadowsocksSettings struct {
	Method   string `json:"method"`
	Password string `json:"password"`
	Network  string `json:"network"`
	Level    int    `json:"level"`
	Ota      bool   `json:"ota"`
}

// Inbound represents an inbound configuration
type ShadowsocksInbound struct {
	Port     int                 `json:"port"`
	Listen   string              `json:"listen,omitempty"`
	Protocol string              `json:"protocol"`
	Settings ShadowsocksSettings `json:"settings"`
}

// Client represents user information (simplified for Shadowsocks)
type ShadowsocksClient struct {
	Port     int    `json:"port"`
	Password string `json:"password"`
}

// CreateShadowsocksUser creates a new Shadowsocks user
func CreateShadowsocksUser(c Client, configFile string, usersFile string) error {
	// Read config file
	configData, err := os.ReadFile(configFile)
	if err != nil {
		log.Println("Error reading config file:", err)
		return InternalServerErr
	}

	// Read users file
	userData, err := os.ReadFile(usersFile)
	if err != nil {
		log.Println("Error reading user data file:", err)
		return InternalServerErr
	}

	// Unmarshal config JSON
	var configResult map[string]json.RawMessage
	err = json.Unmarshal(configData, &configResult)
	if err != nil {
		log.Println("Error unmarshalling config JSON:", err)
		return InternalServerErr
	}

	// Unmarshal users JSON
	var userResult map[string]json.RawMessage
	err = json.Unmarshal(userData, &userResult)
	if err != nil {
		log.Println("Error unmarshalling users JSON:", err)
		return InternalServerErr
	}

	// Unmarshal inbounds
	var inbounds []ShadowsocksInbound
	err = json.Unmarshal(configResult["inbounds"], &inbounds)
	if err != nil {
		log.Println("Error unmarshalling 'inbounds':", err)
		return InternalServerErr
	}

	// Unmarshal users
	var users []Client
	err = json.Unmarshal(userResult["clients"], &users)
	if err != nil {
		log.Println("Error unmarshalling 'clients':", err)
		return InternalServerErr
	}

	// Check if password already exists that can uniquely identified a user for frontend.
	for _, inbound := range inbounds {
		if inbound.Settings.Password == c.Password {
			log.Println("Error: password is already in use")
			return InternalServerErr
		}
	}

	// Get next available port
	port := getNextPort(inbounds)

	// Create new Shadowsocks inbound
	newInbound := ShadowsocksInbound{
		Port:     port,
		Listen:   "0.0.0.0",
		Protocol: "shadowsocks",
		Settings: ShadowsocksSettings{
			Method:   "aes-128-gcm", // Default encryption method
			Password: c.Password,
			Network:  "tcp,udp",
			Level:    1,
			Ota:      false,
		},
	}

	c.Port = port // set the port.

	// Append new configurations
	inbounds = append(inbounds, newInbound)
	users = append(users, c)

	// Marshal updated inbounds
	inboundBytes, err := json.Marshal(inbounds)
	if err != nil {
		log.Println("Error marshalling modified inbounds:", err)
		return InternalServerErr
	}
	configResult["inbounds"] = inboundBytes

	// Marshal updated users
	usersBytes, err := json.Marshal(users)
	if err != nil {
		log.Println("Error marshalling modified users:", err)
		return InternalServerErr
	}
	userResult["clients"] = usersBytes

	// Marshal final JSON with indentation
	finalConfigJSON, err := json.MarshalIndent(configResult, "", "  ")
	if err != nil {
		log.Println("Error marshalling final config JSON:", err)
		return InternalServerErr
	}

	finalUserJSON, err := json.MarshalIndent(userResult, "", "  ")
	if err != nil {
		log.Println("Error marshalling final users JSON:", err)
		return InternalServerErr
	}

	// Write back to files
	err = os.WriteFile(configFile, finalConfigJSON, 0644)
	if err != nil {
		log.Println("Error writing modified config JSON to file:", err)
		return InternalServerErr
	}

	err = os.WriteFile(usersFile, finalUserJSON, 0644)
	if err != nil {
		log.Println("Error writing modified users JSON to file:", err)
		return InternalServerErr
	}

	err = AllowPort(c.Port)
	if err != nil {
		log.Println("port error. please fix ufw.: ", err)
		return InternalServerErr
	}

	return nil
}

// getNextPort finds the next available port starting from 10000
// just to use with shadowsocks since those can only be made one user by one port.
func getNextPort(inbounds []ShadowsocksInbound) int {
	if len(inbounds) == 0 {
		return 10000 // Starting port if no users exist
	}

	// Collect all used ports
	ports := make([]int, len(inbounds))
	for i, inbound := range inbounds {
		ports[i] = inbound.Port
	}

	// Sort ports to find gaps or next available
	sort.Ints(ports)

	// Start from 10000 and find the first available port
	nextPort := 10000
	for _, port := range ports {
		if port > nextPort {
			return nextPort // Return the first gap found
		}
		if port == nextPort {
			nextPort++ // Increment if current port is taken
		}
	}
	return nextPort // Return next port after highest used
}

// EditShadowsocksUser edits an existing Shadowsocks user's password
func EditShadowsocksUser(client Client, configFile, usersFile string) (*Client, int, error) {
	configData, err := os.ReadFile(configFile)
	if err != nil {
		log.Println("Error reading file:", err)
		return nil, http.StatusInternalServerError, InternalServerErr
	}

	userData, err := os.ReadFile(usersFile)
	if err != nil {
		log.Println("Error reading user data file: ", err)
		return nil, http.StatusInternalServerError, InternalServerErr
	}

	var configResult map[string]json.RawMessage
	err = json.Unmarshal(configData, &configResult)
	if err != nil {
		log.Println("Error unmarshalling JSON to map in config:", err)
		return nil, http.StatusInternalServerError, InternalServerErr
	}

	var userResult map[string]json.RawMessage
	err = json.Unmarshal(userData, &userResult)
	if err != nil {
		log.Println("Error unmarshalling JSON to map in users:", err)
		return nil, http.StatusInternalServerError, InternalServerErr
	}

	var inbounds []ShadowsocksInbound
	err = json.Unmarshal(configResult["inbounds"], &inbounds)
	if err != nil {
		log.Println("Error unmarshalling 'inbounds':", err)
		return nil, http.StatusInternalServerError, InternalServerErr
	}

	var users []Client
	err = json.Unmarshal(userResult["clients"], &users)
	if err != nil {
		log.Println("Error unmarshalling 'users': ", err)
		return nil, http.StatusInternalServerError, InternalServerErr
	}

	found, index := false, -1
	for i, c := range users {
		if client.Password == c.Password { // finding inside the config file.
			found, index = true, i
			break
		}
	}

	if !found {
		log.Println("Invalid user is being searched.")
		return nil, http.StatusBadRequest, errors.New("Bad Request")
	}

	// Create new Shadowsocks inbound
	modifiedInbound := ShadowsocksInbound{
		Port:     inbounds[index].Port,
		Listen:   "0.0.0.0",
		Protocol: "shadowsocks",
		Settings: ShadowsocksSettings{
			Method:   "aes-128-gcm", // Default encryption method
			Password: client.Password,
			Network:  "tcp,udp",
			Level:    1,
			Ota:      false,
		},
	}

	// newly modified client.
	modifiedClient := Client{
		AlterId:    DefaultAlterID,
		Username:   client.Username,
		DeviceId:   client.DeviceId,
		StartDate:  client.StartDate,
		ExpireDate: client.ExpireDate,
		Password:   client.Password,
	}

	oldClient := users[index]
	inbounds[index] = modifiedInbound
	users[index] = modifiedClient

	inboundBytes, err := json.Marshal(inbounds)
	if err != nil {
		log.Println("Error marshalling modified inbounds:", err)
		return nil, http.StatusInternalServerError, InternalServerErr
	}
	configResult["inbounds"] = inboundBytes

	usersBytes, err := json.Marshal(users)
	if err != nil {
		log.Println("Error marshalling modified users:", err)
		return nil, http.StatusInternalServerError, InternalServerErr
	}
	userResult["clients"] = usersBytes

	finalConfigJSON, err := json.MarshalIndent(configResult, "", "  ")
	if err != nil {
		log.Println("Error marshalling final config JSON:", err)
		return nil, http.StatusInternalServerError, InternalServerErr
	}

	finalUserJSON, err := json.MarshalIndent(userResult, "", " ")
	if err != nil {
		log.Println("Error marshalling final users JSON:", err)
		return nil, http.StatusInternalServerError, InternalServerErr
	}

	err = os.WriteFile(configFile, finalConfigJSON, 0644)
	if err != nil {
		log.Println("Error writing modified JSON to file:", err)
		return nil, http.StatusInternalServerError, InternalServerErr
	}

	err = os.WriteFile(usersFile, finalUserJSON, 0644)
	if err != nil {
		log.Println("Error writing modified JSON to file:", err)
		return nil, http.StatusInternalServerError, InternalServerErr
	}
	return &oldClient, http.StatusOK, nil
}

func DeleteShadowsocksUser(password, deviceId, configFile, usersFile string) (*Client, int, error) {
	configData, err := os.ReadFile(configFile)
	if err != nil {
		log.Println("Error reading file:", err)
		return nil, http.StatusInternalServerError, errors.New("Internal server error")
	}

	userData, err := os.ReadFile(usersFile)
	if err != nil {
		log.Println("Error reading user data file: ", err)
		return nil, http.StatusInternalServerError, errors.New("Internal server error")
	}

	var configResult map[string]json.RawMessage
	err = json.Unmarshal(configData, &configResult)
	if err != nil {
		log.Println("Error unmarshalling JSON to map in config:", err)
		return nil, http.StatusInternalServerError, errors.New("Internal server error")
	}

	var userResult map[string]json.RawMessage
	err = json.Unmarshal(userData, &userResult)
	if err != nil {
		log.Println("Error unmarshalling JSON to map in users:", err)
		return nil, http.StatusInternalServerError, errors.New("Internal server error")
	}

	var inbounds []Inbound
	err = json.Unmarshal(configResult["inbounds"], &inbounds)
	if err != nil {
		log.Println("Error unmarshalling 'inbounds':", err)
		return nil, http.StatusInternalServerError, errors.New("Internal server error")
	}

	var users []Client
	err = json.Unmarshal(userResult["clients"], &users)
	if err != nil {
		log.Println("Error unmarshalling 'users': ", err)
		return nil, http.StatusInternalServerError, errors.New("Internal server error")
	}

	var user Client
	var index int
	for index, user = range users {
		if user.Password == password {
			break
		}
	}

	if users[index].Password != password && users[index].DeviceId != deviceId {
		log.Println("Error invoking user deletion with incorrect information")
		return nil, http.StatusForbidden, errors.New("User's not found")
	}

	deletedUser := users[index]
	inbounds = slices.Delete(inbounds, index, index+1)
	users = slices.Delete(users, index, index+1)

	inboundBytes, err := json.Marshal(inbounds)
	if err != nil {
		log.Println("Error marshalling modified inbounds:", err)
		return nil, http.StatusInternalServerError, errors.New("Internal server error")
	}
	configResult["inbounds"] = inboundBytes

	usersBytes, err := json.Marshal(users)
	if err != nil {
		log.Println("Error marshalling modified users:", err)
		return nil, http.StatusInternalServerError, errors.New("Internal server error")
	}
	userResult["clients"] = usersBytes

	finalConfigJSON, err := json.MarshalIndent(configResult, "", "  ")
	if err != nil {
		log.Println("Error marshalling final config JSON:", err)
		return nil, http.StatusInternalServerError, errors.New("Internal server error")
	}

	finalUserJSON, err := json.MarshalIndent(userResult, "", " ")
	if err != nil {
		log.Println("Error marshalling final users JSON:", err)
		return nil, http.StatusInternalServerError, errors.New("Internal server error")
	}

	err = os.WriteFile(configFile, finalConfigJSON, 0644)
	if err != nil {
		log.Println("Error writing modified JSON to file:", err)
		return nil, http.StatusInternalServerError, errors.New("Internal server error")
	}

	err = os.WriteFile(usersFile, finalUserJSON, 0644)
	if err != nil {
		log.Println("Error writing modified JSON to file:", err)
		return nil, http.StatusInternalServerError, errors.New("Internal server error")
	}

	err = DeletePort(deletedUser.Port)
	if err != nil {
		log.Println("port error. please fix ufw.: ", err)
		return nil, 0, InternalServerErr
	}

	return &deletedUser, http.StatusOK, nil
}

// ShadowsocksConfig represents the Shadowsocks configuration structure
type ShadowsocksConfig struct {
	Method   string `json:"method"`   // Encryption method (e.g., "aes-128-gcm")
	Password string `json:"password"` // Password for the connection
	Host     string `json:"host"`     // Server hostname or IP
	Port     int    `json:"port"`     // Server port
	Ps       string `json:"ps"`       // Name or description (optional, for client display)
}

// GenerateShadowsocksLockedURI generates a locked Shadowsocks URI
func GenerateShadowsocksLockedURI(data Client) (string, error) {
	if data.DeviceId == "" {
		return "", fmt.Errorf("unable to generate locked URI without device id")
	}

	subDomain := strings.Split(*config.WebHost, ".")[0]

	// Define the Shadowsocks configuration with DeviceID
	ssConfig := ShadowsocksConfig{
		Method:   "aes-128-gcm",
		Password: data.Password,
		Host:     *config.WebHost,
		Port:     data.Port,
		Ps: fmt.Sprintf("valid before (%s) %s-%s-%s [locked:%s]",
			data.ExpireDate,
			subDomain,
			*config.WebHostRegion,
			data.Password[len(data.Password)-4:],
			data.DeviceId), // Include DeviceId in the name
	}

	// Construct the base part: method:password
	basePart := fmt.Sprintf("%s:%s", ssConfig.Method, ssConfig.Password)
	// Base64 encode the base part
	encodedBase := base64.StdEncoding.EncodeToString([]byte(basePart))
	// Construct the standard URI
	standardURI := fmt.Sprintf("%s%s@%s:%d#%s",
		ShadowsocksPrefix,
		encodedBase,
		ssConfig.Host,
		ssConfig.Port,
		url.QueryEscape(ssConfig.Ps))

	// Double-encode the standard URI for locking
	lockedURI := base64.StdEncoding.EncodeToString([]byte(standardURI))
	return V2boxLockedPrefix + lockedURI, nil
}

// GenerateShadowsocksURI generates a standard Shadowsocks URI
func GenerateShadowsocksURI(data Client) (string, error) {
	subDomain := strings.Split(*config.WebHost, ".")[0]

	// Define the Shadowsocks configuration
	ssConfig := ShadowsocksConfig{
		Method:   "aes-128-gcm", // Matches your inbound settings
		Password: data.Password,
		Host:     *config.WebHost,
		Port:     data.Port, // Assuming Shadowsocks uses the same port as VMESS
		Ps: fmt.Sprintf("valid before (%s) %s-%s-%s",
			data.ExpireDate,
			subDomain,
			*config.WebHostRegion,
			data.Password[len(data.Password)-4:]), // Consistent naming with VMESS
	}

	// Validate required fields
	if ssConfig.Password == "" {
		return "", fmt.Errorf("password is required for Shadowsocks URI")
	}

	// Construct the base part: method:password
	basePart := fmt.Sprintf("%s:%s", ssConfig.Method, ssConfig.Password)
	// Base64 encode the base part
	encodedBase := base64.StdEncoding.EncodeToString([]byte(basePart))
	// Construct the full URI
	uri := fmt.Sprintf("%s%s@%s:%d#%s",
		ShadowsocksPrefix,
		encodedBase,
		ssConfig.Host,
		ssConfig.Port,
		url.QueryEscape(ssConfig.Ps)) // URL-encode the name for safety

	return uri, nil
}
