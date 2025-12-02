# FlowEngine REST API - Quick Start Guide

## Inicio Rápido

### 1. Iniciar el servidor

```bash
# Ejecutar el servidor API
make run

# O directamente:
go run cmd/api/main.go
```

El servidor estará disponible en `http://localhost:8080`

### 2. Verificar estado del servidor

```bash
curl http://localhost:8080/health
```

Respuesta:
```json
{
  "status": "healthy",
  "service": "FlowEngine",
  "version": "0.1.0"
}
```

---

## Endpoints Disponibles

### Workflows

#### 1. Crear un Workflow

**`POST /api/v1/workflows`**

Crea una nueva definición de workflow.

**Request Body:**
```json
{
  "name": "Proceso de Aprobación",
  "description": "Workflow simple de aprobación con 3 estados",
  "created_by": "550e8400-e29b-41d4-a716-446655440000",
  "initial_state": {
    "id": "draft",
    "name": "Borrador",
    "description": "Documento en borrador",
    "is_final": false
  },
  "states": [
    {
      "id": "draft",
      "name": "Borrador",
      "description": "Documento en borrador",
      "is_final": false
    },
    {
      "id": "review",
      "name": "En Revisión",
      "description": "Documento siendo revisado",
      "is_final": false
    },
    {
      "id": "approved",
      "name": "Aprobado",
      "description": "Documento aprobado",
      "is_final": true
    },
    {
      "id": "rejected",
      "name": "Rechazado",
      "description": "Documento rechazado",
      "is_final": true
    }
  ],
  "events": [
    {
      "name": "submit",
      "sources": ["draft"],
      "destination": "review"
    },
    {
      "name": "approve",
      "sources": ["review"],
      "destination": "approved"
    },
    {
      "name": "reject",
      "sources": ["review"],
      "destination": "rejected"
    }
  ]
}
```

**Response (201 Created):**
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "name": "Proceso de Aprobación",
  "version": "1.0.0"
}
```

**cURL Example:**
```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/create_workflow.json
```

---

#### 2. Listar Workflows

**`GET /api/v1/workflows`**

Obtiene todos los workflows disponibles.

**Response (200 OK):**
```json
{
  "workflows": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "name": "Proceso de Aprobación",
      "description": "Workflow simple de aprobación",
      "version": "1.0.0",
      "initial_state": "draft",
      "states": [...],
      "events": [...],
      "created_at": "2025-01-19T10:30:00Z",
      "updated_at": "2025-01-19T10:30:00Z"
    }
  ],
  "count": 1
}
```

**cURL Example:**
```bash
curl http://localhost:8080/api/v1/workflows
```

---

#### 3. Obtener Workflow por ID

**`GET /api/v1/workflows/:id`**

Obtiene los detalles completos de un workflow.

**Response (200 OK):**
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "name": "Proceso de Aprobación",
  "description": "Workflow simple de aprobación",
  "version": "1.0.0",
  "initial_state": "draft",
  "states": [
    {
      "id": "draft",
      "name": "Borrador",
      "description": "Documento en borrador",
      "is_final": false
    },
    {
      "id": "review",
      "name": "En Revisión",
      "description": "Documento siendo revisado",
      "is_final": false
    },
    {
      "id": "approved",
      "name": "Aprobado",
      "description": "Documento aprobado",
      "is_final": true
    }
  ],
  "events": [
    {
      "name": "submit",
      "sources": ["draft"],
      "destination": "review"
    },
    {
      "name": "approve",
      "sources": ["review"],
      "destination": "approved"
    }
  ],
  "created_at": "2025-01-19T10:30:00Z",
  "updated_at": "2025-01-19T10:30:00Z"
}
```

**cURL Example:**
```bash
curl http://localhost:8080/api/v1/workflows/a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

---

### Instances (Instancias de Workflow)

#### 4. Crear una Instancia

**`POST /api/v1/instances`**

Crea una nueva instancia de un workflow existente.

**Request Body:**
```json
{
  "workflow_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "started_by": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "title": "Propuesta de Proyecto Q1 2025",
    "requester": "Juan Pérez",
    "department": "IT",
    "amount": 50000
  },
  "variables": {
    "priority": "high",
    "sla_hours": 48
  }
}
```

**Response (201 Created):**
```json
{
  "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
  "workflow_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "workflow_name": "Proceso de Aprobación",
  "current_state": "draft",
  "status": "running",
  "version": "1",
  "created_at": "2025-01-19T11:00:00Z"
}
```

**cURL Example:**
```bash
curl -X POST http://localhost:8080/api/v1/instances \
  -H "Content-Type: application/json" \
  -d @examples/create_instance.json
```

---

#### 5. Listar Instancias

**`GET /api/v1/instances`**

Obtiene todas las instancias.

**Query Parameters:**
- `workflow_id` (opcional): Filtrar por workflow específico

**Response (200 OK):**
```json
{
  "instances": [
    {
      "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "workflow_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "workflow_name": "Proceso de Aprobación",
      "current_state": "draft",
      "status": "running",
      "version": "1",
      "data": {...},
      "variables": {...},
      "transitions": [],
      "created_at": "2025-01-19T11:00:00Z",
      "updated_at": "2025-01-19T11:00:00Z",
      "completed_at": "",
      "transition_count": 0
    }
  ],
  "count": 1
}
```

**cURL Examples:**
```bash
# Todas las instancias
curl http://localhost:8080/api/v1/instances

