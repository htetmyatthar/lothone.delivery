// each vpn protocol like (vmess, shadowsocks, vless, ...) should use one config file, one users file each.
package utils

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/goccy/go-json"
	"github.com/htetmyatthar/lothone.delivery/internal/config"
)

const (
	VmessPrefix = "vmess://"
)

var InternalServerErr = errors.New("Internal Server Error")

// CreateV2rayUser creates a user using a pair of config file for each vpn protocol.
func CreateVmessUser(c Client, configFile string, usersFile string) error {
	configData, err := os.ReadFile(configFile)
	if err != nil {
		log.Println("Error reading file:", err)
		return InternalServerErr
	}

	userData, err := os.ReadFile(usersFile)
	if err != nil {
		log.Println("Error reading user data file: ", err)
		return InternalServerErr
	}

	var configResult map[string]json.RawMessage
	err = json.Unmarshal(configData, &configResult)
	if err != nil {
		log.Println("Error unmarshalling JSON to map in config:", err)
		return InternalServerErr
	}

	var userResult map[string]json.RawMessage
	err = json.Unmarshal(userData, &userResult)
	if err != nil {
		log.Println("Error unmarshalling JSON to map in users:", err)
		return InternalServerErr
	}

	var inbounds []Inbound
	err = json.Unmarshal(configResult["inbounds"], &inbounds)
	if err != nil {
		log.Println("Error unmarshalling 'inbounds':", err)
		return InternalServerErr
	}

	// unmarshal the "users" key into a slice of clients.
	var users []Client
	err = json.Unmarshal(userResult["clients"], &users)
	if err != nil {
		log.Println("Error unmarshalling 'users': ", err)
		return InternalServerErr
	}

	// make sure that the server id doesn't already exist.
	for _, client := range inbounds[0].Settings.Clients {
		if client.Id == c.Id {
			log.Println("Error : ", err)
			return errors.New("Internal Server Error, Server ID already exists.")
		}
	}

	newV2rayClient := V2rayClient{
		Id:      c.Id,
		AlterId: DefaultAlterID,
	}

	c.Port, _ = strconv.Atoi(*config.V2rayPort) // NOTE: ignored error

	// append the new user.
	inbounds[0].Settings.Clients = append(inbounds[0].Settings.Clients, newV2rayClient)
	users = append(users, c)

	inboundBytes, err := json.Marshal(inbounds)
	if err != nil {
		log.Println("Error marshalling modified inbounds:", err)
		return InternalServerErr
	}
	configResult["inbounds"] = inboundBytes

	usersBytes, err := json.Marshal(users)
	if err != nil {
		log.Println("Error marshalling modified users:", err)
		return InternalServerErr
	}
	userResult["clients"] = usersBytes

	finalConfigJSON, err := json.MarshalIndent(configResult, "", "  ")
	if err != nil {
		log.Println("Error marshalling final config JSON:", err)
		return InternalServerErr
	}

	finalUserJSON, err := json.MarshalIndent(userResult, "", " ")
	if err != nil {
		log.Println("Error marshalling final users JSON:", err)
		return InternalServerErr
	}

	err = os.WriteFile(configFile, finalConfigJSON, 0644)
	if err != nil {
		log.Println("Error writing modified JSON to file:", err)
		return InternalServerErr
	}

	err = os.WriteFile(usersFile, finalUserJSON, 0644)
	if err != nil {
		log.Println("Error writing modified JSON to file:", err)
		return InternalServerErr
	}

	return nil
}

// DeleteV2rayUser deletes a user from
func DeleteVmessUser(serverId, deviceId, configFile, usersFile string) (*Client, int, error) {
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

	var user V2rayClient
	var index int
	for index, user = range inbounds[0].Settings.Clients { // find user in smaller slice to hit cache?
		if user.Id == serverId {
			break
		}
	}

	if users[index].Id != serverId && users[index].DeviceId != deviceId {
		log.Println("Error invoking user deletion with incorrect information")
		return nil, http.StatusForbidden, errors.New("User's not found")
	}

	deletedUser := users[index]
	inbounds[0].Settings.Clients = slices.Delete(inbounds[0].Settings.Clients, index, index+1)
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

	return &deletedUser, http.StatusOK, nil
}

