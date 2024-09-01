package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
)

type Command struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
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
			reportProgress(conn, fmt.Sprintf("Erreur: %v", err))
			return
		}
		if err := cmd.Start(); err != nil {
			reportProgress(conn, fmt.Sprintf("Erreur: %v", err))
			return
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			reportProgress(conn, fmt.Sprintf("Output: %s", line))
		}

		if err := cmd.Wait(); err != nil {
			reportProgress(conn, fmt.Sprintf("Erreur: %v", err))
			return
		}
		reportProgress(conn, "T'as réussi bg le script s'est exécuté!")
	default:
		reportProgress(conn, "Commande inconnue")
	}
}

func main() {
	ln, err := net.Listen("tcp", ":8080")
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
		}(conn)
	}
}