# Filtrar por workflow
curl http://localhost:8080/api/v1/instances?workflow_id=a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

---

#### 6. Obtener Instancia por ID

**`GET /api/v1/instances/:id`**

Obtiene los detalles completos de una instancia, incluyendo historial de transiciones.

**Response (200 OK):**
```json
{
  "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
  "workflow_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "workflow_name": "Proceso de Aprobación",
  "current_state": "review",
  "status": "running",
  "version": "2",
  "data": {
    "title": "Propuesta de Proyecto Q1 2025",
    "requester": "Juan Pérez"
  },
  "variables": {
    "priority": "high"
  },
  "transitions": [
    {
      "id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
      "from": "draft",
      "to": "review",
      "event": "submit",
      "actor": "550e8400-e29b-41d4-a716-446655440000",
      "timestamp": "2025-01-19T11:15:00Z",
      "reason": "Enviando para aprobación",
      "feedback": "",
      "metadata": {}
    }
  ],
  "created_at": "2025-01-19T11:00:00Z",
  "updated_at": "2025-01-19T11:15:00Z",
  "completed_at": "",
  "transition_count": 1
}
```

**cURL Example:**
```bash
curl http://localhost:8080/api/v1/instances/b2c3d4e5-f6a7-8901-bcde-f12345678901
```

---

#### 7. Ejecutar Transición

**`POST /api/v1/instances/:id/transitions`**

Ejecuta una transición de estado en una instancia.

**Request Body:**
```json
{
  "event": "approve",
  "actor_id": "660f9511-f3ac-52e5-b827-557766551111",
  "reason": "Propuesta aprobada por el comité",
  "feedback": "Excelente propuesta, adelante con el proyecto",
  "metadata": {
    "reviewer": "María García",
    "review_score": 95,
    "approved_amount": 50000
  },
  "data": {
    "approval_date": "2025-01-19",
    "approved_by": "María García"
  }
}
```

**Response (200 OK):**
```json
{
  "instance_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
  "previous_state": "review",
  "current_state": "approved",
  "status": "running",
  "version": "3",
  "transition_id": "d4e5f6a7-b8c9-0123-def0-234567890123"
}
```

**cURL Example:**
```bash
curl -X POST http://localhost:8080/api/v1/instances/b2c3d4e5-f6a7-8901-bcde-f12345678901/transitions \
  -H "Content-Type: application/json" \
  -d @examples/transition_instance.json
```

---

## Manejo de Errores

Todos los errores siguen este formato:

```json
{
  "error": "NOT_FOUND",
  "message": "workflow not found: invalid-id",
  "code": "NOT_FOUND",
  "context": {
    "workflow_id": "invalid-id"
  }
}
```

### Códigos de Error Comunes

| HTTP Status | Error Code | Descripción |
|-------------|------------|-------------|
| 400 | INVALID_INPUT | Request inválido o datos malformados |
| 404 | NOT_FOUND | Recurso no encontrado |
| 409 | CONFLICT | Conflicto de estado o versión |
| 409 | INVALID_STATE | Transición inválida |
| 500 | INTERNAL_ERROR | Error interno del servidor |

---

## Ejemplo Completo: Flujo de Trabajo

```bash
# 1. Crear workflow
WORKFLOW_ID=$(curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/create_workflow.json | jq -r '.id')

echo "Workflow creado: $WORKFLOW_ID"

# 2. Crear instancia
INSTANCE_ID=$(curl -X POST http://localhost:8080/api/v1/instances \
  -H "Content-Type: application/json" \
  -d "{
    \"workflow_id\": \"$WORKFLOW_ID\",
    \"started_by\": \"550e8400-e29b-41d4-a716-446655440000\",
    \"data\": {\"title\": \"Test Document\"}
  }" | jq -r '.id')

echo "Instancia creada: $INSTANCE_ID"

# 3. Ejecutar transición: submit
curl -X POST http://localhost:8080/api/v1/instances/$INSTANCE_ID/transitions \
  -H "Content-Type: application/json" \
  -d "{
    \"event\": \"submit\",
    \"actor_id\": \"550e8400-e29b-41d4-a716-446655440000\",
    \"reason\": \"Enviando para revisión\"
  }"

# 4. Ejecutar transición: approve
curl -X POST http://localhost:8080/api/v1/instances/$INSTANCE_ID/transitions \
  -H "Content-Type: application/json" \
  -d "{
    \"event\": \"approve\",
    \"actor_id\": \"660f9511-f3ac-52e5-b827-557766551111\",
    \"reason\": \"Aprobado\",
    \"feedback\": \"Todo en orden\"
  }"

# 5. Ver estado final
curl http://localhost:8080/api/v1/instances/$INSTANCE_ID | jq
```

---

## Próximos Pasos

- Consulta `examples/` para ver archivos JSON de ejemplo
- Revisa el código en `cmd/demo/` para ver ejemplos programáticos
- Lee `requirements.md` para entender los requerimientos completos del sistema

---

## Notas Importantes

⚠️ **Estado Actual**: Esta implementación usa repositorios **in-memory**.
- Los datos se pierden al reiniciar el servidor
- No hay persistencia en base de datos
- Ideal para desarrollo y pruebas
- La migración a PostgreSQL/Redis está planificada

✅ **Listo para usar**:
- Domain layer completo con DDD
- Application layer con use cases
- REST API funcional
- Manejo de errores robusto
- Optimistic locking implementado
