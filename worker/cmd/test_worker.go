package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"os"

	"gopkg.in/yaml.v3"

	"worker/cmd/handler"
)

var workerHome string
var masterAddr string
var logFile *os.File

// Configuration structure
type Config struct {
	MasterIP string `yaml:"master_ip"`
}

func getConfig() Config {
	var config Config
	yamlFile, err := os.ReadFile(workerHome + "/config/config.yaml")
	if err != nil {
		log.Fatalf("Erreur dans la lecture du .yaml: %v", err)
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("Erreur dans le décodage du .yaml: %v", err)
	}
	return config
}

func init() {
	// Ouvre le fichier de log dès l'initialisation
	var err error
	logFile, err = os.OpenFile("/var/log/worker_cb.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Erreur lors de l'ouverture du fichier de log : %v", err)
	}

	// Redirige les logs à la fois vers le fichier et la sortie standard
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	// Initialisation de la variable globale WORKER_HOME
	workerHome = os.Getenv("WORKER_HOME")
	if workerHome == "" {
		log.Fatalf("WORKER_HOME non défini!!")
	} else {
		log.Println("WORKER_HOME défini :", workerHome)
	}

	// Configuration adresse worker
	config := getConfig()
	masterAddr = config.MasterIP
}

func main() {
	defer logFile.Close() // on s'ssaure que le fichier de log se ferme bien à la fin du prog

	ln, err := net.Listen("tcp", masterAddr)
	if err != nil {
		log.Fatalf("Erreur lors de la création d'un listener: %v", err)
	}
	log.Println("Worker ecoute sur le port 8080")
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Erreur pour accepter la connexion:", err)
			continue
		}

		go func(conn net.Conn) {
			defer conn.Close()
			decoder := json.NewDecoder(conn)
			var cmd handler.Command
			if err := decoder.Decode(&cmd); err != nil {
				handler.ReportProgress(conn, "Erreur dans le décodage de la commande")
				log.Println("Erreur dans le décodage de la commande")
				return
			}
			log.Println("Commande reçu du master : ", cmd.Command)
			handler.HandleCommand(conn, cmd, workerHome)
		}(conn)
	}
}
