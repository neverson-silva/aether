ALTER TABLE databases
    ADD COLUMN internal_port INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN public_access BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN external_port INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN data_volume TEXT NOT NULL DEFAULT '',
    ADD COLUMN data_volume_target TEXT NOT NULL DEFAULT '';

UPDATE databases
SET internal_port = CASE engine
        WHEN 'postgres' THEN 5432
        WHEN 'mysql' THEN 3306
        WHEN 'mariadb' THEN 3306
        WHEN 'redis' THEN 6379
        WHEN 'mongodb' THEN 27017
        WHEN 'mssql' THEN 1433
        WHEN 'oracle' THEN 1521
        ELSE port
    END,
    public_access = TRUE,
    external_port = CASE
        WHEN port BETWEEN 1024 AND 65535 THEN port
        WHEN engine = 'postgres' THEN 5432
        WHEN engine IN ('mysql', 'mariadb') THEN 3306
        WHEN engine = 'redis' THEN 6379
        WHEN engine = 'mongodb' THEN 27017
        WHEN engine = 'mssql' THEN 1433
        WHEN engine = 'oracle' THEN 1521
        ELSE port
    END
WHERE internal_port = 0;

ALTER TABLE databases
    ADD CONSTRAINT databases_internal_port_limit CHECK (internal_port BETWEEN 1 AND 65535),
    ADD CONSTRAINT databases_external_port_limit CHECK (external_port BETWEEN 0 AND 65535);
