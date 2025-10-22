package config

import (
	"fmt"
	"os"
	"os/exec"
)

const WebRoot string = "/var/www/html"

// we should use it or not? still deciding.
// var NginxConf string = "/etc/nginx/conf.d" + *WebHost

// Installs the whole panel in one go.
func Install() {
	//BUG: need some more ERROR handling
	err := certsInstall()
	if err != nil {
		fmt.Println(err)
	}
	err = vmessInstall()
	if err != nil {
		fmt.Println(err)
	}
	err = shadowsocksInstall()
	if err != nil {
		fmt.Println(err)
	}
	err = sstpInstall()
	if err != nil {
		fmt.Println(err)
	}
	err = serverManagerInstall()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("The whole panel successfully installed.")
}

func serverManagerInstall() error {
	// BUG: make sure the nginx is not running.
	cmd := exec.Command("sudo", "systemctl", "stop", "nginx")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to stop the nginx service before the server manager service initialiation: %s, %v", string(output), err)
	}

	cmd = exec.Command("cp", "v2ray/server-manager.service", "/etc/systemd/system/server-manager.service")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to copy the server-manager service file for the vpn services: %s, %v", string(output), err)
	}

	cmd = exec.Command("sudo", "systemctl", "daemon-reload")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to restart the systemd daemon: %s, %v", string(output), err)
	}

	cmd = exec.Command("sudo", "systemctl", "restart", "server-manager.service")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to restart the server-manager systemd unit: %s, %v", string(output), err)
	}

	fmt.Println("Server Manager Service installed successfully.")
	return nil
}

// BUG: there's still more bug in this, certificates are not being requested after run, check this after.
func certsInstall() error {
	// todo: install nginx and get the certs from letsencrypt

	var nginxServerConfig string = `server {
			listen 80;
			server_name "` + *WebHost + `";

			root "/var/www/html";

			location ~ /.well-known/acme-challenge {
				allow all;
			}
		}
	`

	err := os.WriteFile("/etc/nginx/conf.d/"+*WebHost, []byte(nginxServerConfig), 0644)
	if err != nil {
		return fmt.Errorf("Failed to overwrite the v2ray service file: %v", err)
	}

	cmd := exec.Command("mkdir", "-p", "/var/www/html")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to create a webroot for certificates: %s, %v", string(output), err)
	}

	cmd = exec.Command("chown", "www-data:www-data", "/var/www/html", "-R")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to create a webroot for certificates: %s, %v", string(output), err)
	}

	cmd = exec.Command("sudo", "systemctl", "reload", "nginx")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to reload Nginx service: %s, %v", string(output), err)
	}

	// Obtain SSL certificate with Certbot using webroot
	cmd = exec.Command("certbot", "certonly", "--webroot", "--agree-tos", "--key-type", "rsa", "--email", *AdminMail, "-d", *WebHost, "-w", "/var/www/html")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to request a certificate from letsencrypt: %s, %v", string(output), err)
	}

	cmd = exec.Command("sudo", "systemctl", "stop", "nginx")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to stop the Nginx service: %s, %v", string(output), err)
	}

	cmd = exec.Command("sudo", "apt", "update", "&&", "sudo", "apt", "install", "-y", "nginx", "certbot", "python3-certbot-nginx")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to restart service: %s, %v", string(output), err)
	}

	fmt.Println("certificates installed successfully")
	return nil
}

func vmessInstall() error {
	const serviceFile string = `[Unit]
		Description=V2Ray Vmess Service
		Documentation=https://www.v2ray.com/ https://www.v2fly.org/
		After=network-online.target nss-lookup.target

		[Service]
		Type=simple
		CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
		AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
		DynamicUser=true
		NoNewPrivileges=true
		Environment=V2RAY_LOCATION_ASSET=/etc/v2ray
		ExecStart=/usr/bin/v2ray -config /etc/v2ray/vmess.json
		Restart=on-failure

		[Install]
		WantedBy=multi-user.target`

	cmd := exec.Command("sudo", "apt", "install", "v2ray")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to restart service: %s, %v", string(output), err)
	}

	cmd = exec.Command("touch", "/usr/lib/systemd/system/v2ray.service")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to find the v2ray service file: %s, %v", string(output), err)
	}

	err = os.WriteFile("/usr/lib/systemd/system/v2ray.service", []byte(serviceFile), 0644)
	if err != nil {
		return fmt.Errorf("Failed to overwrite the v2ray service file: %v", err)
	}

	cmd = exec.Command("cp", "v2ray/vmess.json", "/etc/v2ray/vmess.json")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to copy the vmess config default file for the vmess service: %s, %v", string(output), err)
	}

	// BUG: can't create the file, why? I touched it and still can't use this?
	cmd = exec.Command("cp", "v2ray/vmess_users.json", "/etc/v2ray_users/vmess_users.json")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to copy the vmess users default file for the vmess service: %s, %v", string(output), err)
	}

	// BUG: executable file not found in $PATH
	cmd = exec.Command("sudo", "systemctl", "restart", "v2ray.service")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to start the vmess service: %s, %v", string(output), err)
	}
	fmt.Println("vmess panel successfully installed.")
	return nil
}

