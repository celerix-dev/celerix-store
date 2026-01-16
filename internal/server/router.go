package server

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/celerix-dev/celerix-store/pkg/sdk"
)

type Router struct {
	store sdk.CelerixStore
	cert  *tls.Certificate
}

func NewRouter(s sdk.CelerixStore) *Router {
	return &Router{store: s}
}

// SetCertificate sets the TLS certificate for the router
func (r *Router) SetCertificate(cert tls.Certificate) {
	r.cert = &cert
}

// Listen starts the TCP server
func (r *Router) Listen(port string) error {
	var listener net.Listener
	var err error

	if r.cert != nil {
		config := &tls.Config{Certificates: []tls.Certificate{*r.cert}}
		listener, err = tls.Listen("tcp", ":"+port, config)
	} else {
		listener, err = net.Listen("tcp", ":"+port)
	}
	if err != nil {
		return err
	}
	defer listener.Close()

	semaphore := make(chan struct{}, 100) // Max 100 concurrent connections

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		// Set aggressive timeouts for light traffic to prevent resource exhaustion
		conn.SetDeadline(time.Now().Add(5 * time.Minute))

		go func(c net.Conn) {
			semaphore <- struct{}{}
			defer func() {
				<-semaphore
				c.Close()
			}()
			r.handleConnection(c)
		}(conn)
	}
}

func (r *Router) handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)

	for {
		// Set a deadline for the next command
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		line, err := reader.ReadString('\n')
		if err != nil {
			return // Connection closed or timeout
		}

		line = strings.TrimSpace(line)
		parts := strings.Fields(line)
		if len(parts) < 1 {
			continue
		}

		command := strings.ToUpper(parts[0])

		switch command {
		case "GET":
			if len(parts) < 4 {
				continue
			}
			val, err := r.store.Get(parts[1], parts[2], parts[3])
			if err != nil {
				fmt.Fprintln(conn, "ERR", err)
			} else {
				// Send back as JSON
				res, err := json.Marshal(val)
				if err != nil {
					fmt.Fprintln(conn, "ERR internal error")
				} else {
					fmt.Fprintln(conn, "OK", string(res))
				}
			}

		case "SET":
			if len(parts) < 5 {
				continue
			}
			// The value is everything after the 4th word
			valueStr := strings.Join(parts[4:], " ")
			var val any
			if err := json.Unmarshal([]byte(valueStr), &val); err != nil {
				fmt.Fprintln(conn, "ERR invalid json value")
				continue
			}

			err := r.store.Set(parts[1], parts[2], parts[3], val)
			if err != nil {
				fmt.Fprintln(conn, "ERR", err)
			} else {
				fmt.Fprintln(conn, "OK")
			}

		case "DEL":
			if len(parts) < 4 {
				continue
			}
			err := r.store.Delete(parts[1], parts[2], parts[3])
			if err != nil {
				fmt.Fprintln(conn, "ERR", err)
			} else {
				fmt.Fprintln(conn, "OK")
			}

		case "LIST_PERSONAS":
			list, err := r.store.GetPersonas()
			if err != nil {
				fmt.Fprintln(conn, "ERR", err)
			} else {
				res, err := json.Marshal(list)
				if err != nil {
					fmt.Fprintln(conn, "ERR internal error")
				} else {
					fmt.Fprintln(conn, "OK", string(res))
				}
			}

		case "LIST_APPS":
			if len(parts) < 2 {
				continue
			}
			list, err := r.store.GetApps(parts[1])
			if err != nil {
				fmt.Fprintln(conn, "ERR", err)
			} else {
				res, err := json.Marshal(list)
				if err != nil {
					fmt.Fprintln(conn, "ERR internal error")
				} else {
					fmt.Fprintln(conn, "OK", string(res))
				}
			}

		case "DUMP":
			if len(parts) < 3 {
				continue
			}
			data, err := r.store.GetAppStore(parts[1], parts[2])
			if err != nil {
				fmt.Fprintln(conn, "ERR", err)
			} else {
				res, err := json.Marshal(data)
				if err != nil {
					fmt.Fprintln(conn, "ERR internal error")
				} else {
					fmt.Fprintln(conn, "OK", string(res))
				}
			}

		case "DUMP_APP":
			if len(parts) < 2 {
				continue
			}
			data, err := r.store.DumpApp(parts[1])
			if err != nil {
				fmt.Fprintln(conn, "ERR", err)
			} else {
				res, err := json.Marshal(data)
				if err != nil {
					fmt.Fprintln(conn, "ERR internal error")
				} else {
					fmt.Fprintln(conn, "OK", string(res))
				}
			}

		case "GET_GLOBAL":
			if len(parts) < 3 {
				continue
			}
			val, personaID, err := r.store.GetGlobal(parts[1], parts[2])
			if err != nil {
				fmt.Fprintln(conn, "ERR", err)
			} else {
				// We return a small JSON object with both value and persona
				out := map[string]any{
					"persona": personaID,
					"value":   val,
				}
				final, err := json.Marshal(out)
				if err != nil {
					fmt.Fprintln(conn, "ERR internal error")
				} else {
					fmt.Fprintln(conn, "OK", string(final))
				}
			}

		case "MOVE":
			if len(parts) < 5 {
				continue
			}
			// MOVE src dst app key
			err := r.store.Move(parts[1], parts[2], parts[3], parts[4])
			if err != nil {
				fmt.Fprintln(conn, "ERR", err)
			} else {
				fmt.Fprintln(conn, "OK")
			}

		case "PING":
			fmt.Fprintln(conn, "PONG")

		case "QUIT":
			return
		}
	}
}
