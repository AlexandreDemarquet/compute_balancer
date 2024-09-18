#!/bin/bash

# Nom des services systemd à gérer
SERVICES=("worker_test" "master_test")

# Parcourir chaque service
for SERVICE in "${SERVICES[@]}"; do
    # Vérifier si le service est actif
    if systemctl is-active --quiet "$SERVICE"; then
        echo "$SERVICE est déjà en cours d'exécution, redémarrage..."
        sudo -E systemctl restart "$SERVICE"
    else
        echo "$SERVICE n'est pas en cours d'exécution, démarrage..."
        sudo -E systemctl start "$SERVICE"
    fi

    # Vérifier si le service a démarré avec succès
    if systemctl is-active --quiet "$SERVICE"; then
        echo "$SERVICE est maintenant actif."
    else
        echo "Erreur: Impossible de démarrer $SERVICE."
    fi
done

