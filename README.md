utiliser ce dépot:
- modifier fichiers go pour adapter les chemins ?
- lancer scrypt de buildœ
- modifier et copier coller les fichœier master_test.service et worker_test.service dans /etc/systemd/system/
- systemctl daemon-reload
- lancer scrypt de lancement daemon



A faire : 
- prendre en compte informations fichier yaml (port, max cpu, memory ect ect)
- faire un choix de distribution de calcul
- décomposition d'un fichier lidar au niveau d'un worker si fichier trop gros
- gérer l'historique des commandes
- gérer l'avancement du calcul
- envoie commande worker OK
- paralléliser avec des go routines la récupération des infos workers
- rendre les chemins universels OK
- faire une belle interface html
- modifier architecture projet OK   ( ou mettre les binaires? /usr/local/bin? ou dans la ./master/bin?)
- Faire des logs à la place des print OK
- resoudre bug ymal worker OK
- factoriser code master
