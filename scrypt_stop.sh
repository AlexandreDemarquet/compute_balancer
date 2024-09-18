#!/bin/bash

# Nom des services systemd à gérer
SERVICES=("worker_test" "master_test")

# Parcourir chaque service
for SERVICE in "${SERVICES[@]}"; do
    # Vérifier si le service est actif
    if systemctl is-active --quiet "$SERVICE"; then
        echo "$SERVICE est en cours d'exécution, arret..."
        sudo systemctl stop "$SERVICE"
    else
        echo "$SERVICE n'est pas en cours d'exécution..."
    fi

done

