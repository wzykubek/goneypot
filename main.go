package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"strings"
)

const (
	User          = "root"
	Password      = "toor"
	Hostname      = "vm"
	ListeningPort = "2222"
	Greeting      = "Last Login: Wed Dec 18 21:31:53 2024 from 192.168.1.1"
	ServerVersion = "SSH-2.0-OpenSSH_8.7"
)

func main() {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalln("Failed to generate private key:", err)
	}

	privateKeySigner, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		log.Fatalln("Failed to create signer:", err)
	}

	config := &ssh.ServerConfig{
		ServerVersion: ServerVersion,
		BannerCallback: func(conn ssh.ConnMetadata) string {
			return ""
		},
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if conn.User() == User && string(password) == Password {
				return nil, nil // Success
			}
			return nil, ssh.ErrNoAuth // Fail
		},
		AuthLogCallback: func(conn ssh.ConnMetadata, method string, err error) {
			return
		},
	}

	config.AddHostKey(privateKeySigner)

	listener, err := net.Listen("tcp", "0.0.0.0"+":"+ListeningPort)
	if err != nil {
		log.Fatalln("Failed to listen on port "+ListeningPort+":", err)
	}
	defer listener.Close()

	log.Printf("SSH server listening on port %s\n", ListeningPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Failed to accept connection:", err)
			continue
		}

		go handleConnection(conn, config)
	}
}

func handleConnection(conn net.Conn, config *ssh.ServerConfig) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Println("Failed to establish SSH connection:", err)
		return
	}
	defer sshConn.Close()

	log.Printf("New SSH connection from %s\n", sshConn.RemoteAddr())

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Println("Failed to accept channel:", err)
			continue
		}

		go handleChannel(channel, requests)
	}
}

func handleChannel(ch ssh.Channel, requests <-chan *ssh.Request) {
	defer ch.Close()

	for req := range requests {
		if req.Type == "shell" {
			req.Reply(true, nil)

			ch.Write([]byte(Greeting + "\n"))
			ch.Write([]byte(handlePrompt("~")))

			buf := make([]byte, 1024)

		loop:
			for {
				n, err := ch.Read(buf)
				if err != nil {
					break
				}

				ch.Write([]byte(handlePrompt("~")))

				cmdArray := strings.Fields(string(buf[:n-1]))

				if len(cmdArray) == 0 {
					ch.Write([]byte(""))
				} else {
					switch cmdArray[0] {
					case "ls":
						dirContent := []string{".", "..", ".bash_history", ".bash_logout", ".bash_profile", ".bashrc", ".local", "www"}
						if len(cmdArray) > 1 {
							if cmdArray[1] == "-a" {
								writeCmdOutput(ch, strings.Join(dirContent, "  "))
							}
						} else {
							var withoutHidden []string
							for _, v := range dirContent {
								if v[0] != '.' {
									withoutHidden = append(withoutHidden, v)
								}
							}
							writeCmdOutput(ch, strings.Join(withoutHidden, "  "))
						}
					case "cd":
						ch.Write([]byte(""))
					case "exit":
						ch.Write([]byte("\nlogout\n"))
						break loop
					default:
						writeCmdOutput(ch, fmt.Sprintf("bash: %s: command not found", cmdArray[0]))
					}
				}
			}

			break
		} else {
			req.Reply(false, nil)
		}
	}
}

func handlePrompt(workDir string) string {
	var userChar string
	if User == "root" {
		userChar = "#"
	} else {
		userChar = "$"
	}

	return fmt.Sprintf("[%s@%s %s]%s ", User, Hostname, workDir, userChar)
}

func writeCmdOutput(ch ssh.Channel, output string) {
	ch.Write([]byte("\n" + output + "\n"))
	ch.Write([]byte(handlePrompt("~")))
}
