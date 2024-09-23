package informationmachine

import (
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"time"
)

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
func GetCPUUsage() (map[string]float64, error) {
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
func GetRAMUsage() (float64, error) {
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
