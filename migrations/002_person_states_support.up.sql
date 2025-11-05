-- ============================================================================
-- Migration: 002_person_states_support
-- Description: Agregar soporte para estados de personas (documentos) con:
--              - Subestados (substates)
--              - Escalamientos
--              - Auditoría extendida
-- Date: 2025-11-05
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. EXTENDER TABLA workflow_instances CON SUBESTADOS
-- ----------------------------------------------------------------------------

ALTER TABLE workflow_instances
ADD COLUMN current_sub_state VARCHAR(100),
ADD COLUMN previous_sub_state VARCHAR(100);

COMMENT ON COLUMN workflow_instances.current_sub_state IS 'Subestado actual dentro del estado principal (ej: escalated_awaiting_response)';
COMMENT ON COLUMN workflow_instances.previous_sub_state IS 'Subestado previo antes de la última transición';

-- Índice para queries por substate
CREATE INDEX idx_instances_sub_state
ON workflow_instances(current_state, current_sub_state)
WHERE current_sub_state IS NOT NULL;

-- ----------------------------------------------------------------------------
-- 2. EXTENDER TABLA workflow_transitions CON INFORMACIÓN DE SUBESTADOS
-- ----------------------------------------------------------------------------

ALTER TABLE workflow_transitions
ADD COLUMN from_sub_state VARCHAR(100),
ADD COLUMN to_sub_state VARCHAR(100),
ADD COLUMN reason TEXT,
ADD COLUMN feedback TEXT;

COMMENT ON COLUMN workflow_transitions.from_sub_state IS 'Subestado de origen';
COMMENT ON COLUMN workflow_transitions.to_sub_state IS 'Subestado de destino';
COMMENT ON COLUMN workflow_transitions.reason IS 'Razón de la transición (para Reject, Escalate, etc)';
COMMENT ON COLUMN workflow_transitions.feedback IS 'Feedback adicional (usado en rechazos)';

-- Índice para búsqueda de rechazos
CREATE INDEX idx_transitions_event_type
ON workflow_transitions(event)
WHERE event IN ('reject', 'escalate', 'reclassify_to_pqrd', 'reclassify_to_control');

-- Índice para auditoría por actor
CREATE INDEX idx_transitions_actor_time
ON workflow_transitions(actor, created_at DESC);

-- ----------------------------------------------------------------------------
-- 3. TABLA DE ESCALAMIENTOS
-- ----------------------------------------------------------------------------

CREATE TABLE workflow_escalations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    transition_id UUID REFERENCES workflow_transitions(id),

    -- Información del escalamiento
    department_id VARCHAR(100) NOT NULL,
    reason TEXT NOT NULL,
    is_auto_escalation BOOLEAN NOT NULL DEFAULT FALSE,

    -- Actor que escaló
    escalated_by VARCHAR(255) NOT NULL,
    escalated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Respuesta del escalamiento
    response TEXT,
    responded_by VARCHAR(255),
    responded_at TIMESTAMP,

    -- Estado del escalamiento
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    -- pending: esperando respuesta
    -- responded: respuesta recibida
    -- closed: cerrado sin respuesta
    -- cancelled: cancelado

    -- Metadata
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT check_escalation_status CHECK (
        status IN ('pending', 'responded', 'closed', 'cancelled')
    ),
    CONSTRAINT check_responded_at CHECK (
        (status = 'responded' AND responded_at IS NOT NULL AND responded_by IS NOT NULL) OR
        (status != 'responded')
    )
);

COMMENT ON TABLE workflow_escalations IS 'Registros de escalamientos de documentos a otros departamentos';
COMMENT ON COLUMN workflow_escalations.department_id IS 'ID del departamento al que se escaló';
COMMENT ON COLUMN workflow_escalations.is_auto_escalation IS 'TRUE si fue escalamiento automático por timeout';
COMMENT ON COLUMN workflow_escalations.status IS 'Estado actual del escalamiento';

-- Índices para escalamientos
CREATE INDEX idx_escalations_instance ON workflow_escalations(instance_id, created_at DESC);
CREATE INDEX idx_escalations_status ON workflow_escalations(status) WHERE status = 'pending';
CREATE INDEX idx_escalations_department ON workflow_escalations(department_id, status, created_at DESC);
CREATE INDEX idx_escalations_actor ON workflow_escalations(escalated_by, created_at DESC);

-- ----------------------------------------------------------------------------
-- 4. TABLA DE RECLASIFICACIONES
-- ----------------------------------------------------------------------------

