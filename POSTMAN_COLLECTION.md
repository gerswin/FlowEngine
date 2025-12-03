# 🚀 FlowEngine API - Colección de Postman

## ✨ Características de la Colección

### 📦 Contenido Completo
- **20+ Requests** organizados por categorías
- **Auto-guardado de IDs** - Los IDs se guardan automáticamente en variables
- **Scripts de validación** - Verificación automática de respuestas
- **Ejemplos completos** - Flujos end-to-end paso a paso
- **Environment configurado** - Variables pre-configuradas para desarrollo local

---

## 📂 Estructura de la Colección

```
FlowEngine API
│
├── 🏥 Health Check
│   └── Health Check
│
├── 📋 Workflows
│   ├── Create Workflow - Simple Approval
│   ├── Create Workflow - Complex Purchase Order
│   ├── List All Workflows
│   └── Get Workflow by ID
│
├── 🎬 Instances
│   ├── Create Instance
│   ├── Create Instance - Purchase Order
│   ├── List All Instances
│   ├── List Instances by Workflow
│   └── Get Instance by ID
│
├── 🔄 Transitions
│   ├── Submit for Review
│   ├── Approve
│   ├── Reject
│   ├── Manager Approve (PO)
│   └── Finance Approve (PO)
│
└── 🎯 Complete Workflow Examples
    └── Example 1: Full Approval Flow
        ├── 1. Create Workflow
        ├── 2. Create Instance
        ├── 3. Submit for Review
        ├── 4. Approve
        └── 5. Get Final State
```

---

## 🎯 Casos de Uso Incluidos

### 1️⃣ Simple Approval Workflow

**Flujo**: Draft → Review → Approved/Rejected

**Estados**:
- `draft` - Borrador inicial
- `review` - En revisión
- `approved` - Aprobado (final)
- `rejected` - Rechazado (final)

**Eventos**:
- `submit` - Enviar para revisión
- `approve` - Aprobar documento
- `reject` - Rechazar documento

**Ejemplo de uso**:
```
1. Crear workflow de aprobación
2. Crear instancia con datos del documento
3. Submit → Documento pasa a revisión
4. Approve → Documento aprobado
```

---

### 2️⃣ Purchase Order Workflow

**Flujo**: Draft → Manager Review → Finance Review → Approved

**Estados**:
- `draft` - Orden en borrador
- `manager_review` - Revisión de gerente
- `finance_review` - Revisión de finanzas
- `approved` - Aprobado y listo para compra (final)
- `rejected` - Rechazado (final)
- `cancelled` - Cancelado por solicitante (final)

**Eventos**:
- `submit` - Enviar para aprobación
- `manager_approve` - Aprobación de gerente
- `manager_reject` - Rechazo de gerente
- `finance_approve` - Aprobación final de finanzas
- `finance_reject` - Rechazo de finanzas
- `cancel` - Cancelar orden

**Ejemplo de uso**:
```
1. Crear workflow de PO
2. Crear instancia con detalles de compra ($25,000)
3. Submit → Manager Review
4. Manager Approve → Finance Review
5. Finance Approve → Approved (listo para procurement)
```

---

## 🔧 Variables del Environment

| Variable | Descripción | Valor por Defecto | Auto-guardado |
|----------|-------------|-------------------|---------------|
| `base_url` | URL base del API | `http://localhost:8080` | No |
| `workflow_id` | ID del workflow activo | _(vacío)_ | ✅ Sí |
| `instance_id` | ID de la instancia activa | _(vacío)_ | ✅ Sí |
| `actor_id` | ID del actor/usuario | `550e8400-...` | No |

### 🤖 Auto-guardado Inteligente

Los requests de **CREATE** incluyen scripts que automáticamente guardan los IDs generados:

```javascript
// Al crear un workflow
✅ Workflow created: a1b2c3d4-e5f6-7890-abcd-ef1234567890

// Al crear una instancia
✅ Instance created: b2c3d4e5-f6a7-8901-bcde-f12345678901
```

Esto te permite ejecutar requests en secuencia **sin copiar/pegar IDs manualmente**.

---

## 📝 Ejemplos de Requests

### Crear Workflow

```json
POST /api/v1/workflows
{
  "name": "Proceso de Aprobación",
  "description": "Workflow simple de aprobación",
  "created_by": "{{actor_id}}",
  "initial_state": {
    "id": "draft",
    "name": "Borrador",
    "is_final": false
  },
  "states": [...],
  "events": [...]
}
```

