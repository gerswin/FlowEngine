# Sistema de Estados para Personas - Guía Rápida

## Índice

1. [Overview](#overview)
2. [Instalación](#instalación)
3. [Configuración](#configuración)
4. [Uso Básico](#uso-básico)
5. [API Reference](#api-reference)
6. [Ejemplos Completos](#ejemplos-completos)
7. [Troubleshooting](#troubleshooting)

---

## Overview

El **Sistema de Estados para Personas** es una extensión de FlowEngine diseñada para gestionar documentos/solicitudes con soporte para:

✅ Estados jerárquicos con subestados
✅ Escalamientos a departamentos
✅ Reclasificaciones de documentos
✅ Rechazos con reentrada
✅ Guards basados en roles
✅ Auditoría completa (usuario + timestamp)
✅ Eventos para webhooks

### Estados del Flujo

```
Filed (Radicación)
  ↓
Assigned (Asignación)
  ↓
InProgress (Gestión)
  ├─ working
  ├─ escalated_awaiting_response
  └─ escalation_responded
  ↓
InReview (Revisión) ──┐
  ↓                    │ reject
Approved              │
  ↓                    │
Sent [FINAL] ←────────┘
```

---

## Instalación

### Prerrequisitos

- PostgreSQL 15+
- Redis 7+
- Event Dispatcher (WebhookDispatcher, LogDispatcher)
- Go 1.24+

### 1. Ejecutar Migraciones

```bash
# Aplicar migraciones para soporte de estados de personas
psql -h localhost -U flowengine -d flowengine -f migrations/002_person_states_support.up.sql

# Verificar tablas creadas
psql -h localhost -U flowengine -d flowengine -c "\dt workflow_*"
```

**Tablas creadas:**
- `workflow_escalations` - Registro de escalamientos
- `workflow_reclassifications` - Historial de reclasificaciones
- `workflow_rejections` - Tracking de rechazos
- `workflow_assignments` - Historial de asignaciones

**Extensiones a tablas existentes:**
- `workflow_instances` - Agregados `current_sub_state`, `previous_sub_state`
- `workflow_transitions` - Agregados `from_sub_state`, `to_sub_state`, `reason`, `feedback`

### 2. Cargar Workflow YAML

```bash
# Copiar workflow configuration
cp config/templates/person_document_flow.yaml /path/to/workflows/

# O cargar vía API
curl -X POST http://localhost:8080/api/v1/workflows \
  -F "file=@config/templates/person_document_flow.yaml"
```

---

## Configuración

### Variables de Entorno

```bash
# .env
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=flowengine
POSTGRES_USER=flowengine
POSTGRES_PASSWORD=secret

REDIS_ADDR=localhost:6379
REDIS_PASSWORD=

WEBHOOK_URL=https://api.example.com/webhooks
WEBHOOK_SECRET=your-webhook-secret

```

### Configurar Roles

Los siguientes roles deben estar configurados en tu sistema de usuarios:

| Rol | Permisos |
|-----|----------|
| `radicador` | Radicar documentos, Asignar a gestores |
| `asignador` | Asignar y reasignar documentos |
| `gestionador` | Gestionar, Escalar, Reclasificar |
| `revisor` | Revisar, Aprobar, Rechazar |
| `aprobador` | Aprobar y enviar documentos |

---

## Uso Básico

### 1. Radicar un Documento

```bash
curl -X POST http://localhost:8080/api/v1/instances \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "workflow_id": "person_document_flow",
    "actor_id": "user-radicador-001",
    "actor_role": "radicador",
    "data": {
      "tipo": "PQRD",
      "remitente": "Juan Pérez",
      "email": "juan@example.com",
      "asunto": "Solicitud de información sobre trámite",
      "descripcion": "Requiero información sobre el estado de mi trámite #12345"
    }
  }'
```

**Respuesta:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "workflow_id": "person_document_flow",
  "current_state": "filed",
  "current_sub_state": "",
  "status": "running",
  "data": {
    "document_id": "RAD-2025-000001",
    "tipo": "PQRD",
    "remitente": "Juan Pérez",
    ...
  },
  "created_at": "2025-11-05T10:00:00Z",
  "version": 1
}
```

### 2. Asignar a Gestor

```bash
curl -X POST http://localhost:8080/api/v1/instances/550e8400-e29b-41d4-a716-446655440001/events \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "event": "assign_document",
    "actor": "user-radicador-001",
    "data": {
      "assigned_to": "user-gestionador-005",
      "assigned_type": "user"
    }
  }'
```

**Respuesta:**
```json
{
  "instance_id": "550e8400-e29b-41d4-a716-446655440001",
  "previous_state": "filed",
  "current_state": "assigned",
  "version": 2
}
```

### 3. Iniciar Gestión

```bash
curl -X POST http://localhost:8080/api/v1/instances/550e8400-e29b-41d4-a716-446655440001/events \
  -H "Authorization: Bearer $TOKEN_GESTIONADOR" \
  -d '{
    "event": "start_work",
    "actor": "user-gestionador-005"
  }'
```

**Respuesta:**
```json
{
  "current_state": "in_progress",
  "current_sub_state": "working",
  "version": 3
}
```

---

## API Reference

### Transiciones Normales

#### Enviar a Revisión

```http
POST /api/v1/instances/:id/events
```

```json
{
  "event": "submit_for_review",
  "actor": "user-gestionador-005",
  "data": {
    "document_content": "...",
    "justification": "Documento completado según normativa",
    "attachments": ["file1.pdf", "file2.pdf"]
  }
}
```

#### Aprobar Revisión

```http
POST /api/v1/instances/:id/events
```

```json
{
  "event": "approve_review",
  "actor": "user-revisor-002"
}
```

#### Enviar Documento

```http
POST /api/v1/instances/:id/events
```

```json
{
  "event": "send_document",
  "actor": "user-aprobador-001",
  "data": {
    "recipient_email": "juan@example.com",
    "recipient_name": "Juan Pérez"
  }
}
```

---

### Transiciones Especiales

#### 1. Escalamiento

**Endpoint:** `POST /api/v1/instances/:id/escalate`

**Request:**
```json
{
  "department_id": "legal",
  "reason": "Requiere revisión legal especializada debido a cláusulas contractuales"
}
```

**Response:**
```json
{
  "escalation_id": "esc-a1b2c3d4",
  "instance_id": "550e8400-e29b-41d4-a716-446655440001",
  "sub_state": "escalated_awaiting_response"
}
```

**Responder Escalamiento:**

```http
POST /api/v1/instances/:id/escalation-reply
```

```json
{
  "escalation_id": "esc-a1b2c3d4",
  "response": "Revisión legal completada. Aprobado para continuar."
}
```

#### 2. Reclasificación

**Endpoint:** `POST /api/v1/instances/:id/reclassify`

**Request:**
```json
{
  "new_type": "PQRD",
  "reason": "El documento corresponde a una petición formal según análisis"
}
```

**Response:**
```json
{
  "instance_id": "550e8400-e29b-41d4-a716-446655440001",
  "from_type": "Consulta",
  "to_type": "PQRD",
  "reclassified_at": "2025-11-05T14:30:00Z"
}
```

#### 3. Rechazo

**Endpoint:** `POST /api/v1/instances/:id/reject`

**Request:**
```json
{
  "reason": "Documentación incompleta",
  "feedback": "Faltan los siguientes documentos:\n- Cédula del solicitante\n- Comprobante de domicilio\nPor favor adjuntar y reenviar."
}
```

**Response:**
```json
{
  "instance_id": "550e8400-e29b-41d4-a716-446655440001",
  "previous_state": "in_review",
  "current_state": "in_progress",
  "current_sub_state": "working",
  "rejection_count": 1
}
```

---

### Consultas

#### Obtener Historial Completo

```bash
curl http://localhost:8080/api/v1/instances/550e8400-e29b-41d4-a716-446655440001/history \
  -H "Authorization: Bearer $TOKEN"
```

**Response:**
```json
{
  "transitions": [
    {
      "id": "trans-001",
      "event": "assign_document",
      "from_state": "filed",
      "to_state": "assigned",
      "actor": "user-radicador-001",
      "actor_role": "radicador",
      "created_at": "2025-11-05T10:05:00Z"
    },
    {
      "id": "trans-002",
      "event": "start_work",
      "from_state": "assigned",
      "to_state": "in_progress",
      "to_sub_state": "working",
      "actor": "user-gestionador-005",
      "created_at": "2025-11-05T11:00:00Z"
    }
  ],
  "escalations": [
    {
      "id": "esc-a1b2c3d4",
      "department_id": "legal",
      "reason": "Requiere revisión legal",
      "status": "responded",
      "escalated_at": "2025-11-05T12:00:00Z",
      "responded_at": "2025-11-05T14:00:00Z"
    }
  ],
  "rejections": [],
  "total_events": 2
}
```

#### Obtener Escalamientos Pendientes

```bash
curl http://localhost:8080/api/v1/instances/550e8400-e29b-41d4-a716-446655440001/escalations \
  -H "Authorization: Bearer $TOKEN"
```

#### Consultar por Estado y Subestado

```bash
curl -X POST http://localhost:8080/api/v1/queries/instances \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "workflow_id": "person_document_flow",
    "states": ["in_progress"],
    "sub_states": ["escalated_awaiting_response"],
    "limit": 50
  }'
```

---

## Ejemplos Completos

### Ejemplo 1: Flujo Completo (Happy Path)

```bash
#!/bin/bash

BASE_URL="http://localhost:8080/api/v1"
TOKEN="your-jwt-token"

# 1. Radicar
INSTANCE=$(curl -s -X POST $BASE_URL/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "person_document_flow",
    "actor_id": "rad-001",
    "actor_role": "radicador",
    "data": {"tipo": "PQRD", "remitente": "Juan Pérez"}
  }' | jq -r '.id')

echo "Instance created: $INSTANCE"

# 2. Asignar
curl -X POST $BASE_URL/instances/$INSTANCE/events \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"event": "assign_document", "actor": "rad-001", "data": {"assigned_to": "gest-001"}}'

# 3. Iniciar gestión
curl -X POST $BASE_URL/instances/$INSTANCE/events \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"event": "start_work", "actor": "gest-001"}'

# 4. Enviar a revisión
curl -X POST $BASE_URL/instances/$INSTANCE/events \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"event": "submit_for_review", "actor": "gest-001"}'

