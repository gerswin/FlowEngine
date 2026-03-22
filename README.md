# FlowEngine

Motor de workflows generico, escalable y cloud-native basado en maquinas de estados finitos (FSM). Arquitectura hexagonal (Clean Architecture) con Domain-Driven Design.

## Caracteristicas

- **Workflows Configurables**: Define workflows via JSON:API o YAML con estados, eventos, guards y actions
- **Guards y Actions**: Motor de reglas que valida condiciones antes de transicionar y ejecuta efectos despues
- **Required Data**: Campos obligatorios por transicion, validados automaticamente
- **Arquitectura Hexagonal**: Separacion clara entre dominio, aplicacion e infraestructura
- **Persistencia Hibrida**: PostgreSQL (persistencia) + Redis (cache) + In-Memory (desarrollo)
- **Paginacion DB-level**: LIMIT/OFFSET con COUNT(*) OVER() en una sola query
- **JSON:API Compliant**: Todas las respuestas siguen la especificacion JSON:API 1.0
- **JWT Authentication**: Autenticacion por Bearer token en todos los endpoints protegidos
- **Domain Events**: Sistema de eventos con MultiDispatcher (InMemory + Log + Webhook)
- **Webhook Delivery**: Entrega asincrona con HMAC signing y reintentos exponenciales
- **Scheduler con Retry**: Timers con backoff exponencial y tracking de reintentos
- **Optimistic Locking**: Control de concurrencia por versionado
- **Subprocesos**: Workflows anidados con relacion padre-hijo

## Stack Tecnologico

- **Go 1.24+**
- **Gin** — Framework HTTP
- **PostgreSQL 15+** — Persistencia
- **Redis 7+** — Cache distribuido
- **Docker & Docker Compose** — Contenedorizacion

## Inicio Rapido

### Prerrequisitos

- Go 1.24+
- (Opcional) PostgreSQL, Redis para modo produccion

### Ejecutar en modo desarrollo

```bash
# Clonar e instalar
git clone https://github.com/LaFabric-LinkTIC/FlowEngine.git
cd FlowEngine
go mod download

# Ejecutar tests
go test ./...

# Iniciar servidor (in-memory, sin dependencias externas)
go run cmd/api/main.go
```

El servidor inicia en `http://localhost:8080` con persistencia in-memory.

### Ejecutar con Docker (PostgreSQL + Redis)

```bash
docker-compose up -d
```

## API

Todos los endpoints usan formato **JSON:API** y requieren autenticacion JWT.

### Obtener Token

```bash
# Endpoint de desarrollo (genera token de admin)
curl -X POST http://localhost:8080/api/v1/auth/token
# Response: {"token": "eyJhbG..."}
```

### Endpoints

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/health` | Health check (publico) |
| POST | `/api/v1/auth/token` | Obtener JWT token (desarrollo) |
| POST | `/api/v1/workflows` | Crear workflow |
| POST | `/api/v1/workflows/from-yaml` | Crear workflow desde YAML |
| GET | `/api/v1/workflows` | Listar workflows (paginado) |
| GET | `/api/v1/workflows/:id` | Obtener workflow por ID |
| POST | `/api/v1/instances` | Crear instancia |
| GET | `/api/v1/instances` | Listar instancias (paginado) |
| GET | `/api/v1/instances/:id` | Obtener instancia |
| GET | `/api/v1/instances/:id/history` | Historial de transiciones |
| POST | `/api/v1/instances/:id/transitions` | Ejecutar transicion |
| POST | `/api/v1/instances/:id/clone` | Clonar instancia |

### Ejemplo: Flujo completo

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/token | jq -r '.token')
AUTH="Authorization: Bearer $TOKEN"

# 1. Crear workflow con guards y required_data
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/vnd.api+json" -H "$AUTH" \
  -d '{
    "data": {
      "type": "workflow",
      "attributes": {
        "name": "Aprobacion",
        "created_by": "550e8400-e29b-41d4-a716-446655440000",
        "initial_state": {"id": "draft", "name": "Borrador", "is_final": false},
        "states": [
          {"id": "draft", "name": "Borrador", "is_final": false},
          {"id": "review", "name": "En Revision", "is_final": false},
          {"id": "approved", "name": "Aprobado", "is_final": true}
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
            "actions": [
              {"type": "mark_as_approved", "params": {}}
            ]
          }
        ]
      }
    }
  }'

# 2. Crear instancia
curl -X POST http://localhost:8080/api/v1/instances \
  -H "Content-Type: application/vnd.api+json" -H "$AUTH" \
  -d '{
    "data": {
      "type": "instance",
      "attributes": {
        "workflow_id": "<WORKFLOW_ID>",
        "started_by": "550e8400-e29b-41d4-a716-446655440000",
        "data": {"title": "Mi documento"}
      }
    }
  }'

# 3. Transicionar
curl -X POST http://localhost:8080/api/v1/instances/<INSTANCE_ID>/transitions \
  -H "Content-Type: application/vnd.api+json" -H "$AUTH" \
  -d '{
    "data": {
      "type": "transition",
      "attributes": {
        "event": "submit",
        "actor_id": "550e8400-e29b-41d4-a716-446655440000",
        "reason": "Listo para revision"
      }
    }
  }'

# 4. Ver historial
curl http://localhost:8080/api/v1/instances/<INSTANCE_ID>/history -H "$AUTH"
```

### Paginacion

```bash
# Pagina 2, 10 items por pagina
curl "http://localhost:8080/api/v1/instances?page[number]=2&page[size]=10" -H "$AUTH"

# Filtrar por workflow
curl "http://localhost:8080/api/v1/instances?filter[workflow_id]=<ID>" -H "$AUTH"
```

