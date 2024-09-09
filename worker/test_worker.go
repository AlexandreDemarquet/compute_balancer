package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Command struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type WorkerInfo struct {
	Address        string             `json:"address"`
	CPUUsage       map[string]float64 `json:"cpu_usage"` // Changement: map pour chaque core
	MemoryUsage    float64            `json:"memory_usage"`
	Machine        string             `json:"nom_machine"`
	DateConnection string             `json:"date_connection"`
}

// reportProgress envoie la progression au client
func reportProgress(conn net.Conn, progress string) error {
	_, err := conn.Write([]byte(progress + "\n"))
	return err
}

// handleCommand gère les différentes commandes reçues
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

	case "infos":
		reportStatus(conn)
	default:
		reportProgress(conn, "Commande inconnue")
	}
}

// getCPUUsage retourne l'utilisation du CPU pour chaque cœur
func getCPUUsage_bis() (map[string]float64, error) {
	// Lire le contenu de /proc/stat
	data, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return nil, err
	}

	cpuUsage := make(map[string]float64)

	// Analyse ligne par ligne
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu") {
			fields := strings.Fields(line)

			// Evite la ligne globale "cpu " et sélectionne les cores individuels comme "cpu0"
			if len(fields[0]) > 3 {
				core := fields[0] // ex : cpu0, cpu1, etc.

				// Conversion des valeurs
				user, _ := strconv.ParseUint(fields[1], 10, 64)
				nice, _ := strconv.ParseUint(fields[2], 10, 64)
				system, _ := strconv.ParseUint(fields[3], 10, 64)
				idle, _ := strconv.ParseUint(fields[4], 10, 64)

				// Calcul de l'utilisation du CPU
				total := user + nice + system + idle
				usage := float64(user+nice+system) / float64(total) * 100

				// Ajout de l'utilisation pour ce core dans la map
				cpuUsage[core] = usage
			}
		}
	}
	return cpuUsage, nil
}

// Fonction pour lire l'utilisation du CPU
func getCPUUsage() (map[string]float64, error) {
	firstStat, err := lireStat()
	if err != nil {
		return nil, err
	}

	time.Sleep(100 * time.Millisecond)

	secondStat, err := lireStat()
	if err != nil {
		return nil, err
	}

	cpuUsage := make(map[string]float64)

	for core, firstValues := range firstStat {
		secondValues, exists := secondStat[core]
		if !exists {
			continue
		}

		totalFirst := total(firstValues)
		totalSecond := total(secondValues)

		idleFirst := firstValues[3]
		idleSecond := secondValues[3]

		totalDiff := totalSecond - totalFirst
		idleDiff := idleSecond - idleFirst

		usage := (1.0 - float64(idleDiff)/float64(totalDiff)) * 100
		cpuUsage[core] = usage
	}

	return cpuUsage, nil
}

// Fonction pour lire les stats du CPU depuis /proc/stat
func lireStat() (map[string][]uint64, error) {
	data, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return nil, err
	}

	cpuStats := make(map[string][]uint64)

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu") {
			fields := strings.Fields(line)

			if len(fields[0]) > 3 {
				core := fields[0]
				var values []uint64

				for _, field := range fields[1:8] {
					val, err := strconv.ParseUint(field, 10, 64)
					if err != nil {
						return nil, err
					}
					values = append(values, val)
				}
				cpuStats[core] = values
			}
		}
	}

	return cpuStats, nil
}

func total(values []uint64) uint64 {
	var total uint64
	for _, val := range values {
		total += val
	}
	return total
}

// Fonction pour lire l'utilisation de la RAM depuis /proc/meminfo
func getRAMUsage() (float64, error) {
	data, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(data), "\n")

	//var totalRAM, freeRAM, availableRAM uint64
	var totalRAM, availableRAM uint64

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := fields[0]
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, err
		}

		// Trouver les lignes de mémoire totale et disponible
		switch key {
		case "MemTotal:":
			totalRAM = value
		case "MemAvailable:":
			availableRAM = value
			//case "MemFree:":
			//	freeRAM = value
		}
	}

	// Calculer l'utilisation de la RAM en pourcentage
	usedRAM := totalRAM - availableRAM
	usage := float64(usedRAM) / float64(totalRAM) * 100

	return usage, nil
}

// getMemoryUsage retourne l'utilisation de la mémoire en Mo
func getMemoryUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / 1024 / 1024 // Convertir en Mo
}

// reportStatus envoie l'état du worker au client
func reportStatus(conn net.Conn) {
	// Récupération de l'utilisation CPU et mémoire
	cpuUsage, err := getCPUUsage()
	if err != nil {
		fmt.Println("Erreur lors de la récupération de l'utilisation CPU:", err)
		return
	}

	//memoryUsage := getMemoryUsage()
	memoryUsage, err := getRAMUsage()
	if err != nil {
		fmt.Println("Erreur lors de la récupération de l'utilisation RAM:", err)
		return
	}
	nomMachine, err := os.Hostname()
	if err != nil {
		fmt.Println("Erreur lors de la récupération le nom de la machine:", err)
		return
	}

	now := time.Now()
	myAddr := "localhost" //recupérer son adresse ip
	// Création de l'objet avec les informations du worker
	workerStatus := WorkerInfo{
		Address:        myAddr,
		CPUUsage:       cpuUsage, // Utilisation CPU par cœur
		MemoryUsage:    memoryUsage,
		Machine:        nomMachine,
		DateConnection: now.Format("2006-01-02 15:04:05"),
	}

	// Envoi de l'état au client
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(workerStatus); err != nil {
		fmt.Println("Erreur lors de l'envoi de l'état du worker:", err)
	}
}

func main() {
	masterAddr := "localhost:8080"

	ln, err := net.Listen("tcp", masterAddr)
	if err != nil {
		fmt.Println("Erreur lors de la création d'un listener:", err)
		return
	}
	fmt.Println("Worker ecoute sur le port 8080:")
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Erreur pour accepter la connexion:", err)
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
			//reportStatus(conn, workerAddr) // Envoi de l'état du worker après l'exécution de la commande
		}(conn)
	}
}