CREATE TABLE workflow_reclassifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    transition_id UUID REFERENCES workflow_transitions(id),

    -- Información de la reclasificación
    from_type VARCHAR(100) NOT NULL,
    to_type VARCHAR(100) NOT NULL,
    reason TEXT NOT NULL,

    -- Actor que reclasificó
    reclassified_by VARCHAR(255) NOT NULL,
    reclassified_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Metadata
    approved_by VARCHAR(255),  -- Si requiere aprobación
    approved_at TIMESTAMP,
    metadata JSONB,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT check_different_types CHECK (from_type != to_type)
);

COMMENT ON TABLE workflow_reclassifications IS 'Historial de reclasificaciones de documentos';
COMMENT ON COLUMN workflow_reclassifications.from_type IS 'Tipo original del documento';
COMMENT ON COLUMN workflow_reclassifications.to_type IS 'Nuevo tipo del documento';

-- Índices para reclasificaciones
CREATE INDEX idx_reclassifications_instance ON workflow_reclassifications(instance_id, created_at DESC);
CREATE INDEX idx_reclassifications_type ON workflow_reclassifications(to_type, created_at DESC);
CREATE INDEX idx_reclassifications_actor ON workflow_reclassifications(reclassified_by, created_at DESC);

-- ----------------------------------------------------------------------------
-- 5. TABLA DE RECHAZOS (Para tracking detallado)
-- ----------------------------------------------------------------------------

CREATE TABLE workflow_rejections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    transition_id UUID REFERENCES workflow_transitions(id),

    -- Información del rechazo
    reason TEXT NOT NULL,
    feedback TEXT NOT NULL,
    from_state VARCHAR(100) NOT NULL,

    -- Actor que rechazó
    rejected_by VARCHAR(255) NOT NULL,
    rejected_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Corrección
    corrected BOOLEAN NOT NULL DEFAULT FALSE,
    corrected_at TIMESTAMP,
    correction_notes TEXT,

    -- Metadata
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE workflow_rejections IS 'Historial de rechazos de documentos con feedback';
COMMENT ON COLUMN workflow_rejections.corrected IS 'TRUE cuando el documento fue corregido y reenviado';

-- Índices para rechazos
CREATE INDEX idx_rejections_instance ON workflow_rejections(instance_id, created_at DESC);
CREATE INDEX idx_rejections_pending ON workflow_rejections(corrected) WHERE corrected = FALSE;
CREATE INDEX idx_rejections_actor ON workflow_rejections(rejected_by, created_at DESC);

-- ----------------------------------------------------------------------------
-- 6. TABLA DE ASIGNACIONES (Tracking de asignaciones)
-- ----------------------------------------------------------------------------

CREATE TABLE workflow_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,

    -- Información de la asignación
    assigned_to VARCHAR(255) NOT NULL,  -- User ID o Group ID
    assigned_to_type VARCHAR(50) NOT NULL DEFAULT 'user',  -- 'user' o 'group'
    assigned_by VARCHAR(255) NOT NULL,
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Reasignación
    is_reassignment BOOLEAN NOT NULL DEFAULT FALSE,
    previous_assignee VARCHAR(255),
    reassignment_reason TEXT,

    -- Estado de la asignación
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    -- active: asignación activa
    -- completed: trabajo completado
    -- reassigned: fue reasignado a otro

    unassigned_at TIMESTAMP,

    -- Metadata
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT check_assignment_type CHECK (assigned_to_type IN ('user', 'group')),
    CONSTRAINT check_assignment_status CHECK (status IN ('active', 'completed', 'reassigned'))
);

COMMENT ON TABLE workflow_assignments IS 'Historial de asignaciones de documentos a usuarios o grupos';

-- Índices para asignaciones
CREATE INDEX idx_assignments_instance ON workflow_assignments(instance_id, created_at DESC);
CREATE INDEX idx_assignments_assignee ON workflow_assignments(assigned_to, status, created_at DESC);
CREATE INDEX idx_assignments_active ON workflow_assignments(status) WHERE status = 'active';

-- ----------------------------------------------------------------------------
-- 7. VISTA: Estadísticas de Documentos
-- ----------------------------------------------------------------------------

CREATE OR REPLACE VIEW v_document_statistics AS
SELECT
    wi.workflow_id,
    wi.current_state,
    wi.current_sub_state,
    wi.status,
    COUNT(*) as count,
    AVG(EXTRACT(EPOCH FROM (COALESCE(wi.completed_at, NOW()) - wi.created_at)) / 3600) as avg_duration_hours,
    MIN(wi.created_at) as oldest_created_at,
    MAX(wi.updated_at) as latest_updated_at
