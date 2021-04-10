package net

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	"time"
)

type Conn struct {
	net.Conn
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type SSHClient struct {
	Config *ssh.ClientConfig
	Host   string
}

func NewSSHClient(host string, user string, privateKeyPath string, privateKeyPassword string) (*SSHClient, error) {
	// read private key file
	pemBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("reading private key file failed: %w", err)
	}
	// create signer
	signer, err := signerFromPem(pemBytes, []byte(privateKeyPassword))
	if err != nil {
		return nil, err
	}
	// build SSH client config
	config := &ssh.ClientConfig{
		User:    user,
		Timeout: 2 * time.Second,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// use OpenSSH's known_hosts file if you care about host validation
			return nil
		},
	}

	client := &SSHClient{
		Config: config,
		Host:   host,
	}

	return client, nil
}

func (c *Conn) Read(b []byte) (int, error) {
	err := c.Conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Read(b)
}

func (c *Conn) Write(b []byte) (int, error) {
	err := c.Conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Write(b)
}

func (s *SSHClient) GetClientWithTimeout(timeout time.Duration) (*ssh.Client, error) {
	if timeout >= 0 {
		// open connection
		conn, err := net.DialTimeout("tcp", s.Host, s.Config.Timeout)
		if err != nil {
			return nil, fmt.Errorf("dial to %v(ssh) failed %w", s.Host, err)
		}

		timeoutConn := &Conn{
			Conn:         conn,
			ReadTimeout:  timeout,
			WriteTimeout: timeout,
		}

		c, chans, reqs, err := ssh.NewClientConn(timeoutConn, s.Host, s.Config)
		if err != nil {
			return nil, err
		}

		client := ssh.NewClient(c, chans, reqs)
		return client, nil
	} else {
		conn, err := ssh.Dial("tcp", s.Host, s.Config)
		if err != nil {
			return nil, fmt.Errorf("dial to %v(ssh) failed %w", s.Host, err)
		}

		return conn, nil
	}
}

func (s *SSHClient) Heartbeat(client *ssh.Client, stopChan <-chan bool) {

	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-t.C:
				_, _, err := client.Conn.SendRequest("keepalive@golang.org", true, nil)
				if err != nil {
					return
				}
			}
		}
	}()

}

// Opens a new SSH connection and runs the specified command
// Returns the combined output of stdout and stderr
func (s *SSHClient) RunCommand(cmd string, timeout time.Duration) (string, error) {

	conn, err := s.GetClientWithTimeout(timeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// open session
	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("create session for %v failed %w", s.Host, err)
	}
	defer session.Close()

	// run command and capture stdout/stderr
	output, err := session.CombinedOutput(cmd)

	return fmt.Sprintf("%s", output), err
}

func signerFromPem(pemBytes []byte, password []byte) (ssh.Signer, error) {

	// read pem block
	err := errors.New("pem decode failed, no key found")
	pemBlock, _ := pem.Decode(pemBytes)
	if pemBlock == nil {
		return nil, err
	}

	// handle encrypted key
	if x509.IsEncryptedPEMBlock(pemBlock) {
		// decrypt PEM
		pemBlock.Bytes, err = x509.DecryptPEMBlock(pemBlock, []byte(password))
		if err != nil {
			return nil, fmt.Errorf("decrypting PEM block failed %v", err)
		}

		// get RSA, EC or DSA key
		key, err := parsePemBlock(pemBlock)
		if err != nil {
			return nil, err
		}

		// generate signer instance from key
		signer, err := ssh.NewSignerFromKey(key)
		if err != nil {
			return nil, fmt.Errorf("creating signer from encrypted key failed %v", err)
		}

		return signer, nil
	} else {
		// generate signer instance from plain key
		signer, err := ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			return nil, fmt.Errorf("parsing plain private key failed %v", err)
		}

		return signer, nil
	}
}

func parsePemBlock(block *pem.Block) (interface{}, error) {
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing PKCS private key failed %w", err)
		} else {
			return key, nil
		}
	case "EC PRIVATE KEY":
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing EC private key failed %w", err)
		} else {
			return key, nil
		}
	case "DSA PRIVATE KEY":
		key, err := ssh.ParseDSAPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing DSA private key failed %w", err)
		} else {
			return key, nil
		}
	default:
		return nil, fmt.Errorf("parsing private key failed, unsupported key type %q", block.Type)
	}
}