# 5. Aprobar
curl -X POST $BASE_URL/instances/$INSTANCE/events \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"event": "approve_review", "actor": "rev-001"}'

# 6. Enviar
curl -X POST $BASE_URL/instances/$INSTANCE/events \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"event": "send_document", "actor": "apr-001"}'

echo "Document sent successfully!"
```

### Ejemplo 2: Flujo con Escalamiento

```bash
#!/bin/bash

BASE_URL="http://localhost:8080/api/v1"
INSTANCE="550e8400-e29b-41d4-a716-446655440001"

# Durante gestión, escalar
ESCALATION=$(curl -s -X POST $BASE_URL/instances/$INSTANCE/escalate \
  -H "Content-Type: application/json" \
  -d '{
    "department_id": "legal",
    "reason": "Requiere revisión legal"
  }' | jq -r '.escalation_id')

echo "Escalation created: $ESCALATION"

# Esperar respuesta del departamento legal...
# (En producción, esto vendría de otro sistema o usuario)

# Responder escalamiento
curl -X POST $BASE_URL/instances/$INSTANCE/escalation-reply \
  -H "Content-Type: application/json" \
  -d "{
    \"escalation_id\": \"$ESCALATION\",
    \"response\": \"Aprobado legalmente\"
  }"

# Continuar con flujo normal
curl -X POST $BASE_URL/instances/$INSTANCE/events \
  -d '{"event": "submit_for_review", "actor": "gest-001"}'
