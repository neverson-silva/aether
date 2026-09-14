ALTER TABLE databases
    ADD CONSTRAINT databases_port_limit CHECK (port BETWEEN 0 AND 65535) NOT VALID;
