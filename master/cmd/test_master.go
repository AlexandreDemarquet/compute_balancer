package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Configuration structure
type Config struct {
	WorkersIP []string `yaml:"workers_ip"`
}

// Log commands to file
func logCommand(command string) {
	f, err := os.OpenFile("commands.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error logging command:", err)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(fmt.Sprintf("%s: %s\n", time.Now().Format(time.RFC3339), command)); err != nil {
		fmt.Println("Error writing to log file:", err)
	}
}

type Command struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type WorkerInfo struct {
	Address        string             `json:"address"`
	CPUUsage       map[string]float64 `json:"cpu_usage"` // Map pour chaque core
	MemoryUsage    float64            `json:"memory_usage"`
	Commands       []Command          `json:"commands"`
	Machine        string             `json:"nom_machine"`
	DateConnection string             `json:"date_connection"`
}

var (
	workersInfo = make(map[string]*WorkerInfo)
	mutex       sync.Mutex
)

func sendCommandToWorker(workerAddr string, cmd Command) error {
	conn, err := net.Dial("tcp", workerAddr)
	if err != nil {
		return fmt.Errorf("erreur de connection au worker pour envoie commande: %v", err)
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

func findMissing(list1, list2 []string) (missing []string) {
	set := make(map[string]struct{}, len(list2))
	for _, v := range list2 {
		set[v] = struct{}{}
	}
	for _, v := range list1 {
		if _, found := set[v]; !found {
			missing = append(missing, v)
		}
	}
	return
}

func recupInfosWorkers(workersAddr []string) []string {
	var nouvIpDispos []string
	// On boucle sur les adresses ip disponibles et on met à jour leurs états et on signale si un des workers est dead
	for i := 0; i < len(workersAddr); i++ {
		workerAddr := workersAddr[i]
		conn, err := net.Dial("tcp", workerAddr+":8080")
		if err != nil {
			fmt.Println("Connection impossible au worker: ", workerAddr)
			continue
		}
		defer conn.Close()

		cmd_first_connection := Command{
			Command: "infos",
			Args:    []string{"cpu_usage", "memory_usage", "nom_worker", "date", "disponible"}, //argument se sert actuellement a rien
		}

		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(cmd_first_connection); err != nil {
			fmt.Println("Envoie commande de première connection impossible au worker: ", workerAddr, "avec erreur: ", err)
			continue
		}

		decoder := json.NewDecoder(conn)
		var info WorkerInfo
		if err := decoder.Decode(&info); err != nil {
			fmt.Println("Erreur décodage infos du worker:", err)
			continue
		}
		nouvIpDispos = append(nouvIpDispos, info.Address)
		updateWorkerInfo(workerAddr, info)
	}
	missing := findMissing(nouvIpDispos, workersAddr)
	if len(missing) != 0 {
		fmt.Println("Perte de contact avec: ", missing)
	}
	return nouvIpDispos
}

func firstConnectionToWorker(workersAddr []string, IPdispos []string) []string {
	// On boucle sur les adresses ip présentent dans le yaml et on renvoie les adresses ip des workers disponibles.
	for i := 0; i < len(workersAddr); i++ {
		workerAddr := workersAddr[i]
		conn, err := net.Dial("tcp", workerAddr+":8080")
		if err != nil {
			fmt.Println("Connection impossible au worker: ", workerAddr)
			continue
		}
		defer conn.Close()

		cmd_first_connection := Command{
			Command: "infos",
			Args:    []string{"cpu_usage", "memory_usage", "nom_worker", "date", "disponible"}, //argument se sert actuellement a rien
		}

		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(cmd_first_connection); err != nil {
			fmt.Println("Envoie commande de première connection impossible au worker: ", workerAddr, "avec erreur: ", err)
			continue
		}

		decoder := json.NewDecoder(conn)
		var info WorkerInfo
		if err := decoder.Decode(&info); err != nil {
			fmt.Println("error decoding worker status:", err)
			continue
		}
		IPdispos = append(IPdispos, info.Address)
		fmt.Println("Worker disponible:", info.Address)
		updateWorkerInfo(workerAddr, info)

	}

	return IPdispos
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
		Args:    []string{"/home/n7student/compute_balancer/worker/test/test_scrypt.py", arg},
	}

	err = sendCommandToWorker(workerAddr, cmd)
	if err != nil {
		fmt.Println("Error sending command to worker:", err)
	} else {
		fmt.Println("Command sent to worker with argument:", arg)
	}
}

func envoiCommandePython(workerAddr string, arg string) {
	cmd := Command{
		Command: "run_python",
		Args:    []string{"/home/n7student/compute_balancer/worker/test/test_scrypt.py", arg}, //le chemin du scrypt python ne sera pas à donner
	}

	err := sendCommandToWorker(workerAddr, cmd)
	if err != nil {
		fmt.Println("erreur envoie commande au worker ", workerAddr, ": ", err)
	} else {
		fmt.Println("Commande envoyée avec l'argument:", arg)
	}
}

func updateWorkerInfo(workerAddr string, info WorkerInfo) {
	mutex.Lock()
	defer mutex.Unlock()

	workersInfo[workerAddr] = &info
	fmt.Printf("Worker info updated: %+v\n", info)
}

func receiveWorkerStatus(workerAddr string) error {
	conn, err := net.Dial("tcp", workerAddr)
	if err != nil {
		return fmt.Errorf("error connecting to worker: %v", err)
	}
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	var info WorkerInfo
	if err := decoder.Decode(&info); err != nil {
		return fmt.Errorf("error decoding worker status: %v", err)
	}

	updateWorkerInfo(workerAddr, info)
	return nil
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

func lireFichiers(dossier string) (map[string]os.FileInfo, error) {
	fichiers := make(map[string]os.FileInfo)
	listeFichiers, err := ioutil.ReadDir(dossier)
	if err != nil {
		return nil, err
	}
	for _, fichier := range listeFichiers {
		fichiers[fichier.Name()] = fichier
	}
	return fichiers, nil
}

func main() {

	// Lecture de la config
	var config Config

	yamlFile, err := os.ReadFile("/home/n7student/compute_balancer/master/config/config.yaml")
	if err != nil {
		fmt.Println("Erreur dans la lecture du .yaml:", err)
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		fmt.Println("Erreur dans le décodage du .yaml:", err)
	}

	var WorkersDispos []string
	// pour chaque worker renseigné on essaye de se connecter à lui et de récupérer ses informations
	WorkersDispos = firstConnectionToWorker(config.WorkersIP, WorkersDispos)

	//listenAddr := ":8081"
	//workerAddr := "localhost:8080"

	// Écoute sur le port 8081 pour lui envoyer l'argument de la commande (ça sera le nom de fichier .las que le serveur web lui enverra)
	//ln, err := net.Listen("tcp", listenAddr)
	//if err != nil {
	//	fmt.Println("Error starting TCP server on port 8081:", err)
	//	os.Exit(1)
	//}
	//defer ln.Close()

	go startHTTPServer() // Démarrer le serveur HTTP dans une goroutine

	//fmt.Println("Master listening on", listenAddr)

	dossier := "/home/n7student/compute_balancer/data"
	fichiersPrecedents, err := lireFichiers(dossier)
	if err != nil {
		fmt.Println("Erreur de lecture du dossier:", err)
	}
	for {
		WorkersDispos = recupInfosWorkers(WorkersDispos)

		fichiersActuels, err := lireFichiers(dossier)
		if err != nil {
			fmt.Println("Erreur de lecture du dossier:", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Chercher les nouveaux fichiers en comparant avec les précédents
		for fichier := range fichiersActuels {
			if _, existaitDeja := fichiersPrecedents[fichier]; !existaitDeja {
				fmt.Printf("Nouveau fichier détecté: %s\n", fichier)
				// ICI faut faire le choix du worker auquel on envoie les boulot.......
				fmt.Println("Worker choisi pour le boulot: ", "localhost pour le test")
				go envoiCommandePython("localhost:8080", fichier) // utilisation d'un go routine pour envoyer la commande python
			}
		}

		// Mettre à jour l'état précédent avec l'état actuel
		fichiersPrecedents = fichiersActuels

		// Pause avant la prochaine vérification (ex : 2 secondes)
		time.Sleep(2 * time.Second)

		//conn, err := ln.Accept()
		//if err != nil {
		//	fmt.Println("Error accepting connection:", err)
		//	continue
		//}

		// Gérer la connexion client dans une goroutine séparée
		//go handleClientConnection(conn, workerAddr)

		// Après avoir traité la commande, récupérer l'état du worker
		//err = receiveWorkerStatus(workerAddr)
		//if err != nil {
		//	fmt.Printf("Error receiving worker status: %v\n", err)
		//}
	}
}
