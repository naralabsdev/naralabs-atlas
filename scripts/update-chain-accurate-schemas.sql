-- Chain-accurate schema bodies (vec approve, fee topics, price vec, oracle data).
-- Apply on production Postgres, then run: ./scripts/replay-semantic-decode.sh <from> <to>

BEGIN;

UPDATE event_schemas
SET schema_body = '{"name":"approve","prefix_topics":["approve"],"data_format":"vec","params":[{"name":"from","type":"address","location":"topic"},{"name":"spender","type":"address","location":"topic"},{"name":"amount","type":"i128","location":"data","vec_index":0},{"name":"expiration_ledger","type":"u32","location":"data","vec_index":1}]}',
    updated_at = now()
WHERE network = 'testnet'
  AND event_name = 'approve'
  AND contract_id IN (
    'CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC',
    'CD4MP2QV4LABFU5OL6Y5RICCLQIX7TWT6T6J546EDYVUXHH6RS2V5S2O'
  );

UPDATE event_schemas
SET schema_body = '{"name":"fee","prefix_topics":["fee"],"params":[{"name":"from","type":"address","location":"topic"},{"name":"to","type":"address","location":"topic"},{"name":"amount","type":"i128","location":"data"}]}',
    updated_at = now()
WHERE network = 'testnet'
  AND event_name = 'fee'
  AND contract_id = 'CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC';

UPDATE event_schemas
SET schema_body = '{"name":"price","prefix_topics":["price"],"data_format":"vec","params":[{"name":"asset","type":"symbol","location":"topic"},{"name":"price","type":"i128","location":"data","vec_index":0},{"name":"timestamp","type":"u64","location":"data","vec_index":1}]}',
    updated_at = now()
WHERE network = 'testnet'
  AND event_name = 'price'
  AND contract_id = 'CB3IUO2Y5NFH7LDX5EOLA63WO7QSYJSOBMMZESLBC62DEPNQJPOYDULC';

UPDATE event_schemas
SET schema_body = '{"name":"oracle_updated","prefix_topics":["oracle_updated"],"data_format":"single_value","params":[{"name":"feed","type":"symbol","location":"topic"},{"name":"price","type":"i128","location":"data"}]}',
    updated_at = now()
WHERE network = 'testnet'
  AND event_name = 'oracle_updated'
  AND contract_id = 'CBTBKYT3OMI4LOUX3UGOZTQSGFAABHHNVRVVGI4SSX3WWNHAIGQAE6TC';

COMMIT;
