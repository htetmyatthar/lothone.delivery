package utils

import (
	"fmt"
	"strconv"

	"github.com/htetmyatthar/lothone.delivery/internal/config"
)

// AccountType is the type of the user account to distinguish different vpn protocol.
type AccountType int

// to use inside the Client struct of type field.
const (
	VmessAccountType AccountType = iota + 1
	ShadowsocksAccountType
	SstpAccountType
)

// String converts AccountType to a string.
func (a AccountType) String() string {
	return strconv.Itoa(int(a))
}

// Protocol returns the vpn protocol of the account type uses in string.
func (a AccountType) Protocol() (protocol string) {
	switch a {
	case VmessAccountType:
		protocol = "vmess"
	case ShadowsocksAccountType:
		protocol = "shadowsocks"
	case SstpAccountType:
		protocol = "sstp"
	}
	return protocol
}

// Filename gets the filename of each v2ray protocol configuration
// NOTE: Filename only gets the filenames for v2ray protocols that are prefixed with configFilePrefix and userFilePrefix.
func (a AccountType) Filename() (string, string) {
	var configFilename string
	var usersFilename string
	switch a {
	case VmessAccountType:
		configFilename = "vmess.json"
		usersFilename = "vmess_users.json"
	case ShadowsocksAccountType:
		configFilename = "shadowsocks.json"
		usersFilename = "shadowsocks_users.json"
	}
	return (*config.ConfigFilePrefix + configFilename), (*config.UserFilePrefix + usersFilename)
}

// ParseAccountType converts a string to an AccountType. Returns an error if the given string is an invalid AccountType.
func ParseAccountType(s string) (AccountType, error) {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid account type: %s", s)
	}

	switch AccountType(val) {
	case VmessAccountType, ShadowsocksAccountType, SstpAccountType:
		return AccountType(val), nil
	default:
		return 0, fmt.Errorf("unknown account type: %d", val)
	}
}