FROM workflow_instances wi
GROUP BY wi.workflow_id, wi.current_state, wi.current_sub_state, wi.status;

COMMENT ON VIEW v_document_statistics IS 'Estadísticas agregadas de documentos por workflow y estado';

-- ----------------------------------------------------------------------------
-- 8. VISTA: Escalamientos Pendientes
-- ----------------------------------------------------------------------------

CREATE OR REPLACE VIEW v_pending_escalations AS
SELECT
    e.id as escalation_id,
    e.instance_id,
    wi.workflow_id,
    wi.data->>'document_id' as document_id,
    e.department_id,
    e.reason,
    e.is_auto_escalation,
    e.escalated_by,
    e.escalated_at,
    EXTRACT(EPOCH FROM (NOW() - e.escalated_at)) / 3600 as hours_pending,
    wi.current_state,
    wi.current_sub_state
FROM workflow_escalations e
INNER JOIN workflow_instances wi ON wi.id = e.instance_id
WHERE e.status = 'pending'
ORDER BY e.escalated_at ASC;

COMMENT ON VIEW v_pending_escalations IS 'Vista de escalamientos pendientes con información del documento';

-- ----------------------------------------------------------------------------
-- 9. VISTA: Documentos Rechazados Pendientes de Corrección
-- ----------------------------------------------------------------------------

CREATE OR REPLACE VIEW v_pending_rejections AS
SELECT
    r.id as rejection_id,
    r.instance_id,
    wi.workflow_id,
    wi.data->>'document_id' as document_id,
    r.reason,
    r.feedback,
    r.rejected_by,
    r.rejected_at,
    EXTRACT(EPOCH FROM (NOW() - r.rejected_at)) / 3600 as hours_since_rejection,
    wi.current_state,
    wi.current_actor
FROM workflow_rejections r
INNER JOIN workflow_instances wi ON wi.id = r.instance_id
WHERE r.corrected = FALSE
ORDER BY r.rejected_at ASC;

COMMENT ON VIEW v_pending_rejections IS 'Vista de rechazos pendientes de corrección';