// oldClient, http_status, error
func EditVmessUser(client Client, configFile, usersFile string) (*Client, int, error) {
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

	var inbounds []Inbound
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
	for i, c := range inbounds[0].Settings.Clients {
		if client.Id == c.Id { // finding inside the config file.
			found, index = true, i
			break
		}
	}

	if !found {
		log.Println("Invalid user is being searched.")
		return nil, http.StatusBadRequest, errors.New("Bad Request")
	}

	modifiedV2rayClient := V2rayClient{
		Id:      client.Id,
		AlterId: DefaultAlterID,
	}

	modifiedClient := Client{
		Id:         client.Id,
		AlterId:    DefaultAlterID,
		Username:   client.Username,
		DeviceId:   client.DeviceId,
		StartDate:  client.StartDate,
		ExpireDate: client.ExpireDate,
	}

	oldClient := users[index]
	inbounds[0].Settings.Clients[index] = modifiedV2rayClient
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

// VmessConfig represents the VMESS configuration structure
type vmessConfig struct {
	Add      string `json:"add"`
	Aid      string `json:"aid"`
	Alpn     string `json:"alpn"`
	DeviceID string `json:"deviceID,omitempty"`
	Fp       string `json:"fp"`
	Host     string `json:"host"`
	ID       string `json:"id,omitempty"`
	Net      string `json:"net"`
	Path     string `json:"path"`
	Port     string `json:"port"`
	Ps       string `json:"ps"`
	Scy      string `json:"scy"`
	Sni      string `json:"sni"`
	Tls      string `json:"tls"`
	Type     string `json:"type"`
	V        string `json:"v"`
}

func GenerateVmessURI(data Client) (string, error) {
	subDomain := strings.Split(*config.WebHost, ".")[0]

	vmessTemplate := vmessConfig{
		Add:  *config.WebHost,
		Aid:  "1",
		Alpn: "",
		Fp:   "",
		Host: "www.youtube.com",
		ID:   data.Id,
		Net:  "tcp",
		Path: "/",
		Port: *config.V2rayPort,
		Ps: fmt.Sprintf("valid before (%s) %s-%s-%s",
			data.ExpireDate,
			subDomain,
			*config.WebHostRegion,
			data.Id[len(data.Id)-4:]),
		Scy:  "none",
		Sni:  "",
		Tls:  "",
		Type: "http",
		V:    "2",
	}

	jsonData, err := json.Marshal(vmessTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	}

	encoded := base64.StdEncoding.EncodeToString(jsonData)
	return VmessPrefix + encoded, nil
}

// GenerateLockedURI generates a locked VMESS URI
func GenerateVmessLockedURI(data Client) (string, error) {
	if data.DeviceId == "" {
		return "", fmt.Errorf("unable to generate locked QR without device id")
	}

	subDomain := strings.Split(*config.WebHost, ".")[0]

	vmessTemplate := vmessConfig{
		Add:      *config.WebHost,
		Aid:      "1",
		Alpn:     "",
		DeviceID: data.DeviceId,
		Fp:       "",
		Host:     "www.youtube.com",
		ID:       data.Id,
		Net:      "tcp",
		Path:     "/",
		Port:     *config.V2rayPort,
		Ps: fmt.Sprintf("valid before (%s) %s-%s-%s",
			data.ExpireDate,
			subDomain,
			*config.WebHostRegion,
			data.Id[len(data.Id)-4:]),
		Scy:  "none",
		Sni:  "",
		Tls:  "",
		Type: "http",
		V:    "2",
	}

	jsonData, err := json.Marshal(vmessTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	}

	unlockedQR := VmessPrefix + base64.StdEncoding.EncodeToString(jsonData)
	lockedQR := base64.StdEncoding.EncodeToString([]byte(unlockedQR))
	return V2boxLockedPrefix + lockedQR, nil
}
