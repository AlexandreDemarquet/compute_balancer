package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"worker/cmd/handler"
)

var workerHome string

func init() {
	// Initialisation de la variable globale MASTER_HOME
	workerHome = os.Getenv("WORKER_HOME")
	if workerHome == "" {
		fmt.Println("WORKER_HOME non défini!!")
	} else {
		fmt.Println("WORKER_HOME défini :", workerHome)
	}
}

func main() {

	//workerHome := os.Getenv("WORKER_HOME")
	masterAddr := "localhost:8080" // a load depuis yaml

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
			var cmd handler.Command
			if err := decoder.Decode(&cmd); err != nil {
				handler.ReportProgress(conn, "Erreur dans le décodage de la commande")
				return
			}
			fmt.Println("Commande reçu du master : ", cmd.Command)
			handler.HandleCommand(conn, cmd, workerHome)
			//reportStatus(conn, workerAddr) // Envoi de l'état du worker après l'exécution de la commande
		}(conn)
	}
}
