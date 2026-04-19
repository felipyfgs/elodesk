## ADDED Requirements

### Requirement: Upload inbound via multipart stream

Quando `POST /api/v1/accounts/:aid/conversations/:cid/messages` chega com `multipart/form-data`, cada arquivo em `attachments[]` SHALL ser streamado (sem buffer em memória) para o MinIO do backend em `{accountId}/{inboxId}/{messageId}.{ext}`. Uma `Attachment` row é criada com:

- `message_id`
- `file_type` (inferido do MIME: `image`/`video`/`audio`/`file`)
- `file_key` (chave no MinIO)
- `mime_type`
- `file_size`
- `external_url` nullable (URL presigned temporária se o provider mandou)

Limite hard: 256 MB por arquivo.

#### Scenario: upload de imagem

- **WHEN** provider faz POST multipart com `attachments[]=<jpg 2MB>`
- **THEN** MinIO tem objeto em `{aid}/{iid}/{mid}.jpg`, `attachments` row criada com `file_type=image`

#### Scenario: arquivo acima do limite

- **WHEN** upload com tamanho > 256 MB
- **THEN** retorna 413 sem persistir nada

### Requirement: Presigned URLs para download e upload direto

O backend SHALL expor:

- `POST /api/v1/accounts/:aid/uploads/signed-url` (JWT) com `{filename, content_type}` → retorna `{url, key, expires_in_sec}` para PUT direto no MinIO (TTL 15 min).
- `GET /api/v1/accounts/:aid/attachments/:id/signed-url` (JWT) → retorna URL presignada GET temporária (TTL 15 min) para download.

#### Scenario: presigned PUT

- **WHEN** frontend faz `POST /uploads/signed-url {filename:"foto.jpg",content_type:"image/jpeg"}`
- **THEN** retorna `{url, key, expires_in_sec:900}` válido por 15 min

#### Scenario: presigned GET expira

- **WHEN** URL de download passa de 15 min sem uso
- **THEN** tentativa de GET retorna 403 do MinIO

### Requirement: Isolamento de bucket por backend

O backend SHALL usar um bucket exclusivo (`wzap-media` por default, configurável via `MINIO_BUCKET`). Mídia recebida do provider é sempre re-uploadada pro bucket próprio (nunca linkada ao bucket do provider).

#### Scenario: bucket isolado

- **WHEN** provider manda attachment com `external_url` apontando pro storage dele
- **THEN** backend baixa do external_url e re-uploada no bucket próprio
- **AND** `Attachment.file_key` aponta pro bucket do backend, não pro do provider

### Requirement: Bucket auto-provisionado

No startup, se o bucket `MINIO_BUCKET` não existe, o backend SHALL criá-lo automaticamente. Falha no init é logada como warn e NÃO bloqueia o startup.

#### Scenario: primeira execução

- **WHEN** backend sobe pela primeira vez contra um MinIO vazio
- **THEN** bucket é criado e logs mostram `bucket created`

#### Scenario: bucket já existe

- **WHEN** backend sobe e bucket já está lá
- **THEN** no-op, sem erro
