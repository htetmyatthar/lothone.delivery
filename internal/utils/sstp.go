package utils

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/htetmyatthar/lothone.delivery/internal/config"
)

const (
	DefaultSSTPServerURL = "https://localhost:5555/api"
	DefaultSSTPHub       = "default"
)

type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type createUserParams struct {
	HubName         string `json:"HubName_str"`
	Name            string `json:"Name_str"`
	Note            string `json:"Note_utf"`
	ExpireTime      string `json:"ExpireTime_dt"`
	AuthType        uint32 `json:"AuthType_u32"`
	AuthPassword    string `json:"Auth_Password_str"`
	UsePolicy       bool   `json:"UsePolicy_bool"`
	PolicyAccess    bool   `json:"policy:Access_bool"`
	PolicyCheckMac  bool   `json:"policy:CheckMac_bool"`
	PolicyCheckIP   bool   `json:"policy:CheckIP_bool"`
	PolicyMaxMac    uint32 `json:"policy:MaxMac_u32"`
	PolicyMaxIP     uint32 `json:"policy:MaxIP_u32"`
	PolicyMaxUpload uint32 `json:"policy:MaxUpload_u32"`
	PolicyMaxDown   uint32 `json:"policy:MaxDownload_u32"`
}

// CreateUserResponse struct
type CreateUserResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      string           `json:"id"`
	Result  createUserParams `json:"result"` // Note: Actual response might be simpler, adjust as per API docs
}

// createSSTPUser create a user in the SSTP server of configured hub.
// Docs link: https://github.com/SoftEtherVPN/SoftEtherVPN/tree/master/developer_tools/vpnserver-jsonrpc-clients/#createuser-rpc-api---create-a-user
func CreateSSTPUser(name, desc, password string, expire time.Time) (*CreateUserResponse, error) {
	params := createUserParams{
		HubName:         *config.SSTPHub,
		Name:            name,
		Note:            desc,
		ExpireTime:      expire.Format(time.RFC3339),
		AuthType:        1, // Password authentication
		AuthPassword:    password,
		UsePolicy:       true,
		PolicyAccess:    true,
		PolicyCheckMac:  true,
		PolicyCheckIP:   true,
		PolicyMaxMac:    1,
		PolicyMaxIP:     1,
		PolicyMaxUpload: 10000000, // 10Mbps in bps
		PolicyMaxDown:   10000000, // 10Mbps in bps
	}

	id := uuid.NewString()
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "CreateUser",
		Params:  params,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", *config.SSTPServerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-VPNADMIN-PASSWORD", *config.SSTPAdminPassword)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response CreateUserResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	if response.ID != id {
		return nil, errors.New("Different Request IDs.")
	}

	return &response, nil
}

type deleteUserParams struct {
	HubName string `json:"HubName_str"`
	Name    string `json:"Name_str"`
}

type DeleteUserResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      string           `json:"id"`
	Result  deleteUserParams `json:"result"`
}

// deleteSSTPUser deletes a user in the SSTP server of configured hub.
func DeleteSSTPUser(username string) (*DeleteUserResponse, error) {
	params := deleteUserParams{
		HubName: *config.SSTPHub,
		Name:    username,
	}

	id := uuid.NewString()
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "DeleteUser",
		Params:  params,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", *config.SSTPServerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-VPNADMIN-PASSWORD", *config.SSTPAdminPassword)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response DeleteUserResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	if response.ID != id {
		return nil, errors.New("Different Request IDs.")
	}

	return &response, nil
}

// enumUserParams for the request
type enumUserParams struct {
	HubName string `json:"HubName_str"`
}

// UserInfo struct for the fields we care about
type UserInfo struct {
	Name      string `json:"Name_str"`
	Note      string `json:"Note_utf"`
	Expires   string `json:"Expires_dt"`
	IsExpires bool   `json:"IsExpiresFilled_bool"`
}

// EnumUserResponse struct for the response
type EnumUserResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Result  struct {
		HubName  string     `json:"HubName_str"`
		UserList []UserInfo `json:"UserList"`
	} `json:"result"`
}

// GetSSTPUsers retrieves the list of users from the VPN server
func GetSSTPUsers() ([]UserInfo, error) {
	params := enumUserParams{
		HubName: *config.SSTPHub,
	}

	id := uuid.NewString()
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "EnumUser",
		Params:  params,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		log.Println("marshalling error")
		return nil, err
	}

	req, err := http.NewRequest("POST", *config.SSTPServerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("requesting error")
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-VPNADMIN-PASSWORD", *config.SSTPAdminPassword)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("after request error")
		return nil, err
	}
	defer resp.Body.Close()

	var response EnumUserResponse

	log.Println("response body: ", resp.Body)

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Println("decoding error")
		return nil, err
	}

	log.Println("sstp server response:\n", response)

	if response.ID != id {
		log.Println("different id error")
		return nil, errors.New("Different Request IDs.")
	}

	// parse the date part only.
	for i := range response.Result.UserList {
		response.Result.UserList[i].Expires = parseSSTPDate(response.Result.UserList[i].Expires)
	}

	return response.Result.UserList, nil
}

func parseSSTPDate(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		log.Println("dates are not in format")
		return s
	}
	return strings.Split(t.String(), " ")[0]
}
