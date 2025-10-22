package utils

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/htetmyatthar/lothone.delivery/internal/config"
	"github.com/htetmyatthar/lothone.delivery/static"
)

// V2rayClient is to add or remove the users from the v2ray config.
type V2rayClient struct {
	Id      string `json:"id"`
	AlterId int    `json:"alterId"`
}

// Client is to store all the user info to create a vpn profile.
type Client struct {
	Id         string `json:"id"`
	AlterId    int    `json:"alterId"`
	Username   string `json:"username"`
	DeviceId   string `json:"deviceId"`
	StartDate  string `json:"startDate"`
	ExpireDate string `json:"expireDate"`
	Password   string `json:"password"` // to use with sstp and shadowsocks vpn configurations.
	Port       int    `json:"port"`
}

type InboundSettings struct {
	Clients []V2rayClient `json:"clients"`
}

type Inbound struct {
	Port           int             `json:"port"`
	Listen         string          `json:"listen"`
	Protocol       string          `json:"protocol"`
	Settings       InboundSettings `json:"settings"`
	StreamSettings json.RawMessage `json:"streamSettings"` // Handles streamSettings dynamically
}

const (
	AuthenticatedField = "b"    // to use with session values that are stored in the backend.
	URLAfterLogin      = "fsim" // to use with session values that are stored in the backend.
	DefaultAlterID     = 1

	V2boxLockedPrefix = "v2box://locked=" // to use with locked uri generation.
)

var (
	ErrWrongPassword = errors.New("Wrong password")
	ErrUserLockedOut = errors.New("User is locked out")

	PanelUsers = InitPanelUsers(*config.Admins)
)

// getMemoryUsage returns the memory usage in the current
// state of the function being called.
func GetMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.TotalAlloc
}

// Function to restart V2Ray service
func RestartService() error {
	cmd := exec.Command("sudo", "systemctl", "restart", "v2ray")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to restart service: %s, %v", string(output), err)
	}

	scmd := exec.Command("sudo", "systemctl", "restart", "shadowsocks")
	output, err = scmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to restart service: %s, %v", string(output), err)
	}

	return nil
}

// Function to validate V2Ray configuration
// Deprecated: Unecessary check?
func ValidateConfig() error {
	// Note: unecessary check? since we're just using the config that is the output from the program.
	return nil

	// cmd := exec.Command("v2ray", "-test", "-config", "configfilestring")
	// output, err := cmd.CombinedOutput()
	// if err != nil {
	// 	return fmt.Errorf("Config test failed: %s, %v", string(output), err)
	// }
	// return nil
}

// VerifyPassword verify the password Given with the correct Password.
// This method can be used to check the input password is the correct u's Password or not,
// while returning an error if there's any.
//
// The password is a correct password, only if the boolean is "true", and error is "nil".
func VerifyPassword(password string, correct string) (bool, error) {
	_, hashBytes, err := HashPassword(password)
	if err != nil {
		return false, err
	}
	userPassword, err := hex.DecodeString(correct)
	if err != nil {
		return false, err
	}
	if subtle.ConstantTimeCompare(hashBytes, userPassword) == 1 {
		return true, nil
	}
	return false, ErrWrongPassword
}

// HashPassword hashes the given password string to sha-256 hash returning the hashed values
// as a hex-dec value string and also in the form of byte slice.
func HashPassword(password string) (string, []byte, error) {
	hasher := sha256.New()
	_, err := hasher.Write([]byte(password))
	if err != nil {
		return "", nil, err
	}
	hashedBytes := hasher.Sum(nil)
	// hex is easier to maintain
	return hex.EncodeToString(hashedBytes), hashedBytes, err
}

// Message data for the gotify server.
type GotifyMessage struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	Priority int    `json:"priority"`
}

// SendNoti sends the current situation to the gotify server.
func SendNoti(gotifyServer, appToken, title, message string, priority int) error {
	msg := GotifyMessage{
		Title:    title,
		Message:  message,
		Priority: priority,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal JSON payload: %v\n", err)
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://%s/message", gotifyServer), bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("failed to create HTTP request: %v", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gotify-Key", appToken)

	client := &http.Client{
		Timeout: time.Duration(10) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to send HTTP request: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("gotify server returned non-2xx status: %d %s", resp.StatusCode, resp.Status)
		return err
	}

	return nil
}

func InitStaticServer() http.Handler {
	// Get the static subdirectory
	staticFS, err := fs.Sub(static.WebFS, "static")
	if err != nil {
		log.Fatal("failed to create sub file system: ", err)
	}

	// Create file server from embedded files
	fsHandler := http.FileServer(http.FS(staticFS))
	return fsHandler
}

// InitPanelUsers returns the panel users map that each username maps to each password which is hashed already.
// Each user should be seperated by comma(,).
// Username and password of each user should be seperated by tilde(~).
func InitPanelUsers(string) map[string]string {
	users := make(map[string]string)
	admins := strings.Split(*config.Admins, ",")
	for _, admin := range admins {
		info := strings.Split(admin, "~")
		// maps the username to the hashed password.
		password, _, err := HashPassword(info[1])
		if err != nil {
			panic("Can't hash the user passwords. Please check panel username and passwords.")
		}

		// sanity check.
		if len(info[1]) > 30 || len(info[0]) > 30 {
			panic("Usernames should not be more than 30 characters and Passwords 30.")
		}
		users[info[0]] = password
	}
	log.Println("users: ", users)
	return users
}
