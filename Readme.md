DB:
```bash
docker run --name gochat -e POSTGRES_USER=chat -e POSTGRES_PASSWORD=secret -e POSTGRES_DB=chatdb -p 5432:5432 -d postgres
```

Load Schema:
```bash
docker cp schema.sql gochat:/schema.sql
docker exec -it gochat psql -U chat -d chatdb -f /schema.sql
```