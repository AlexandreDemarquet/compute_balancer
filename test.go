package main

import (
	"fmt"
	"os"
)

var masterHome string

func main() {
	masterHome, booel := os.LookupEnv("MASTER_HOME")
	fmt.Println("MASTER_HOME :", masterHome)
	fmt.Println("MASTER_HOME :", booel)

}
