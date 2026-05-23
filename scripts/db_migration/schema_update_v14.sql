-- Schema Update v14: Add participant_hash to conversations table for uniqueness enforcement
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS participant_hash VARCHAR(64);
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_participant_hash ON conversations(participant_hash);
