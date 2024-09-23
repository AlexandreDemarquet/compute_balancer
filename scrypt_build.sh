cd ./worker
sudo -E /usr/local/go/bin/go build -o /usr/local/bin/worker_test ./cmd/test_worker.go
cd ..
cd ./master
sudo -E /usr/local/go/bin/go build -o /usr/local/bin/master_test ./cmd/test_master.go
