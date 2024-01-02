# faker

## Description

faker is a tool to generate fake data for testing purposes.

## Usage

### Build

```bash
go build ./...
```

### Run

```bash
go run ./...
```

### Test

```bash
go test -v --cover --race ./...
```

## Apendix

### MySQL

#### Show database table sizes in GB and fragmented space in GB

```sql
SELECT
 table_schema AS `Database`,
 table_name AS `Table`,
 round(((data_length + index_length) / 1024 / 1024 / 1024), 4) 'Size in GB',
 round(((data_free) / 1024 / 1024 / 1024), 4) 'Fragmented Space in GB'
 FROM
 information_schema.TABLES
 WHERE
 table_schema NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys');
```
