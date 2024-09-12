utiliser ce dépot:
- modifier fichiers go pour adapter les chemins ?
- lancer scrypt de buildœ
- copier coller les fichœier master_test.service et worker_test.service dans /etc/systemd/system/
- lancer scrypt de lancement daemon
- dans un autre terminal lancer commande "telnet localhost 8081"



A faire : 
- prendre en compte informations fichier yaml (port, max cpu, memory ect ect)
- faire un choix de distribution de calcul
- décomposition d'un fichier lidar au niveau d'un worker si fichier trop gros
- gérer l'historique des commandes
- gérer l'avancement du calcul
- envoie commande worker
- paralléliser avec des go routines la récupération des infos workers
- rendre les chemins universels
- faire une belle interface html