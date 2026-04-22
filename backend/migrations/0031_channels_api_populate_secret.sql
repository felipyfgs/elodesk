UPDATE channels_api
   SET secret = encode(gen_random_bytes(48), 'base64')
 WHERE secret IS NULL OR secret = '';
