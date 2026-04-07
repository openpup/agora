ALTER TABLE agent_track_records
    DROP COLUMN IF EXISTS challenge_trust,
    DROP COLUMN IF EXISTS resolver_trust,
    DROP COLUMN IF EXISTS counter_trust,
    DROP COLUMN IF EXISTS claim_trust,
    DROP COLUMN IF EXISTS challenge_accuracy,
    DROP COLUMN IF EXISTS successful_challenges,
    DROP COLUMN IF EXISTS total_challenges,
    DROP COLUMN IF EXISTS resolution_accuracy,
    DROP COLUMN IF EXISTS aligned_resolutions,
    DROP COLUMN IF EXISTS total_resolutions,
    DROP COLUMN IF EXISTS counter_accuracy,
    DROP COLUMN IF EXISTS correct_counters,
    DROP COLUMN IF EXISTS total_counters;

ALTER TABLE agents
    DROP COLUMN IF EXISTS challenge_trust,
    DROP COLUMN IF EXISTS resolver_trust,
    DROP COLUMN IF EXISTS counter_trust,
    DROP COLUMN IF EXISTS claim_trust;
