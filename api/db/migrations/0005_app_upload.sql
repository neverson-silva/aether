-- Suporte a deploy de serviço a partir de ZIP upload.
ALTER TABLE apps ADD COLUMN upload_id TEXT NOT NULL DEFAULT '';
