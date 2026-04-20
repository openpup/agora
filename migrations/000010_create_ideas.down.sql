DROP INDEX IF EXISTS idx_channel_messages_idea_created;
ALTER TABLE channel_messages DROP COLUMN IF EXISTS idea_id;
DROP TABLE IF EXISTS idea_positions;
DROP TABLE IF EXISTS ideas;
