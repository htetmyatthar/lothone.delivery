// This is the whole configurations needed for the server-manager to run.
// It includes the servers, and variables.
package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	gotifyAPIKeys *string
	WebHost       *string
	WebHostRegion *string
	WebHostIP     *string
	Admins        *string
	WebPort       *string
	WebCert       *string
	WebKey        *string
	V2rayPort     *string

	UserFile         *string
	ConfigFile       *string
	UserFilePrefix   *string
	ConfigFilePrefix *string

	SSTPServerURL     *string
	SSTPAdminPassword *string
	SSTPHub           *string

	SuperNodeHost *string // super node for backup data and automated emails
	SuperNodePort *string

	AdminMail *string // admin mail is for getting certificates from letsencrypt certificates.

	SessionDuration *int
	LockOutDuration *int

	GotifyServer  *string
	GotifyAPIKeys []string

	TrustedIPs    []string
	TrustedIPsMap map[string]struct{} = make(map[string]struct{}, 20)

	TemplateBasePath string = "web/templates/"
)

const (
	// SessionCookieName is the name of the cookie, the session id will be stored in.
	// Using this to make sure the cookies are not named explicitly to avoid from adavasaries
	// attempt of stealing cookies.
	SessionCookieName string = "lothoneId"

	// SessionPublic is the value of the values that is stored as the public session used for
	// logining in for and such. Value of NaN is consider public session for session id keys.
	SessionPublic string = "NaN"

	// Name of the form field the csrf token will be.
	// Deprecated: Use the csrf package in this code base to get the value.
	CSRFFormFieldName string = "token"

	// Name of the header the csrf token will be stored in.
	// Using this to make sure the cookies are not named explicityly to avoid from advasaries
	// attempt of stealing cookies.
	// Deprecated: Use the csrf package in this code base to get the value.
	CSRFHeaderFieldName string = "token"

	// Maximum allowed failed attempts for user authentications.
	MaxFailedAttempts int = 5

	// version number of this server.
	Version string = "v1.0.0"
)

func init() {
	WebHost = flag.String("hostname", "127.0.0.1", "fully qualify domain name of the server")
	WebHostRegion = flag.String("region", "127.0.0.1", "geo location region of the physical server")
	WebHostIP = flag.String("hostip", "127.0.0.1", "ipv4 or ipv6 address of the server")
	WebPort = flag.String("webport", ":8888", "port number of the control panel web server")
	WebCert = flag.String("webcert", "localhost.crt", "ssl/tls certificate for the web server")
	WebKey = flag.String("webkey", "localhost.key", "ssl/tls certificate key for the web server")

	V2rayPort = flag.String("v2rayport", "443", "port number of the v2ray proxy server")
	Admins = flag.String("admins", "lothoneadmin~lothoneadmin0,lothoneadmin1~lothoneadmin1,h~h", "panel users with username and passwords seperated by tilde(~) and for each user seperated by comma(,)")

	UserFile = flag.String("userfile", "test/users.json", "track the users of the server")
	ConfigFile = flag.String("configfile", "test/", "config file of the v2ray proxy server")
	UserFilePrefix = flag.String("userFilePrefix", "test/", "path to the user files.")
	ConfigFilePrefix = flag.String("configFilePrefix", "test/", "path to the config files.")

	GotifyServer = flag.String("gotifyserver", "noti.localhost:11111", "push nofication server domain name")
	gotifyAPIKeys = flag.String("gotifyapikeys", "somekey,somekey", "keys for using with push notification system seperated by comma(,)")

	SSTPServerURL = flag.String("sstp-server-url", "https://localhost:992/api", "url of sstp server for api access.")
	SSTPAdminPassword = flag.String("sstp-admin-password", "password", "password for sstp server api access.")
	SSTPHub = flag.String("sstp-hub", "default", "hub for sstp server api management.")

	ipList := flag.String("trusted", "127.0.0.1,192.168.100.0", "used for preventing unwanted access to the server.")
	SessionDuration = flag.Int("sessionduration", 10, "loggedin session remembered duration in minutes")
	LockOutDuration = flag.Int("lockoutduration", 30, "locking out time for wrong password in minutes")

	SuperNodeHost = flag.String("super-node-host", "127.0.0.1", "used for backup and central data repo.")
	SuperNodePort = flag.String("super-node-port", ":11111", "used for backup and central data repo.")

	AdminMail = flag.String("adminMail", "admin@lothone.shop", "use for requesting certificates from letsencrypt")

	versionFlag := flag.Bool("version", false, "Show verion number.")
	installFlag := flag.Bool("install", false, "Install server manager, get cert using certbot, setup vpn protocols")

	// parse the flags
	flag.Parse()

	TrustedIPs = strings.Split(*ipList, ",")
	for _, ip := range TrustedIPs {
		TrustedIPsMap[ip] = struct{}{}
	}

	// Check if the version flag was set
	if *versionFlag {
		fmt.Printf("LoThone V2ray VPN server Panel.%s\nVmess protocol.\n", Version)
		os.Exit(0) // Exit after showing the version
	}

	if *installFlag {
		// install all the things.
		Install()
		os.Exit(0) // Exit after installing the programs.
	}

	GotifyAPIKeys = strings.Split(*gotifyAPIKeys, ",")
}
