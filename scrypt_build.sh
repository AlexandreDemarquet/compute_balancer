go build -o ./worker/worker_test ./worker/test_worker.go
cd ./master
go build -o ./master_test ./test_master.go