```

### Ejemplo 3: Flujo con Rechazo y Corrección

```bash
#!/bin/bash

BASE_URL="http://localhost:8080/api/v1"
INSTANCE="550e8400-e29b-41d4-a716-446655440001"

# Revisor rechaza
curl -X POST $BASE_URL/instances/$INSTANCE/reject \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Documentación incompleta",
    "feedback": "Falta adjuntar cédula"
  }'

# Documento vuelve a estado in_progress
# Gestor corrige y reenvía

curl -X POST $BASE_URL/instances/$INSTANCE/events \
  -d '{
    "event": "submit_for_review",
    "actor": "gest-001",
    "data": {
      "correction_notes": "Cédula adjuntada",
      "attachments": ["cedula.pdf"]
    }
  }'
```

---

## Webhooks

### Configuración

Los webhooks se configuran en el workflow YAML:

```yaml
webhooks:
  - url: "https://api.example.com/state-changes"
    events:
      - state.changed
      - document.escalated
      - document.rejected
    secret: "webhook-secret"
```

### Payload de Webhook

**Ejemplo: document.escalated**

```json
{
  "event_type": "document.escalated",
  "timestamp": "2025-11-05T12:00:00Z",
  "instance_id": "550e8400-e29b-41d4-a716-446655440001",
  "workflow_id": "person_document_flow",
  "data": {
    "department_id": "legal",
    "reason": "Requiere revisión legal",
    "escalated_by": "user-gest-005",
    "is_auto": false
  },
  "signature": "sha256=a1b2c3d4..."
}
```

### Verificar Firma HMAC

```python
import hmac
import hashlib

