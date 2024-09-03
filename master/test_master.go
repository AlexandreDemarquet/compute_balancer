package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
)

type Command struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type WorkerInfo struct {
	Address     string    `json:"address"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	Commands    []Command `json:"commands"`
}

var (
	workersInfo = make(map[string]*WorkerInfo)
	mutex       sync.Mutex
)

func sendCommandToWorker(workerAddr string, cmd Command) error {
	conn, err := net.Dial("tcp", workerAddr)
	if err != nil {
		return fmt.Errorf("error connecting to worker: %v", err)
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		return err
	}

	// Lire les réponses du worker jusqu'à la fermeture de la connexion
	reader := bufio.NewReader(conn)
	for {
		status, err := reader.ReadString('\n')
		if err != nil {
			break // Sortir de la boucle si la connexion est fermée
		}
		fmt.Println("Worker Status:", status)
	}

	return nil
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
		Args:    []string{"/home/n7student/compute_balancer/worker/test_scrypt.py", arg},
	}

	err = sendCommandToWorker(workerAddr, cmd)
	if err != nil {
		fmt.Println("Error sending command to worker:", err)
	} else {
		fmt.Println("Command sent to worker with argument:", arg)
	}
}

func workerInfoHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workersInfo)
}

func startHTTPServer() {
	// Serve static files from the "static" directory
	fs := http.FileServer(http.Dir("/home/n7student/compute_balancer/master/static"))
	http.Handle("/", fs)

	// Endpoint to get worker information
	http.HandleFunc("/workers", workerInfoHandler)

	fmt.Println("HTTP server running on :8082")
	http.ListenAndServe(":8082", nil)
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

	go startHTTPServer() // Démarrer le serveur HTTP dans une goroutine

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
