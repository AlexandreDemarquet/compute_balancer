#!/bin/bash

# Nom des services systemd à gérer
SERVICES=("master_test" "worker_test")

# Parcourir chaque service
for SERVICE in "${SERVICES[@]}"; do
    # Vérifier si le service est actif
    if systemctl is-active --quiet "$SERVICE"; then
        echo "$SERVICE est déjà en cours d'exécution, redémarrage..."
        sudo systemctl restart "$SERVICE"
    else
        echo "$SERVICE n'est pas en cours d'exécution, démarrage..."
        sudo systemctl start "$SERVICE"
    fi

    # Vérifier si le service a démarré avec succès
    if systemctl is-active --quiet "$SERVICE"; then
        echo "$SERVICE est maintenant actif."
    else
        echo "Erreur: Impossible de démarrer $SERVICE."
    fi
done

