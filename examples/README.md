# Ejemplos de API FlowEngine

Esta carpeta contiene ejemplos de peticiones JSON para usar con la API de FlowEngine.

> **Nota**: Todos los endpoints requieren JWT auth. Los ejemplos asumen que ya tienes un token.

## Obtener Token

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/token | jq -r '.token')
```

## Archivos Disponibles

### `create_workflow.json`
Ejemplo de creacion de un workflow simple con estados y transiciones.

**Uso:**
```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/vnd.api+json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "data": {
      "type": "workflow",
      "attributes": '"$(cat examples/create_workflow.json)"'
    }
  }'
```

### `create_instance.json`
Ejemplo de creacion de una instancia de workflow.

**Nota:** Reemplaza `REPLACE_WITH_ACTUAL_WORKFLOW_ID` con un ID real de workflow.

### `transition_instance.json`
Ejemplo de ejecucion de una transicion (aprobar documento).

## Script de Prueba Completo

```bash
#!/bin/bash
# Requiere: curl, jq

BASE=http://localhost:8080

# 1. Obtener token
TOKEN=$(curl -s -X POST $BASE/api/v1/auth/token | jq -r '.token')
AUTH="Authorization: Bearer $TOKEN"
CT="Content-Type: application/vnd.api+json"
echo "Token obtenido"

# 2. Crear workflow
echo ""
echo "Creando workflow..."
WF=$(curl -s -X POST $BASE/api/v1/workflows \
  -H "$CT" -H "$AUTH" \
  -d '{
    "data": {
      "type": "workflow",
      "attributes": {
        "name": "Proceso de Aprobacion",
        "created_by": "550e8400-e29b-41d4-a716-446655440000",
        "initial_state": {"id": "draft", "name": "Borrador", "is_final": false},
        "states": [
          {"id": "draft", "name": "Borrador", "is_final": false},
          {"id": "review", "name": "En Revision", "is_final": false},
          {"id": "approved", "name": "Aprobado", "is_final": true}
        ],
        "events": [
          {"name": "submit", "sources": ["draft"], "destination": "review"},
          {"name": "approve", "sources": ["review"], "destination": "approved"}
        ]
      }
    }
  }')
WF_ID=$(echo $WF | jq -r '.data.id')
echo "Workflow creado: $WF_ID"

# 3. Crear instancia
echo ""
echo "Creando instancia..."
INST=$(curl -s -X POST $BASE/api/v1/instances \
  -H "$CT" -H "$AUTH" \
  -d "{
    \"data\": {
      \"type\": \"instance\",
      \"attributes\": {
        \"workflow_id\": \"$WF_ID\",
        \"started_by\": \"550e8400-e29b-41d4-a716-446655440000\",
        \"data\": {\"title\": \"Propuesta Q1\"}
      }
    }
  }")
INST_ID=$(echo $INST | jq -r '.data.id')
echo "Instancia creada: $INST_ID"
echo "Estado: $(echo $INST | jq -r '.data.attributes.current_state')"

# 4. Transicion: submit
echo ""
echo "Transicion: submit..."
curl -s -X POST $BASE/api/v1/instances/$INST_ID/transitions \
  -H "$CT" -H "$AUTH" \
  -d '{
    "data": {
      "type": "transition",
      "attributes": {
        "event": "submit",
        "actor_id": "550e8400-e29b-41d4-a716-446655440000",
        "reason": "Listo para revision"
      }
    }
  }' | jq '{state: .data.attributes.current_state}'

# 5. Transicion: approve
echo ""
echo "Transicion: approve..."
curl -s -X POST $BASE/api/v1/instances/$INST_ID/transitions \
  -H "$CT" -H "$AUTH" \
  -d '{
    "data": {
      "type": "transition",
      "attributes": {
        "event": "approve",
        "actor_id": "660f9511-f3ac-52e5-b827-557766551111",
        "reason": "Aprobado"
      }
    }
  }' | jq '{state: .data.attributes.current_state}'

# 6. Ver historial
echo ""
echo "Historial de transiciones:"
curl -s $BASE/api/v1/instances/$INST_ID/history \
  -H "$AUTH" | jq '[.data[] | {event: .attributes.event, from: .attributes.from, to: .attributes.to}]'

# 7. Estado final
echo ""
echo "Estado final:"
curl -s $BASE/api/v1/instances/$INST_ID \
  -H "$AUTH" | jq '{state: .data.attributes.current_state, status: .data.attributes.status}'
```
