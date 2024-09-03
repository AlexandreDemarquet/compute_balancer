package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"runtime"
)

type Command struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type WorkerInfo struct {
	Address     string  `json:"address"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
}

func reportProgress(conn net.Conn, progress string) error {
	_, err := conn.Write([]byte(progress + "\n"))
	return err
}

func handleCommand(conn net.Conn, cmd Command) {
	switch cmd.Command {
	case "run_python":
		if len(cmd.Args) < 2 {
			reportProgress(conn, "Erreur: nombre d'arguments insuffisant")
			return
		}
		// Exécution du script Python
		script := cmd.Args[0]
		arg := cmd.Args[1]

		cmd := exec.Command("python3", script, arg)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			reportProgress(conn, fmt.Sprintf("Erreur1: %v", err))
			return
		}
		scanner := bufio.NewScanner(stdout)
		if err := cmd.Start(); err != nil {
			reportProgress(conn, fmt.Sprintf("Erreur2: %v", err))
			return
		}

		for scanner.Scan() {
			line := scanner.Text()
			reportProgress(conn, fmt.Sprintf("Output: %s", line))
		}
		if scanner.Err() != nil {
			cmd.Process.Kill()
			cmd.Wait()
			reportProgress(conn, fmt.Sprintf("Erreur_scann: %v", scanner.Err()))
		}
		if err := cmd.Wait(); err != nil {
			reportProgress(conn, fmt.Sprintf("Erreur3: %v", err))
			return
		}

		reportProgress(conn, "T'as réussi bg le script s'est exécuté!")
	default:
		reportProgress(conn, "Commande inconnue")
	}
}

func getCPUUsage() float64 {
	// Simulation de l'utilisation du CPU
	return float64(runtime.NumGoroutine()) * 10 // Remplacez par une vraie mesure si possible
}

func getMemoryUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / 1024 / 1024 // Convertir en Mo
}

func reportStatus(conn net.Conn, workerAddr string) {
	cpuUsage := getCPUUsage()
	memoryUsage := getMemoryUsage()

	workerStatus := WorkerInfo{
		Address:     workerAddr,
		CPUUsage:    cpuUsage,
		MemoryUsage: memoryUsage,
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(workerStatus); err != nil {
		fmt.Println("Erreur lors de l'envoi de l'état du worker:", err)
	}
}

func main() {
	workerAddr := "localhost:8080"

	ln, err := net.Listen("tcp", workerAddr)
	if err != nil {
		fmt.Println("Erreur lors de la création d'un listener:", err)
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Erreur pour accepter la connection:", err)
			continue
		}

		go func(conn net.Conn) {
			defer conn.Close()
			decoder := json.NewDecoder(conn)
			var cmd Command
			if err := decoder.Decode(&cmd); err != nil {
				reportProgress(conn, "Erreur dans le décodage de la commande")
				return
			}
			handleCommand(conn, cmd)
			reportStatus(conn, workerAddr) // Envoi de l'état du worker après l'exécution de la commande
		}(conn)
	}
}