### Guards disponibles

| Guard | Params | Descripcion |
|-------|--------|-------------|
| `field_exists` | `field` | Campo existe en data |
| `field_not_empty` | `field` | Campo existe y no esta vacio |
| `field_equals` | `field`, `value` | Campo tiene valor especifico |
| `field_matches` | `field`, `pattern` | Campo coincide con regex |
| `has_role` | `role` | Actor tiene el rol |
| `has_any_role` | `roles` | Actor tiene al menos un rol |
| `validate_required_fields` | `fields` | Multiples campos existen |
| `is_assigned_to_actor` | — | Instancia asignada al actor |
| `is_not_assigned` | — | Instancia no asignada |
| `substate_equals` | `substate` | Sub-estado actual coincide |
| `instance_age_less_than` | `duration` | Instancia creada hace menos de X |
| `instance_age_more_than` | `duration` | Instancia creada hace mas de X |

### Actions disponibles

| Action | Params | Descripcion |
|--------|--------|-------------|
| `set_metadata` | `key`, `value` | Setea campo en data (`$now` para timestamp) |
| `increment_field` | `field` | Incrementa campo numerico |
| `assign_to_user` | `user_id` (opcional) | Asigna instancia a usuario |
| `mark_as_approved` | — | Setea approved, approved_at, approved_by |
| `mark_as_completed` | — | Setea completed, completed_at |
| `increment_rejection_count` | — | Incrementa rejection_count |
| `add_feedback_to_instance` | `feedback` | Guarda ultimo feedback |
| `update_document_type` | `document_type` | Actualiza tipo de documento |
| `log_reclassification` | — | Registra reclasificacion |
| `emit_event` | `event_name` | Emite evento custom |

### Errores (JSON:API)

```json
{
  "errors": [{
    "status": "400",
    "code": "INVALID_INPUT",
    "title": "guard failed: field 'currency' expected 'COP', got 'USD'",
    "detail": "[INVALID_INPUT] guard failed: field 'currency' expected 'COP', got 'USD'"
  }],
  "jsonapi": {"version": "1.0"}
}
```

| HTTP | Code | Cuando |
|------|------|--------|
| 400 | INVALID_INPUT | Datos invalidos, guards fallidos, campos faltantes |
| 401 | UNAUTHORIZED | Token faltante o invalido |
| 403 | FORBIDDEN | Sin permisos (guard has_role) |
| 404 | NOT_FOUND | Recurso no existe |
| 409 | CONFLICT / INVALID_STATE | Version mismatch, transicion invalida |
| 500 | INTERNAL | Error interno |

## Estructura del Proyecto

```
FlowEngine/
├── cmd/
│   ├── api/                  # Servidor REST API
│   ├── demo/                 # Demo interactiva
│   └── emulator/             # Emulador de workflows
├── internal/
│   ├── domain/               # Logica de negocio pura (DDD)
│   │   ├── workflow/         # Aggregate: Workflow, State, Event, Guards, Actions
│   │   ├── instance/         # Aggregate: Instance, Transition, Engine
│   │   ├── event/            # Domain Events + Dispatchers
│   │   ├── timer/            # Timers con retry policy
│   │   └── shared/           # Value Objects: ID, Timestamp, Pagination, Errors
│   ├── application/          # Use Cases
│   │   ├── workflow/         # Create, Get, List workflows
│   │   └── instance/         # Create, Get, List, Transition, Clone instances
│   └── infrastructure/       # Adaptadores
│       ├── http/             # Handlers, Router, Middleware (auth, CORS, logger)
│       ├── persistence/      # PostgreSQL + In-Memory repositories
│       ├── messaging/        # MultiDispatcher, WebhookDispatcher, LogDispatcher
│       ├── parser/yaml/      # Parser YAML para workflows
│       ├── cache/            # Redis cache
│       ├── security/         # JWT token service
│       └── scheduler/        # Timer worker con retry
├── pkg/
│   ├── jsonapi/              # JSON:API helpers, pagination
│   └── logger/               # Structured logging (slog/JSON)
├── bruno/                    # Coleccion Bruno para testing API
├── postman/                  # Coleccion Postman
├── examples/                 # JSON de ejemplo
├── migrations/               # SQL migrations
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## Testing

```bash
# Unit tests
go test ./...

# Con cobertura
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Solo handler tests
go test ./internal/infrastructure/http/handler/... -v
```

### Bruno (API Testing)

Abrir la coleccion en Bruno desde `bruno/FlowEngine/`:
1. Seleccionar environment **Local**
2. Ejecutar **Auth > Get Token**
3. Ejecutar requests en orden

## Variables de Entorno

| Variable | Default | Descripcion |
|----------|---------|-------------|
| `PORT` | `8080` | Puerto del servidor |
| `POSTGRES_HOST` | — | Host de PostgreSQL (sin esto usa in-memory) |
| `POSTGRES_PORT` | `5432` | Puerto de PostgreSQL |
| `POSTGRES_USER` | `postgres` | Usuario |
| `POSTGRES_PASSWORD` | `postgres` | Password |
| `POSTGRES_DB` | `flowengine` | Base de datos |
| `REDIS_ADDR` | — | Direccion de Redis (sin esto desactiva cache) |
| `JWT_SECRET` | dev default | Secret para firmar tokens |
| `GIN_MODE` | `debug` | `release` para produccion |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

## Documentacion

- [API Quick Start](docs/api_quickstart.md)
- [Requisitos](requirements.md)
- [Diseno Arquitectonico](design.md)
