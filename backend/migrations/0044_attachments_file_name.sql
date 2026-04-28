-- Adiciona coluna `file_name` na tabela attachments para preservar o nome
-- original do arquivo enviado pelo usuário (antes da sanitização que faz parte
-- da chave do MinIO). UI usa esse campo pra mostrar o nome bonito (com
-- acentos, espaços, parênteses) tipo o WhatsApp.
ALTER TABLE attachments ADD COLUMN file_name TEXT;