def verify_webhook(payload, signature, secret):
    expected = hmac.new(
        secret.encode(),
        payload.encode(),
        hashlib.sha256
    ).hexdigest()

    return hmac.compare_digest(f"sha256={expected}", signature)
```

---

## Troubleshooting

### Error: "current state does not support sub-states"

**Causa:** Intentas establecer un subestado en un estado que no los soporta.

**Solución:** Solo el estado `in_progress` soporta subestados. Verifica que la instancia esté en este estado antes de usar `SetSubState`.

### Error: "actor does not have required role"

**Causa:** El usuario no tiene el rol necesario para la transición.

**Solución:** Verifica que el usuario tenga el rol correcto:
- `filed` → `assigned`: requiere rol `radicador`
- `in_progress`: requiere rol `gestionador`
- `in_review`: requiere rol `revisor`
- `approved`: requiere rol `aprobador`

### Error: "can only escalate from in_progress state"

**Causa:** Intentas escalar desde un estado que no permite escalamientos.

**Solución:** Solo puedes escalar documentos en estado `in_progress`.

### Error: "version conflict"

**Causa:** Optimistic locking - la instancia fue modificada por otro proceso.

**Solución:** Recargar la instancia y reintentar la operación.

---

## Métricas y Monitoring

### Métricas Disponibles

```promql
# Tasa de rechazo
rate(document_rejected_total[1h]) / rate(submit_for_review_total[1h])

# Tiempo promedio por estado
histogram_quantile(0.5, state_duration_seconds_bucket{state="in_progress"})

# Escalamientos por departamento
escalations_total{department_id="legal"}

# Documentos por estado
documents_by_state{state="in_review"}
```

### Dashboard Recomendado

- **Panel 1:** Documentos por estado (pie chart)
- **Panel 2:** Tiempo promedio en cada estado (bar chart)
- **Panel 3:** Tasa de rechazo (time series)
- **Panel 4:** Escalamientos pendientes (gauge)

---

## Soporte

- **Documentación Técnica:** `docs/person-state-design.md`
- **Implementación:** `docs/person-state-implementation.md`
- **Issues:** https://github.com/LaFabric-LinkTIC/FlowEngine/issues

---

**Versión:** 1.0
**Última actualización:** 2025-11-05