**Response**:
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "name": "Proceso de Aprobación",
  "version": "1.0.0"
}
```

---

### Crear Instancia

```json
POST /api/v1/instances
{
  "workflow_id": "{{workflow_id}}",  // ✅ Auto-completado
  "started_by": "{{actor_id}}",
  "data": {
    "title": "Propuesta de Proyecto Q1 2025",
    "requester": "Juan Pérez",
    "amount": 50000
  },
  "variables": {
    "priority": "high",
    "sla_hours": 48
  }
}
```

**Response**:
```json
{
  "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
  "workflow_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "current_state": "draft",
  "status": "running",
  "version": "1"
}
```

---

### Ejecutar Transición

```json
POST /api/v1/instances/{{instance_id}}/transitions
{
  "event": "approve",
  "actor_id": "660f9511-f3ac-52e5-b827-557766551111",
  "reason": "Propuesta aprobada por el comité",
  "feedback": "Excelente propuesta, adelante",
  "metadata": {
    "reviewer": "María García",
    "review_score": 95
  },
  "data": {
    "approval_date": "2025-01-19",
    "approved_by": "María García"
  }
}
```

**Response**:
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

---

## 🎬 Cómo Empezar

### 1. Importar en Postman

1. Abre Postman
2. Click en **Import**
3. Arrastra los archivos:
   - `postman/FlowEngine_API.postman_collection.json`
   - `postman/FlowEngine_Environment.postman_environment.json`
4. Click en **Import**

### 2. Configurar Environment

1. Selecciona "FlowEngine - Local" en el dropdown superior derecho
2. (Opcional) Modifica el `actor_id` si deseas usar uno personalizado

### 3. Ejecutar Ejemplo Completo

1. Navega a: **Complete Workflow Examples → Example 1: Full Approval Flow**
2. Ejecuta los requests **en orden** (1 → 2 → 3 → 4 → 5)
3. Observa la consola de Postman para ver los logs:
   ```
   ✅ Workflow created: a1b2c3d4...
   ✅ Instance created: b2c3d4e5...
   ✅ Transition to review successful
   ✅ Transition to approved successful
   ✅ Final state: approved
   ```

---

## 🔍 Scripts Automáticos Incluidos

### Pre-request Scripts

Configuran valores por defecto:
```javascript
// Si no hay base_url definida, usa localhost
if (!pm.environment.get("base_url")) {
    pm.environment.set("base_url", "http://localhost:8080");
}
```

### Test Scripts

Validan respuestas y auto-guardan IDs:
```javascript
// Al crear un workflow
if (pm.response.code === 201) {
    var jsonData = pm.response.json();
    pm.environment.set("workflow_id", jsonData.id);
    console.log("✅ Workflow created: " + jsonData.id);
}
```

---

## 📊 Validaciones Automáticas

Todos los requests incluyen validaciones que se ejecutan automáticamente:

- ✅ **Status code** - Verifica 200, 201, etc.
- ✅ **Response structure** - Valida campos obligatorios
- ✅ **Auto-save IDs** - Guarda IDs para requests posteriores
- ✅ **Console logs** - Mensajes útiles en la consola

---

## 🎯 Workflows de Prueba Sugeridos

### Testing Básico
1. Health Check
2. Create Workflow - Simple Approval
3. List All Workflows
4. Create Instance
5. Get Instance by ID

### Testing de Transiciones
1. Create Workflow
2. Create Instance
3. Submit for Review
4. Approve
5. Get Instance by ID (ver historial completo)

### Testing de Rechazo
1. Create Workflow
2. Create Instance
3. Submit for Review
4. Reject
5. Get Instance by ID

### Testing de Purchase Order
1. Create Workflow - Complex Purchase Order
2. Create Instance - Purchase Order
3. Submit
4. Manager Approve
5. Finance Approve
6. Get Instance by ID

---

## 🐛 Troubleshooting

### Error: "workflow_id is not defined"
**Causa**: No has creado un workflow aún
**Solución**: Ejecuta primero "Create Workflow"

### Error: "instance_id is not defined"
**Causa**: No has creado una instancia aún
**Solución**: Ejecuta primero "Create Instance"

### Error: Connection refused
**Causa**: El servidor no está corriendo
**Solución**:
```bash
make run
# O: go run cmd/api/main.go
```

### Error: Invalid transition
**Causa**: La transición no es válida desde el estado actual
**Solución**: Verifica el estado actual con "Get Instance by ID" y ejecuta la transición correcta

---

## 📚 Recursos

- **Documentación completa**: `postman/README.md`
- **API Quickstart**: `docs/api_quickstart.md`
- **Ejemplos JSON**: `examples/*.json`
- **Código del servidor**: `cmd/api/main.go`

---

## 🎉 ¡Listo para Usar!

La colección de Postman está **100% funcional** y lista para:
- ✅ Desarrollo local
- ✅ Testing manual
- ✅ Demos y presentaciones
- ✅ Validación de workflows
- ✅ Aprendizaje del API

**¡Importa y comienza a probar! 🚀**
