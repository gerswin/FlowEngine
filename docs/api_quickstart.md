# FlowEngine REST API - Quick Start

## 1. Iniciar el servidor

```bash
go run cmd/api/main.go
# o con Make:
make run
```

Servidor disponible en `http://localhost:8080` (in-memory, sin dependencias externas).

## 2. Obtener token de autenticacion

Todos los endpoints `/api/v1/*` requieren JWT. El endpoint de desarrollo genera un token de admin:

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/token | jq -r '.token')
```

Usar en todas las requests:
```bash
-H "Authorization: Bearer $TOKEN"
-H "Content-Type: application/vnd.api+json"
```

## 3. Crear un Workflow

**POST /api/v1/workflows**

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/vnd.api+json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "data": {
      "type": "workflow",
      "attributes": {
        "name": "Proceso de Aprobacion",
        "description": "Workflow con 4 estados y validaciones",
        "created_by": "550e8400-e29b-41d4-a716-446655440000",
        "initial_state": {
          "id": "draft",
          "name": "Borrador",
          "is_final": false
        },
        "states": [
          {"id": "draft", "name": "Borrador", "is_final": false},
          {"id": "review", "name": "En Revision", "is_final": false},
          {"id": "approved", "name": "Aprobado", "is_final": true},
          {"id": "rejected", "name": "Rechazado", "is_final": true}
        ],
        "events": [
          {
            "name": "submit",
            "sources": ["draft"],
            "destination": "review",
            "required_data": ["title"],
            "guards": [
              {"type": "field_not_empty", "params": {"field": "title"}}
            ],
            "actions": [
              {"type": "set_metadata", "params": {"key": "submitted_at", "value": "$now"}}
            ]
          },
          {
            "name": "approve",
            "sources": ["review"],
            "destination": "approved",
            "required_data": ["approved_by"],
            "actions": [
              {"type": "mark_as_approved", "params": {}}
            ]
          },
          {
            "name": "reject",
            "sources": ["review"],
            "destination": "rejected",
            "actions": [
              {"type": "increment_rejection_count", "params": {}}
            ]
          }
        ]
      }
    }
  }'
```

**Response (201):**
```json
{
  "data": {
    "type": "workflow",
    "id": "49a5d892-aa26-45f1-bce4-c9151c03657d",
    "attributes": {
      "name": "Proceso de Aprobacion",
      "version": "1.0.0",
      "states": [...],
      "events": [...]
    }
  },
  "jsonapi": {"version": "1.0"},
  "links": {"self": "/api/v1/workflows/49a5d892-..."}
}
```

## 4. Crear una Instancia

**POST /api/v1/instances**

```bash
curl -X POST http://localhost:8080/api/v1/instances \
  -H "Content-Type: application/vnd.api+json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "data": {
      "type": "instance",
      "attributes": {
        "workflow_id": "<WORKFLOW_ID>",
        "started_by": "550e8400-e29b-41d4-a716-446655440000",
        "data": {
          "title": "Propuesta Q1 2026",
          "department": "IT",
          "amount": 50000
        },
        "variables": {
          "priority": "high",
          "sla_hours": 48
        }
      }
    }
  }'
```

**Response (201):**
```json
{
  "data": {
    "type": "instance",
    "id": "1821a63d-471e-40dd-9582-bb474875fd04",
    "attributes": {
      "workflow_id": "49a5d892-...",
      "current_state": "draft",
      "status": "RUNNING",
      "version": "v3"
    }
  }
}
```

## 5. Ejecutar Transicion

**POST /api/v1/instances/:id/transitions**

```bash
curl -X POST http://localhost:8080/api/v1/instances/<INSTANCE_ID>/transitions \
  -H "Content-Type: application/vnd.api+json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "data": {
      "type": "transition",
      "attributes": {
        "event": "submit",
        "actor_id": "550e8400-e29b-41d4-a716-446655440000",
        "reason": "Listo para revision",
        "data": {
          "submitted_from": "web_portal"
        }
      }
    }
  }'
```

