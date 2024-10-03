package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

var logFile *os.File
var configWorkersIP []string

type Config struct {
	WorkersIP []string `yaml:"workers_ip"`
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

type WorkerEnVie struct {
	EtatWorker           string `json:"etat"`
	CommandeNonExectutee string `json:"commande_non_executee"`
}

var (
	workersInfo = make(map[string]*WorkerInfo)
	mutex       sync.Mutex
	masterHome  string
)

func init() {
	// Ouvre le fichier de log dès l'initialisation
	var err error
	logFile, err = os.OpenFile("/var/log/masterc_cb.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Erreur lors de l'ouverture du fichier de log : %v", err)
	}

	// Redirige les logs à la fois vers le fichier et la sortie standard
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	// Initialisation de la variable globale MASTER_HOME
	masterHome = os.Getenv("MASTER_HOME")
	if masterHome == "" {
		log.Fatalf("MASTER_HOME non défini!!")
	} else {
		log.Println("MASTER_HOME défini :", masterHome)
	}

	// Lecture de la config
	config := getConfig()
	configWorkersIP = config.WorkersIP
}

func getConfig() Config {
	var config Config
	yamlFile, err := os.ReadFile(masterHome + "/config/config.yaml")
	if err != nil {
		log.Println("Erreur dans la lecture du .yaml:", err)
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Println("Erreur dans le décodage du .yaml:", err)
	}
	return config
}

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
		conn, err := net.Dial("tcp", workerAddr)
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
		nouvIpDispos = append(nouvIpDispos, workerAddr)
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
		conn, err := net.Dial("tcp", workerAddr)
		if err != nil {
			log.Println("Connection impossible au worker: ", workerAddr)
			continue
		}
		defer conn.Close()

		cmd_first_connection := Command{
			Command: "infos",
			Args:    []string{"cpu_usage", "memory_usage", "nom_worker", "date", "disponible"}, //argument se sert actuellement a rien
		}

		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(cmd_first_connection); err != nil {
			log.Println("Envoie commande de première connection impossible au worker: ", workerAddr, "avec erreur: ", err)
			continue
		}

		decoder := json.NewDecoder(conn)
		var info WorkerInfo
		if err := decoder.Decode(&info); err != nil {
			log.Println("error decoding worker status:", err)
			continue
		}
		IPdispos = append(IPdispos, workerAddr)
		log.Println("Worker disponible:", info.Address)
		updateWorkerInfo(workerAddr, info)
	}

	return IPdispos
}

func handleClientConnection(conn net.Conn, workerAddr string) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	arg, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Error reading argument:", err)
		return
	}
	arg = arg[:len(arg)-1]

	cmd := Command{
		Command: "run_python",
		Args:    []string{arg},
	}

	err = sendCommandToWorker(workerAddr, cmd)
	if err != nil {
		log.Println("Error sending command to worker:", err)
	} else {
		log.Println("Command sent to worker with argument:", arg)
	}
}

// function qui envoie une commande python à workerAddr (ip:port) avec l'argument arg
func envoiCommandePython(workerAddr string, arg string) {
	cmd := Command{
		Command: "run_python",
		Args:    []string{arg}, //le chemin du scrypt python ne sera pas à donner
	}
	err := sendCommandToWorker(workerAddr, cmd)
	if err != nil {
		log.Println("Erreur envoie commande au worker ", workerAddr, ": ", err)
	} else {
		log.Println("Commande envoyée avec l'argument:", arg)
	}
}

func updateWorkerInfo(workerAddr string, info WorkerInfo) {
	mutex.Lock()
	defer mutex.Unlock()
	workersInfo[workerAddr] = &info
	log.Printf("Worker info updated: %+v\n", info)
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
	fs := http.FileServer(http.Dir(masterHome + "/static"))
	http.Handle("/", fs)

	// Endpoint to get worker information
	http.HandleFunc("/workers", workerInfoHandler)

	log.Println("HTTP server running on :8082")
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

func testRepriseContact(config_ip []string, worker_actuel []string) {
	missing := findMissing(config_ip, worker_actuel)
	for i := 0; i < len(missing); i++ {
		workerAddr := missing[i]
		conn, err := net.Dial("tcp", workerAddr)
		if err != nil {
			log.Println("Tentative de reconnection impossible au worker: ", workerAddr)
			continue
		}
		defer conn.Close()

		cmd_first_connection := Command{
			Command: "vivantoupas",
			Args:    []string{"est-ce que t'es vivant"}, //argument se sert actuellement a rien
		}

		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(cmd_first_connection); err != nil {
			log.Println("Envoie commande de reconnection impossible au worker: ", workerAddr, "avec erreur: ", err)
			continue
		}

		decoder := json.NewDecoder(conn)
		var retour WorkerEnVie
		if err := decoder.Decode(&retour); err != nil {
			log.Println("Erreur décodage retour de commande pour reconnection", err)
			continue
		}
		log.Println("Retour commande de reconnection du worker:", workerAddr, " ->")
	}

}

func main() {
	defer logFile.Close() // on s'ssaure que le fichier de log se ferme bien à la fin du prog

	var WorkersDispos []string
	// pour chaque worker renseigné on essaye de se connecter à lui et de récupérer ses informations
	WorkersDispos = firstConnectionToWorker(configWorkersIP, WorkersDispos)
	log.Println("Worker dispo :", configWorkersIP)

	go startHTTPServer() // Démarrer le serveur HTTP dans une goroutine

	dossier := masterHome + "/data"
	fichiersPrecedents, err := lireFichiers(dossier)
	if err != nil {
		log.Println("Erreur de lecture du dossier:", err)
	}
	for {
		WorkersDispos = recupInfosWorkers(WorkersDispos)

		fichiersActuels, err := lireFichiers(dossier)
		if err != nil {
			log.Println("Erreur de lecture du dossier:", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Chercher les nouveaux fichiers en comparant avec les précédents
		for fichier := range fichiersActuels {
			if _, existaitDeja := fichiersPrecedents[fichier]; !existaitDeja {
				log.Printf("Nouveau fichier détecté: %s\n", fichier)
				// ICI faut faire le choix du worker auquel on envoie les boulot.......
				log.Println("Worker choisi pour le boulot: ", "localhost pour le test")
				go envoiCommandePython("localhost:8080", fichier) // utilisation d'un go routine pour envoyer la commande python
			}
		}

		// Mettre à jour l'état précédent avec l'état actuel
		fichiersPrecedents = fichiersActuels

		// Pause avant la prochaine vérification (ex : 2 secondes)
		time.Sleep(2 * time.Second)

		go testRepriseContact(configWorkersIP, WorkersDispos)
	}
}
