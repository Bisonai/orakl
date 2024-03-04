curl -X POST http://localhost:8088/api/v1/adapter/sync
sleep 10
curl -X POST http://localhost:8088/api/v1/fetcher/refresh