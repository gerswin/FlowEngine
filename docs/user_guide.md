# FlowEngine - User Guide

## Table of Contents

1. [Getting Started](#getting-started)
2. [Core Concepts](#core-concepts)
3. [Creating Workflows](#creating-workflows)
4. [Managing Instances](#managing-instances)
5. [YAML Workflow Definition](#yaml-workflow-definition)
6. [API Reference](#api-reference)
7. [Best Practices](#best-practices)
8. [Troubleshooting](#troubleshooting)

---

## Getting Started

### Prerequisites

- Go 1.21+
- Docker & Docker Compose (optional, for full stack)
- PostgreSQL 16+ (production)
- Redis 7+ (optional, for caching)

### Quick Start

```bash
# Clone repository
git clone https://github.com/LaFabric-LinkTIC/FlowEngine.git
cd FlowEngine

# Run with in-memory storage (development)
make run

# Or with Docker (full stack)
docker-compose up
```

The API will be available at `http://localhost:8080`

### Verify Installation

```bash
curl http://localhost:8080/health
# Response: {"status":"ok","timestamp":"2024-..."}
```

---

## Core Concepts

### Workflow

A **Workflow** defines the blueprint for a process. It consists of:

- **States**: The possible positions in the workflow (e.g., `draft`, `review`, `approved`)
- **Events**: Actions that trigger transitions between states (e.g., `submit`, `approve`, `reject`)
- **Initial State**: The starting point for all instances

### Instance

An **Instance** is a running execution of a workflow. It:

- Tracks the current state
- Maintains data and variables
- Records transition history
- Has a status (RUNNING, PAUSED, COMPLETED, CANCELED, FAILED)

### Transition

A **Transition** is a state change triggered by an event. It includes:

- Source and destination states
- Actor who performed the transition
- Timestamp and metadata

### State Types

| Type | Description |
|------|-------------|
| Initial | Starting state (one per workflow) |
| Intermediate | Normal processing states |
| Final | Terminal states that complete the instance |

---

## Creating Workflows

### Method 1: JSON API

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "data": {
      "type": "workflow",
      "attributes": {
        "name": "Document Approval",
        "description": "Simple document approval process",
        "created_by": "550e8400-e29b-41d4-a716-446655440000",
        "initial_state": {
          "id": "draft",
          "name": "Draft",
          "is_final": false
        },
        "states": [
          {"id": "draft", "name": "Draft", "is_final": false},
          {"id": "review", "name": "Under Review", "is_final": false},
          {"id": "approved", "name": "Approved", "is_final": true},
          {"id": "rejected", "name": "Rejected", "is_final": true}
        ],
        "events": [
          {"name": "submit", "sources": ["draft"], "destination": "review"},
          {"name": "approve", "sources": ["review"], "destination": "approved"},
          {"name": "reject", "sources": ["review"], "destination": "rejected"},
          {"name": "revise", "sources": ["review"], "destination": "draft"}
        ]
      }
    }
  }'
```

### Method 2: YAML File

```bash
curl -X POST http://localhost:8080/api/v1/workflows/from-yaml \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@workflow.yaml" \
  -F "created_by=550e8400-e29b-41d4-a716-446655440000"
```

---

## Managing Instances

### Create Instance

```bash
curl -X POST http://localhost:8080/api/v1/instances \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "data": {
      "type": "instance",
      "attributes": {
        "workflow_id": "WORKFLOW_UUID",
        "started_by": "USER_UUID",
        "data": {
          "document_type": "invoice",
          "amount": 1500.00
        },
        "variables": {
          "department": "finance",
          "priority": "high"
        }
      }
    }
  }'
```

### Perform Transition

```bash
curl -X POST http://localhost:8080/api/v1/instances/INSTANCE_UUID/transitions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "data": {
      "type": "transition",
      "attributes": {
        "event": "submit",
        "actor_id": "USER_UUID",
        "reason": "Document ready for review",
        "feedback": "Please review by EOD",
        "data": {
          "submitted_at": "2024-01-15T10:30:00Z"
        }
      }
    }
  }'
```

### Get Instance Status

```bash
curl http://localhost:8080/api/v1/instances/INSTANCE_UUID \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### List Instances

```bash
# All instances
curl "http://localhost:8080/api/v1/instances" \
  -H "Authorization: Bearer YOUR_TOKEN"

# Filter by workflow
curl "http://localhost:8080/api/v1/instances?filter[workflow_id]=WORKFLOW_UUID" \
  -H "Authorization: Bearer YOUR_TOKEN"

# With pagination
curl "http://localhost:8080/api/v1/instances?page[number]=1&page[size]=20" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Clone Instance

For parallel reviews or multi-department approvals:

```bash
curl -X POST http://localhost:8080/api/v1/instances/INSTANCE_UUID/clone \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "data": {
      "type": "clone-request",
      "attributes": {
        "assignees": [
          {"user_id": "USER_UUID_1", "office_id": "office-001"},
          {"user_id": "USER_UUID_2", "office_id": "office-002"}
        ],
        "consolidator_id": "CONSOLIDATOR_UUID",
        "reason": "Multi-department review required",
        "timeout_duration": "7d"
      }
    }
  }'
```

---

## YAML Workflow Definition

### Basic Structure

```yaml
name: "Document Processing"
description: "Standard document processing workflow"
initial_state: draft

states:
  - id: draft
    name: "Draft"
    description: "Document is being prepared"

  - id: review
    name: "Under Review"
    description: "Document is being reviewed"
    timeout: "7d"
    on_timeout: escalate

  - id: approved
    name: "Approved"
    is_final: true

  - id: rejected
    name: "Rejected"
    is_final: true

events:
  - name: submit
    sources: [draft]
    destination: review

  - name: approve
    sources: [review]
    destination: approved

  - name: reject
    sources: [review]
    destination: rejected

  - name: revise
    sources: [review]
    destination: draft

  - name: escalate
    sources: [review]
    destination: review
```

### Advanced Features

#### Timeouts and Escalations

```yaml
states:
  - id: pending_approval
    name: "Pending Approval"
    timeout: "3d"
    on_timeout: auto_escalate
```

#### Actions

```yaml
events:
  - name: approve
    sources: [review]
    destination: approved
    actions:
      - type: notify
        params:
          channel: email
          template: approval_notification
      - type: set_metadata
        params:
          approved_at: "{{now}}"
```

#### Guards (Conditions)

```yaml
events:
  - name: approve
    sources: [review]
    destination: approved
    guards:
      - type: has_role
        params:
          role: approver
      - type: field_equals
        params:
          field: amount
          operator: lt
          value: 10000
```

### MinTrabajo Workflow Example

```yaml
name: "Gestion Tramites MinTrabajo"
description: "Workflow para tramites del Ministerio de Trabajo"
initial_state: radicacion

states:
  - id: radicacion
    name: "Radicacion"
    description: "Tramite radicado"

  - id: asignacion
    name: "Asignacion"
    description: "Asignado a funcionario"
    timeout: "1d"
    on_timeout: escalar_asignacion

  - id: revision
    name: "En Revision"
    timeout: "15d"
    on_timeout: alerta_vencimiento

  - id: respuesta
    name: "Respuesta Generada"

  - id: finalizado
    name: "Finalizado"
    is_final: true

events:
  - name: radicar
    sources: [radicacion]
    destination: asignacion
    actions:
      - type: generate_document_id
      - type: notify

  - name: asignar
    sources: [asignacion]
    destination: revision
    actions:
      - type: assign_to_user

  - name: responder
    sources: [revision]
    destination: respuesta

  - name: finalizar
    sources: [respuesta]
    destination: finalizado
```

---

## API Reference

### Authentication

All endpoints (except `/health`) require JWT authentication:

```bash
# Get development token
curl -X POST http://localhost:8080/api/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{"user_id": "YOUR_UUID", "roles": ["admin"]}'

# Use token in requests
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/v1/workflows
```

### Response Format (JSON:API)

All responses follow JSON:API specification:

```json
{
  "data": {
    "type": "workflow",
    "id": "uuid",
    "attributes": { ... },
    "links": {
      "self": "/api/v1/workflows/uuid"
    }
  },
  "meta": {
    "total": 100,
    "page": { "number": 1, "size": 20 }
  },
  "links": {
    "self": "/api/v1/workflows?page[number]=1",
    "next": "/api/v1/workflows?page[number]=2"
  }
}
```

### Error Response

```json
{
  "errors": [
    {
      "status": "409",
      "code": "TRANSITION_ERROR",
      "title": "Invalid transition",
      "detail": "Cannot transition from 'approved' with event 'submit'"
    }
  ]
}
```

### Endpoints Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/api/v1/auth/token` | Get JWT token (dev) |
| POST | `/api/v1/workflows` | Create workflow |
| POST | `/api/v1/workflows/from-yaml` | Create from YAML |
| GET | `/api/v1/workflows` | List workflows |
| GET | `/api/v1/workflows/:id` | Get workflow |
| POST | `/api/v1/instances` | Create instance |
| GET | `/api/v1/instances` | List instances |
| GET | `/api/v1/instances/:id` | Get instance |
| POST | `/api/v1/instances/:id/transitions` | Transition |
| POST | `/api/v1/instances/:id/clone` | Clone instance |

---

## Best Practices

### Workflow Design

1. **Use lowercase IDs**: State and event IDs should be lowercase with underscores (`pending_review`, not `PendingReview`)

2. **Always have final states**: Mark terminal states as `is_final: true`

3. **Include descriptions**: Document each state's purpose

4. **Plan for errors**: Include rejected/failed states

5. **Use timeouts wisely**: Configure timeouts for states that need deadlines

### Instance Management

1. **Set initial data**: Provide relevant data when creating instances

2. **Track actors**: Always specify `actor_id` in transitions

3. **Add context**: Use `reason` and `feedback` fields for audit trails

4. **Handle concurrency**: The system uses optimistic locking, handle version conflicts

### Security

1. **Protect tokens**: Never expose JWT secrets

2. **Use HTTPS**: Always use TLS in production

3. **Validate UUIDs**: Ensure IDs are valid UUIDs

4. **Audit transitions**: Log all state changes

---

## Troubleshooting

### Common Errors

#### "Invalid transition"

```json
{
  "errors": [{
    "code": "TRANSITION_ERROR",
    "detail": "Cannot transition from 'approved' to 'review' with event 'submit'"
  }]
}
```

**Cause**: The event is not valid for the current state.

**Solution**: Check the workflow definition to verify allowed transitions from the current state.

#### "Instance not found"

**Cause**: Invalid instance ID or instance was deleted.

**Solution**: Verify the UUID is correct using `GET /api/v1/instances`.

#### "Workflow validation failed"

**Cause**: YAML has invalid structure or references non-existent states.

**Solution**: Check that all event sources and destinations reference defined states.

#### "Version conflict"

**Cause**: Concurrent modification of the same instance.

**Solution**: Retry the operation. The system uses optimistic locking.

### Debug Mode

Enable debug logging:

```bash
LOG_LEVEL=debug make run
```

Or in Docker:

```yaml
environment:
  - LOG_LEVEL=debug
  - GIN_MODE=debug
```

### Health Check

```bash
# Check API health
curl http://localhost:8080/health

# Check database connection (if configured)
docker-compose exec postgres pg_isready -U flowuser -d flowengine

# Check Redis
docker-compose exec redis redis-cli ping
```

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | API server port | `8080` |
| `LOG_LEVEL` | Log verbosity | `info` |
| `GIN_MODE` | Gin mode (debug/release) | `debug` |
| `POSTGRES_HOST` | PostgreSQL host | - |
| `POSTGRES_PORT` | PostgreSQL port | `5432` |
| `POSTGRES_USER` | Database user | `postgres` |
| `POSTGRES_PASSWORD` | Database password | - |
| `POSTGRES_DB` | Database name | `flowengine` |
| `REDIS_ADDR` | Redis address | - |
| `JWT_SECRET` | JWT signing key | - |
| `JWT_EXPIRATION` | Token expiration | `24h` |

---

## Support

- **Documentation**: [docs/](./docs/)
- **Issues**: [GitHub Issues](https://github.com/LaFabric-LinkTIC/FlowEngine/issues)
- **API Spec**: [docs/openapi.yaml](./docs/openapi.yaml)