-- ----------------------------------------------------------------------------
-- 10. FUNCIÓN: Obtener Historial Completo de una Instancia
-- ----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION get_instance_full_history(p_instance_id UUID)
RETURNS TABLE (
    event_type VARCHAR(50),
    event_name VARCHAR(100),
    from_state VARCHAR(100),
    to_state VARCHAR(100),
    from_sub_state VARCHAR(100),
    to_sub_state VARCHAR(100),
    actor VARCHAR(255),
    actor_role VARCHAR(100),
    reason TEXT,
    feedback TEXT,
    occurred_at TIMESTAMP,
    additional_data JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        'transition'::VARCHAR(50) as event_type,
        t.event as event_name,
        t.from_state,
        t.to_state,
        t.from_sub_state,
        t.to_sub_state,
        t.actor,
        t.actor_role,
        t.reason,
        t.feedback,
        t.created_at as occurred_at,
        t.data as additional_data
    FROM workflow_transitions t
    WHERE t.instance_id = p_instance_id

    UNION ALL

    SELECT
        'escalation'::VARCHAR(50),
        'escalate'::VARCHAR(100),
        NULL, NULL, NULL, NULL,
        e.escalated_by,
        NULL,
        e.reason,
        e.response,
        e.escalated_at,
        jsonb_build_object(
            'department_id', e.department_id,
            'status', e.status,
            'is_auto', e.is_auto_escalation
        )
    FROM workflow_escalations e
    WHERE e.instance_id = p_instance_id

    UNION ALL

    SELECT
        'rejection'::VARCHAR(50),
        'reject'::VARCHAR(100),
        r.from_state,
        NULL, NULL, NULL,
        r.rejected_by,
        NULL,
        r.reason,
        r.feedback,
        r.rejected_at,
        jsonb_build_object('corrected', r.corrected)
    FROM workflow_rejections r
    WHERE r.instance_id = p_instance_id

    UNION ALL

    SELECT
        'assignment'::VARCHAR(50),
        CASE WHEN a.is_reassignment THEN 'reassign' ELSE 'assign' END,
        NULL, NULL, NULL, NULL,
        a.assigned_by,
        NULL,
        a.reassignment_reason,
        NULL,
        a.assigned_at,
        jsonb_build_object(
            'assigned_to', a.assigned_to,
            'assigned_to_type', a.assigned_to_type,
            'status', a.status
        )
    FROM workflow_assignments a
    WHERE a.instance_id = p_instance_id

    ORDER BY occurred_at ASC;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_instance_full_history IS 'Obtiene el historial completo de eventos de una instancia incluyendo transiciones, escalamientos, rechazos y asignaciones';

-- ----------------------------------------------------------------------------
-- 11. FUNCIÓN: Calcular Métricas de Rendimiento por Estado
-- ----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION calculate_state_performance(
    p_workflow_id VARCHAR(255),
    p_from_date TIMESTAMP DEFAULT NOW() - INTERVAL '30 days'
)
RETURNS TABLE (
    state_id VARCHAR(100),
    total_instances BIGINT,
    avg_duration_hours NUMERIC,
    min_duration_hours NUMERIC,
    max_duration_hours NUMERIC,
    median_duration_hours NUMERIC,
    timeouts_count BIGINT,
    rejection_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    WITH state_durations AS (
        SELECT
            t.to_state as state,
            t.instance_id,
            EXTRACT(EPOCH FROM (
                LEAD(t.created_at) OVER (PARTITION BY t.instance_id ORDER BY t.created_at) - t.created_at
            )) / 3600 as duration_hours
        FROM workflow_transitions t
        INNER JOIN workflow_instances wi ON wi.id = t.instance_id
        WHERE wi.workflow_id = p_workflow_id
          AND t.created_at >= p_from_date
    )
    SELECT
        sd.state,
        COUNT(DISTINCT sd.instance_id)::BIGINT,
        ROUND(AVG(sd.duration_hours)::NUMERIC, 2),
        ROUND(MIN(sd.duration_hours)::NUMERIC, 2),
        ROUND(MAX(sd.duration_hours)::NUMERIC, 2),
        ROUND(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY sd.duration_hours)::NUMERIC, 2),
        COUNT(DISTINCT CASE WHEN EXISTS (
            SELECT 1 FROM workflow_transitions t2
            WHERE t2.instance_id = sd.instance_id
              AND t2.event = 'escalate_timeout'
        ) THEN sd.instance_id END)::BIGINT,
        COUNT(DISTINCT CASE WHEN EXISTS (
            SELECT 1 FROM workflow_rejections r
            WHERE r.instance_id = sd.instance_id
        ) THEN sd.instance_id END)::BIGINT
    FROM state_durations sd
    WHERE sd.duration_hours IS NOT NULL
    GROUP BY sd.state
    ORDER BY sd.state;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION calculate_state_performance IS 'Calcula métricas de rendimiento por estado para un workflow';

-- ----------------------------------------------------------------------------
-- 12. TRIGGERS: Actualizar updated_at automáticamente
-- ----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER escalations_updated_at
    BEFORE UPDATE ON workflow_escalations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TRIGGER escalations_updated_at ON workflow_escalations IS 'Actualiza el campo updated_at automáticamente';

-- ----------------------------------------------------------------------------
-- 13. DATOS DE EJEMPLO (Opcional - comentado por defecto)
-- ----------------------------------------------------------------------------

-- Descomentar para insertar datos de ejemplo

/*
-- Insertar workflow de ejemplo
INSERT INTO workflows (id, name, description, version, config) VALUES (
    'person_document_flow',
    'Flujo de Radicación de Documentos (Personas)',
    'Sistema completo de estados para documentos con escalamientos y rechazos',
    '1.0',
    '{"initial_state": "filed"}'::jsonb
);

-- Insertar instancia de ejemplo
INSERT INTO workflow_instances (id, workflow_id, current_state, current_sub_state, status, data) VALUES (
    'a0000000-0000-0000-0000-000000000001'::UUID,
    'person_document_flow',
    'in_progress',
    'working',
    'running',
    '{"document_id": "RAD-2025-000001", "tipo": "PQRD", "remitente": "Juan Pérez"}'::jsonb
);

-- Insertar escalamiento de ejemplo
INSERT INTO workflow_escalations (instance_id, department_id, reason, escalated_by) VALUES (
    'a0000000-0000-0000-0000-000000000001'::UUID,
    'legal',
    'Requiere revisión legal especializada',
    'user-gest-001'
);
*/

-- ============================================================================
-- FIN DE MIGRACIÓN
-- ============================================================================
