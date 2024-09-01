package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type Command struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func sendCommandToWorker(workerAddr string, cmd Command) error {
	conn, err := net.Dial("tcp", workerAddr)
	if err != nil {
		return fmt.Errorf("error connecting to worker: %v", err)
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	return encoder.Encode(cmd)
}

func handleClientConnection(conn net.Conn, workerAddr string) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	arg, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading argument:", err)
		return
	}
	arg = arg[:len(arg)-1]

	cmd := Command{
		Command: "run_python",
		Args:    []string{"~/compute_balancer/worker/test_scrypt.py", arg},
	}

	err = sendCommandToWorker(workerAddr, cmd)
	if err != nil {
		fmt.Println("Error sending command to worker:", err)
	} else {
		fmt.Println("Command sent to worker with argument:", arg)
	}
}

func main() {
	listenAddr := ":8081"
	workerAddr := "localhost:8080" 

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		os.Exit(1)
	}
	defer ln.Close()

	fmt.Println("Master listening on", listenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleClientConnection(conn, workerAddr)
	}
}

