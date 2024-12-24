# Решение задачи "справиться с амплификацией нагрузки на БД для look-aside кэша"

**Идея**: ограничить количество одновременных запросов к БД с помощью семафора (в случае Go -- канала).
Все запросы к БД занесены в критические секции семафора, что позволяет не увеличивать нагрузку на нее даже в случае уменьшения cache hit-а.

Предусмотрены неуспешные ответы на запросы пользователей в случае превышения лимита одновременных запросов к БД. Это делается с помощью неблокующей записи в канал. Убедиться в этом можно установив `maxConcurrentDBQueries=1` и выполнив много запросов к серверу одновременно (например, из файла [high_load_requests.sh](high_load_requests.sh)). В БД вставится не 600 строк, а меньше, на невставленные будет возвращен код ответа 500.

[Как поднять БД](db.md)

Запуск сервера:
```
go run app/main.go
```

Примеры запросов на сервер:
```
curl -X POST -d "target_link=https://example.com" http://localhost:8080/write
curl -X GET "http://localhost:8080/read?short_id=1"
```
