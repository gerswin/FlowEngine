# Ejemplos de API FlowEngine

Esta carpeta contiene ejemplos de peticiones JSON para usar con la API de FlowEngine.

## Archivos Disponibles

### `create_workflow.json`
Ejemplo de creación de un workflow simple con estados y transiciones.

**Uso:**
```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/create_workflow.json
```

### `create_instance.json`
Ejemplo de creación de una instancia de workflow.

**Nota:** Reemplaza `REPLACE_WITH_ACTUAL_WORKFLOW_ID` con un ID real de workflow.

**Uso:**
```bash
# Obtener un workflow ID primero
WORKFLOW_ID=$(curl http://localhost:8080/api/v1/workflows | jq -r '.workflows[0].id')

# Crear instancia (reemplazando el ID en el archivo)
sed "s/REPLACE_WITH_ACTUAL_WORKFLOW_ID/$WORKFLOW_ID/" examples/create_instance.json | \
curl -X POST http://localhost:8080/api/v1/instances \
  -H "Content-Type: application/json" \
  -d @-
```

### `transition_instance.json`
Ejemplo de ejecución de una transición (aprobar documento).

**Uso:**
```bash
curl -X POST http://localhost:8080/api/v1/instances/INSTANCE_ID/transitions \
  -H "Content-Type: application/json" \
  -d @examples/transition_instance.json
```

## Script de Prueba Completo

```bash
#!/bin/bash

# 1. Crear workflow
echo "📋 Creando workflow..."
WORKFLOW_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/create_workflow.json)

WORKFLOW_ID=$(echo $WORKFLOW_RESPONSE | jq -r '.id')
echo "✅ Workflow creado: $WORKFLOW_ID"

# 2. Crear instancia
echo ""
echo "🎬 Creando instancia..."
INSTANCE_JSON=$(cat examples/create_instance.json | \
  sed "s/REPLACE_WITH_ACTUAL_WORKFLOW_ID/$WORKFLOW_ID/")

INSTANCE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/instances \
  -H "Content-Type: application/json" \
  -d "$INSTANCE_JSON")

INSTANCE_ID=$(echo $INSTANCE_RESPONSE | jq -r '.id')
echo "✅ Instancia creada: $INSTANCE_ID"
echo "   Estado inicial: $(echo $INSTANCE_RESPONSE | jq -r '.current_state')"

# 3. Ejecutar transición: submit
echo ""
echo "🔄 Ejecutando transición: submit..."
curl -s -X POST http://localhost:8080/api/v1/instances/$INSTANCE_ID/transitions \
  -H "Content-Type: application/json" \
  -d '{
    "event": "submit",
    "actor_id": "550e8400-e29b-41d4-a716-446655440000",
    "reason": "Enviando para revisión"
  }' | jq

# 4. Ejecutar transición: approve
echo ""
echo "✅ Ejecutando transición: approve..."
curl -s -X POST http://localhost:8080/api/v1/instances/$INSTANCE_ID/transitions \
  -H "Content-Type: application/json" \
  -d @examples/transition_instance.json | jq

# 5. Ver estado final
echo ""
echo "📊 Estado final de la instancia:"
curl -s http://localhost:8080/api/v1/instances/$INSTANCE_ID | jq '{
  id: .id,
  workflow: .workflow_name,
  current_state: .current_state,
  status: .status,
  transitions: .transition_count
}'
```

Guarda este script como `test_flow.sh`, dale permisos de ejecución y ejecútalo:

```bash
chmod +x test_flow.sh
./test_flow.sh
```