**Response (200):**
```json
{
  "data": {
    "type": "instance",
    "id": "1821a63d-...",
    "attributes": {
      "current_state": "review",
      "previous_state": "draft",
      "version": "v4"
    }
  }
}
```

### Errores de validacion

Si faltan campos obligatorios (`required_data`) o un guard falla:

```json
{
  "errors": [{
    "status": "400",
    "code": "INVALID_INPUT",
    "title": "missing required data for event 'submit': title",
    "detail": "[INVALID_INPUT] missing required data for event 'submit': title"
  }]
}
```

## 6. Consultar Historial

**GET /api/v1/instances/:id/history**

```bash
curl http://localhost:8080/api/v1/instances/<INSTANCE_ID>/history \
  -H "Authorization: Bearer $TOKEN"
```

**Response (200):**
```json
{
  "data": [
    {
      "type": "transition",
      "id": "c3d4e5f6-...",
      "attributes": {
        "from": "draft",
        "to": "review",
        "event": "submit",
        "actor": "550e8400-...",
        "timestamp": "2026-03-19T15:58:01Z",
        "reason": "Listo para revision",
        "feedback": "",
        "metadata": {}
      }
    }
  ]
}
```

## 7. Listar con Paginacion

**GET /api/v1/instances** y **GET /api/v1/workflows** soportan paginacion:

```bash
# Pagina 1, 10 items
curl "http://localhost:8080/api/v1/instances?page[number]=1&page[size]=10" \
  -H "Authorization: Bearer $TOKEN"

# Filtrar por workflow
curl "http://localhost:8080/api/v1/instances?filter[workflow_id]=<ID>" \
  -H "Authorization: Bearer $TOKEN"
```

**Response incluye meta de paginacion:**
```json
{
  "data": [...],
  "meta": {
    "total": 42,
    "page": {
      "number": 1,
      "size": 10,
      "total": 5
    }
  }
}
```

## 8. Manejo de Errores

Todos los errores usan formato JSON:API:

| HTTP | Code | Cuando |
|------|------|--------|
| 400 | INVALID_INPUT | Datos invalidos, guards fallidos, campos faltantes |
| 401 | UNAUTHORIZED | Token faltante o invalido |
| 403 | FORBIDDEN | Sin permisos (guard has_role) |
| 404 | NOT_FOUND | Recurso no existe |
| 409 | CONFLICT | Version mismatch |
| 409 | INVALID_STATE | Transicion invalida para el estado actual |

## Guards y Actions

Los eventos pueden definir **guards** (condiciones pre-transicion) y **actions** (efectos post-transicion):

```json
{
  "name": "approve",
  "sources": ["review"],
  "destination": "approved",
  "required_data": ["approved_by"],
  "guards": [
    {"type": "field_exists", "params": {"field": "approved_by"}},
    {"type": "field_equals", "params": {"field": "currency", "value": "COP"}}
  ],
  "actions": [
    {"type": "mark_as_approved", "params": {}},
    {"type": "set_metadata", "params": {"key": "status", "value": "approved"}}
  ]
}
```

**Orden de ejecucion:**
1. Se aplican los datos de la transicion (`data` del request)
2. Se evaluan los guards (si alguno falla -> 400)
3. Se validan los `required_data`
4. Se ejecuta la transicion (cambio de estado)
5. Se ejecutan las actions
6. Se persiste y se despachan eventos

## Testing con Bruno

La coleccion Bruno esta en `bruno/FlowEngine/`. Para usarla:

1. Abrir Bruno e importar la carpeta `bruno/FlowEngine/`
2. Seleccionar el environment **Local** (top-right)
3. Ejecutar **Auth > Get Token** primero
4. Ejecutar los requests en orden

Colecciones disponibles:
- **Auth** — Obtener token
- **Workflows** — CRUD de workflows
- **Instances** — CRUD de instancias + transiciones
- **Facturacion** — Flujo completo de facturacion con guards y actions
