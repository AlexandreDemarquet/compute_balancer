package handler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"
	"worker/cmd/informationmachine"
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

type WorkerEnVie struct {
	EtatWorker           string `json:"etat"`
	CommandeNonExectutee string `json:"commande_non_executee"`
}

// reportProgress envoie la progression au client
func ReportProgress(conn net.Conn, progress string) error {
	_, err := conn.Write([]byte(progress + "\n"))
	return err
}

// reportStatus envoie l'état du worker au client
func reportStatus(conn net.Conn) {
	// Récupération de l'utilisation CPU et mémoire
	cpuUsage, err := informationmachine.GetCPUUsage()
	if err != nil {
		log.Println("Erreur lors de la récupération de l'utilisation CPU:", err)
		return
	}

	//memoryUsage := getMemoryUsage()
	memoryUsage, err := informationmachine.GetRAMUsage()
	if err != nil {
		log.Println("Erreur lors de la récupération de l'utilisation RAM:", err)
		return
	}
	nomMachine, err := os.Hostname()
	if err != nil {
		log.Println("Erreur lors de la récupération le nom de la machine:", err)
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
		log.Println("Erreur lors de l'envoi de l'état du worker:", err)
	}
}

func handleRunPython(conn net.Conn, cmd_python Command, workerHome string) {
	if len(cmd_python.Args) < 1 {
		ReportProgress(conn, "Erreur: nombre d'arguments insuffisant")
		return
	}
	// Exécution du script Python
	//script := cmd_python.Args[0]
	script := workerHome + "/test/test_scrypt.py"
	arg := cmd_python.Args[0]

	cmd := exec.Command("python3", script, arg)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ReportProgress(conn, fmt.Sprintf("Erreur1: %v", err))
		return
	}
	scanner := bufio.NewScanner(stdout)
	if err := cmd.Start(); err != nil {
		ReportProgress(conn, fmt.Sprintf("Erreur2: %v", err))
		return
	}

	for scanner.Scan() {
		line := scanner.Text()
		ReportProgress(conn, fmt.Sprintf("Output: %s", line))
	}
	if scanner.Err() != nil {
		cmd.Process.Kill()
		cmd.Wait()
		ReportProgress(conn, fmt.Sprintf("Erreur_scann: %v", scanner.Err()))
	}
	if err := cmd.Wait(); err != nil {
		ReportProgress(conn, fmt.Sprintf("Erreur3: %v", err))
		return
	}

	ReportProgress(conn, "T'as réussi bg le script s'est exécuté!")
}

func handleVivantOuPas(conn net.Conn) {
	//test
	workerAlive := WorkerEnVie{
		EtatWorker:           "disponible",
		CommandeNonExectutee: "nada test",
	}

	// Envoi de l'état au client
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(workerAlive); err != nil {
		log.Println("Erreur lors de l'envoi de l'état du worker pour tentative de reconnection:", err)
	}
}

// handleCommand gère les différentes commandes reçues
func HandleCommand(conn net.Conn, cmd Command, workerHome string) {
	switch cmd.Command {
	case "run_python":
		handleRunPython(conn, cmd, workerHome)
	case "infos":
		reportStatus(conn)
	case "vivantoupas":
		handleVivantOuPas(conn)
	default:
		ReportProgress(conn, "Commande inconnue")
	}
}
