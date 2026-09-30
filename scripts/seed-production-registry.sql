-- Community schema catalog from live testnet indexer activity (2026-03-30).
-- publisher_user_id must exist in users table.

BEGIN;

DELETE FROM event_schemas
WHERE publisher_user_id = '13b21dc5-ff89-48b6-814e-e30b958a77fc'::uuid
  AND author LIKE 'NaraLabs community catalog%';

-- 1) High-volume USDC-style token (transfer / approve / fee)
INSERT INTO event_schemas (contract_id, network, event_name, version, schema_body, author, publisher_user_id, trust_tier, status) VALUES
('CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC', 'testnet', 'transfer', 1, '{"name":"transfer","prefix_topics":["transfer"],"params":[{"name":"from","type":"address","location":"topic"},{"name":"to","type":"address","location":"topic"},{"name":"asset","type":"string","location":"topic"},{"name":"amount","type":"i128","location":"data"}]}', 'NaraLabs community catalog (testnet USDC)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),
('CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC', 'testnet', 'approve', 1, '{"name":"approve","prefix_topics":["approve"],"params":[{"name":"from","type":"address","location":"topic"},{"name":"spender","type":"address","location":"topic"},{"name":"amount","type":"i128","location":"data"}]}', 'NaraLabs community catalog (testnet USDC)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),
('CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC', 'testnet', 'fee', 1, '{"name":"fee","prefix_topics":["fee"],"params":[{"name":"amount","type":"i128","location":"data"}]}', 'NaraLabs community catalog (testnet USDC)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),

-- 2) Stellar DEX orderbook (largest rested/settled volume)
('CAYPAQDKNWMHRATKU5DQ327VDHVRSIVK7UGVWT2A5SUZCUFTLUHXH2JA', 'testnet', 'settled', 1, '{"name":"settled","prefix_topics":["settled"],"params":[{"name":"market_index","type":"u32","location":"topic"}]}', 'NaraLabs community catalog (Stellar DEX orderbook)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),
('CAYPAQDKNWMHRATKU5DQ327VDHVRSIVK7UGVWT2A5SUZCUFTLUHXH2JA', 'testnet', 'rested', 1, '{"name":"rested","prefix_topics":["rested"],"params":[{"name":"market_index","type":"u32","location":"topic"}]}', 'NaraLabs community catalog (Stellar DEX orderbook)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),
('CAYPAQDKNWMHRATKU5DQ327VDHVRSIVK7UGVWT2A5SUZCUFTLUHXH2JA', 'testnet', 'filled', 1, '{"name":"filled","prefix_topics":["filled"],"params":[{"name":"market_index","type":"u32","location":"topic"}]}', 'NaraLabs community catalog (Stellar DEX orderbook)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),

-- 3) Soroban asset token
('CD4MP2QV4LABFU5OL6Y5RICCLQIX7TWT6T6J546EDYVUXHH6RS2V5S2O', 'testnet', 'transfer', 1, '{"name":"transfer","prefix_topics":["transfer"],"params":[{"name":"from","type":"address","location":"topic"},{"name":"to","type":"address","location":"topic"},{"name":"amount","type":"i128","location":"data"}]}', 'NaraLabs community catalog (Soroban token)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),
('CD4MP2QV4LABFU5OL6Y5RICCLQIX7TWT6T6J546EDYVUXHH6RS2V5S2O', 'testnet', 'approve', 1, '{"name":"approve","prefix_topics":["approve"],"params":[{"name":"from","type":"address","location":"topic"},{"name":"spender","type":"address","location":"topic"},{"name":"amount","type":"i128","location":"data"}]}', 'NaraLabs community catalog (Soroban token)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),

-- 4) USDC vault / pool token activity
('CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA', 'testnet', 'transfer', 1, '{"name":"transfer","prefix_topics":["transfer"],"params":[{"name":"from","type":"address","location":"topic"},{"name":"to","type":"address","location":"topic"},{"name":"asset","type":"string","location":"topic"},{"name":"amount","type":"i128","location":"data"}]}', 'NaraLabs community catalog (pool USDC token)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),
('CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA', 'testnet', 'mint', 1, '{"name":"mint","prefix_topics":["mint"],"params":[{"name":"to","type":"address","location":"topic"},{"name":"amount","type":"i128","location":"data"}]}', 'NaraLabs community catalog (pool USDC token)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),
('CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA', 'testnet', 'burn', 1, '{"name":"burn","prefix_topics":["burn"],"params":[{"name":"from","type":"address","location":"topic"},{"name":"amount","type":"i128","location":"data"}]}', 'NaraLabs community catalog (pool USDC token)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),

-- 5) Blend perpetuals pool
('CB25X5ISYYR5MDGQPWUOMSFAW27MAZWZN34A32J22I4TBCBXV3YEPSAY', 'testnet', 'create_order', 1, '{"name":"create_order","prefix_topics":["create_order"],"params":[{"name":"user","type":"address","location":"topic"},{"name":"market_index","type":"u32","location":"topic"}]}', 'NaraLabs community catalog (Blend perps)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),
('CB25X5ISYYR5MDGQPWUOMSFAW27MAZWZN34A32J22I4TBCBXV3YEPSAY', 'testnet', 'cancel_order', 1, '{"name":"cancel_order","prefix_topics":["cancel_order"],"params":[{"name":"user","type":"address","location":"topic"},{"name":"market_index","type":"u32","location":"topic"}]}', 'NaraLabs community catalog (Blend perps)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),
('CB25X5ISYYR5MDGQPWUOMSFAW27MAZWZN34A32J22I4TBCBXV3YEPSAY', 'testnet', 'open_fill', 1, '{"name":"open_fill","prefix_topics":["open_fill"],"params":[{"name":"user","type":"address","location":"topic"}]}', 'NaraLabs community catalog (Blend perps)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),

-- 6) Blend backstop
('CBSWA5P75NGV2LP5KOY7A7LOAX2CENI5OYBSJ5IVLHENKQJF2I3ZBSYE', 'testnet', 'exposure_synced', 1, '{"name":"exposure_synced","prefix_topics":["exposure_synced"],"params":[{"name":"pool","type":"address","location":"topic"}]}', 'NaraLabs community catalog (Blend backstop)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),

-- 7) Reflector-style oracle
('CB3IUO2Y5NFH7LDX5EOLA63WO7QSYJSOBMMZESLBC62DEPNQJPOYDULC', 'testnet', 'price', 1, '{"name":"price","prefix_topics":["price"],"params":[{"name":"asset","type":"symbol","location":"topic"},{"name":"price","type":"i128","location":"data"}]}', 'NaraLabs community catalog (Reflector oracle)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),

-- 8) Oracle feed updates
('CBTBKYT3OMI4LOUX3UGOZTQSGFAABHHNVRVVGI4SSX3WWNHAIGQAE6TC', 'testnet', 'oracle_updated', 1, '{"name":"oracle_updated","prefix_topics":["oracle_updated"],"params":[{"name":"feed","type":"symbol","location":"topic"}]}', 'NaraLabs community catalog (oracle feed)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),

-- 9) Second Blend backstop module
('CBMLLYBHACT3Y26TOITJZB4KKAOHZPE7FMA27IQGLERESUTO5WJUOVE6', 'testnet', 'exposure_synced', 1, '{"name":"exposure_synced","prefix_topics":["exposure_synced"],"params":[{"name":"pool","type":"address","location":"topic"}]}', 'NaraLabs community catalog (Blend backstop module)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published'),

-- 10) Secondary DEX market
('CAMHFJ32KHIJJIKCE35SRL37JES4QAWLFVLEAYCWVJGP2NZHU47F56F4', 'testnet', 'settled', 1, '{"name":"settled","prefix_topics":["settled"],"params":[{"name":"market_index","type":"u32","location":"topic"}]}', 'NaraLabs community catalog (Stellar DEX market)', '13b21dc5-ff89-48b6-814e-e30b958a77fc', 'verified', 'published');

COMMIT;