func shadowsocksInstall() error {
	const serviceFile = `[Unit]
		Description=V2Ray Shadowsocks Service
		Documentation=https://www.v2ray.com/ https://www.v2fly.org/
		After=network-online.target nss-lookup.target

		[Service]
		Type=simple
		CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
		AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
		DynamicUser=true
		NoNewPrivileges=true
		Environment=V2RAY_LOCATION_ASSET=/etc/v2ray
		ExecStart=/usr/bin/v2ray -config /etc/v2ray/shadowsocks.json
		Restart=on-failure

		[Install]
		WantedBy=multi-user.target`

	cmd := exec.Command("sudo", "apt", "install", "v2ray")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to restart service: %s, %v", string(output), err)
	}

	cmd = exec.Command("touch", "/usr/lib/systemd/system/shadowsocks.service")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to find the v2ray service file: %s, %v", string(output), err)
	}

	err = os.WriteFile("/usr/lib/systemd/system/shadowsocks.service", []byte(serviceFile), 0644)
	if err != nil {
		return fmt.Errorf("Failed to overwrite the v2ray service file: %s, %v", string(output), err)
	}

	cmd = exec.Command("cp", "v2ray/shadowsocks.json", "/etc/v2ray/shadowsocks.json")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to copy the shadowsocks config default file for the shadowsocks service: %s, %v", string(output), err)
	}

	// BUG: don't have any file even if I touched it. I think its due to the directory
	cmd = exec.Command("cp", "v2ray/shadowsocks_users.json", "/etc/v2ray_users/shadowsocks_users.json")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to copy the shadowsocks users default file for the shadowsocks service: %s, %v", string(output), err)
	}

	cmd = exec.Command("sudo", "systemctl", "restart", "shadowsocks.service")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to start the shadowsocks service: %s, %v", string(output), err)
	}
	fmt.Println("shadowsocks panel successfully installed.")
	return nil
}

func sstpInstall() error {
	const serviceFile string = `[Unit]
		Description=SoftEther VPN server
		After=network-online.target
		After=dbus.service

		[Service]
		Type=forking
		ExecStart=/opt/softether/vpnserver start
		ExecReload=/bin/kill -HUP $MAINPID

		[Install]
		WantedBy=multi-user.target`

	cmd := exec.Command("sudo", "apt", "install", "unzip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to restart service: %s, %v", string(output), err)
	}

	cmd = exec.Command("unzip", "softether.zip")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to unzip softether service zip file: %s, %v", string(output), err)
	}

	cmd = exec.Command("mkdir", "-p", "/opt/softether/")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to move softether service files to opt directory: %s, %v", string(output), err)
	}

	cmd = exec.Command("mv", "softether/", "/opt/softether/")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to move softether service files to opt directory: %s, %v", string(output), err)
	}

	cmd = exec.Command("/opt/softether/vpncmd", "127.0.0.1:5555", "/server", "/password:htetmyatthar", "<<EOF", "\nservercertset", "\n/etc/letsencrypt/live/"+*WebHost+"/fullchain.pem", "\n/etc/letsencrypt/live/"+*WebHost+"/privkey.pem", "\nEOF")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to start the softether service: %s, %v", string(output), err)
	}

	cmd = exec.Command("sudo systemctl", "restart", "softether-vpnserver.service")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to start the softether service: %s, %v", string(output), err)
	}
	fmt.Println("sstp panel successfully installed.")
	return nil
}
