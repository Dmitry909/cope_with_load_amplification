```
sudo -u postgres psql
CREATE DATABASE db1;
CREATE USER dmitry WITH PASSWORD 'pwd';
GRANT ALL PRIVILEGES ON DATABASE db1 TO dmitry;
GRANT ALL ON DATABASE db1 TO dmitry;
ALTER DATABASE db1 OWNER TO dmitry;
\q
psql -U dmitry -d db1

CREATE TABLE pastes (
    short_id BIGINT NOT NULL PRIMARY KEY,
    target_link TEXT NOT NULL
);
```
