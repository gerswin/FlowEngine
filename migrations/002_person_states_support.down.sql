-- ============================================================================
-- Migration Rollback: 002_person_states_support
-- Description: Revertir cambios de soporte para estados de personas
-- Date: 2025-11-05
-- ============================================================================

-- Drop functions
DROP FUNCTION IF EXISTS calculate_state_performance(VARCHAR, TIMESTAMP);
DROP FUNCTION IF EXISTS get_instance_full_history(UUID);
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;

-- Drop views
DROP VIEW IF EXISTS v_pending_rejections;
DROP VIEW IF EXISTS v_pending_escalations;
DROP VIEW IF EXISTS v_document_statistics;

-- Drop tables (en orden inverso por dependencias)
DROP TABLE IF EXISTS workflow_assignments;
DROP TABLE IF EXISTS workflow_rejections;
DROP TABLE IF EXISTS workflow_reclassifications;
DROP TABLE IF EXISTS workflow_escalations;

-- Remove columns from workflow_transitions
ALTER TABLE workflow_transitions
DROP COLUMN IF EXISTS feedback,
DROP COLUMN IF EXISTS reason,
DROP COLUMN IF EXISTS to_sub_state,
DROP COLUMN IF EXISTS from_sub_state;

-- Drop índices de workflow_transitions
DROP INDEX IF EXISTS idx_transitions_actor_time;
DROP INDEX IF EXISTS idx_transitions_event_type;

-- Remove columns from workflow_instances
ALTER TABLE workflow_instances
DROP COLUMN IF EXISTS previous_sub_state,
DROP COLUMN IF EXISTS current_sub_state;

-- Drop índice de workflow_instances
DROP INDEX IF EXISTS idx_instances_sub_state;

-- ============================================================================
-- FIN DE ROLLBACK
-- ============================================================================
