// +build freebsd linux darwin

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

var (
	sshPubKey       = flag.String("sshpubkey", "$HOME/.ssh/id_ecdsa.pub", "SSH public key to request signed")
	sshCert         = flag.String("sshcert", "$HOME/.ssh/id_ecdsa-cert.pub", "SSH certificate to write")
	sshKnownHosts   = flag.String("sshknownhosts", "$HOME/.ssh/known_hosts", "SSH known hosts file to use")
	vaultTokenPath  = flag.String("vault_token", "$HOME/.vault-token", "Path to Vault token to update")
	vmwareCertPath  = flag.String("vmware_cert_path", "$HOME/vmware-user.pfx", "Path to store VMware user certificate")
	browserCertPath = flag.String("browser_cert_path", "$HOME/browser-user.pfx", "Path to store Browswer user certificate")

	certAuthority = "@cert-authority *.event.dreamhack.se ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjEAAAAIbmlzdHA1MjEAAACFBAC/xT7a8A4Gm1Tf0mpKstqncWsOZpGPKa0lqf7EuYSpWUnx5QLaiP2TcI80AELTw2gP9jzOkpN7/QO91V3edRXGLAGk3NiNZLqvJspYfAnEo9f3/E4GBZf4kcDC93+04SzbFg+qMY3iCmJNaIttUMdQwaR22c+HbOYhaGEFWN3OCa6Erw== vault@tech.dreamhack.se"
)

func sshLoadCertificate(c string) {
	cp := os.ExpandEnv(*sshCert)
	err := ioutil.WriteFile(cp, []byte(c), 0644)
	if err != nil {
		log.Printf("failed to write SSH certificate: %v", err)
	}

	// Add cert authority to known_hosts
	path := os.ExpandEnv(*sshKnownHosts)
	kh, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("failed to read SSH known hosts: %v", err)
	} else {
		if !strings.Contains(string(kh), certAuthority) {
			log.Printf("adding server identity to SSH known hosts")
			f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				log.Printf("failed to open SSH known hosts file for writing: %v", err)
			}
			defer f.Close()
			if _, err = f.WriteString(certAuthority); err != nil {
				log.Printf("failed to write to SSH known hosts file: %v", err)
			}
		} else {
			log.Printf("skipping SSH known hosts, already exists")
		}
	}

	// OpenSSH requires adding the private key again to load certificates
	pp := strings.TrimSuffix(cp, "-cert.pub")
	exec.Command("/usr/bin/env", "ssh-add", pp).Run()
}

func sshGetPublicKey() (string, error) {
	key, err := ioutil.ReadFile(os.ExpandEnv(*sshPubKey))
	if err != nil {
		log.Printf("could not read SSH public key: %v", err)
		return "", err
	}
	return string(key), nil
}

func saveVaultToken(t string) {
	tp := os.ExpandEnv(*vaultTokenPath)
	os.Remove(tp)
	err := ioutil.WriteFile(tp, []byte(t), 0400)
	if err != nil {
		log.Printf("failed to write Vault token: %v", err)
	}
}

func hasKubectl() bool {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		return false
	}
	return true
}

func saveKubernetesCertificate(c string, k string) {
	cf, _ := ioutil.TempFile("", "prodaccess-k8s")
	kf, _ := ioutil.TempFile("", "prodaccess-k8s")
	cf.Write([]byte(c))
	kf.Write([]byte(k))
	cp := cf.Name()
	kp := kf.Name()
	cf.Close()
	kf.Close()

	exec.Command("/usr/bin/env", "kubectl", "config", "set-credentials",
		"dhtech", "--embed-certs=true",
		fmt.Sprintf("--client-certificate=%s", cp),
		fmt.Sprintf("--client-key=%s", kp)).Run()
	os.Remove(cp)
	os.Remove(kp)
}

func saveVmwareCertificate(c string, k string) {
	cf, _ := ioutil.TempFile("", "prodaccess-vmware")
	kf, _ := ioutil.TempFile("", "prodaccess-vmware")
	cf.Write([]byte(c))
	kf.Write([]byte(k))
	cp := cf.Name()
	kp := kf.Name()
	cf.Close()
	kf.Close()

	fp := os.ExpandEnv(*vmwareCertPath)
	os.Remove(fp)
	os.OpenFile(fp, os.O_CREATE, 0600)
	cmd := exec.Command("/usr/bin/env", "openssl", "pkcs12", "-export", "-password", "pass:",
		"-in", cp, "-inkey", kp, "-out", fp)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("Failed to emit VMware certificate: %v", err)
		log.Printf("Standard output: %s", stdout.String())
		log.Printf("Error output: %s", stderr.String())
	}
	os.Remove(cp)
	os.Remove(kp)
}

func saveBrowserCertificate(c string, k string) {
	cf, _ := ioutil.TempFile("", "prodaccess-browser")
	kf, _ := ioutil.TempFile("", "prodaccess-browser")
	cf.Write([]byte(c))
	kf.Write([]byte(k))
	cp := cf.Name()
	kp := kf.Name()
	cf.Close()
	kf.Close()

	fp := os.ExpandEnv(*browserCertPath)
	os.Remove(fp)
	os.OpenFile(fp, os.O_CREATE, 0600)
	cmd := exec.Command("/usr/bin/env", "openssl", "pkcs12", "-export", "-password", "pass:",
		"-in", cp, "-inkey", kp, "-out", fp)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("Failed to emit Browser certificate: %v", err)
		log.Printf("Standard output: %s", stdout.String())
		log.Printf("Error output: %s", stderr.String())
	}
	os.Remove(cp)
	os.Remove(kp)
}
