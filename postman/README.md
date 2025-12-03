# FlowEngine API - Postman Collection

Esta carpeta contiene las colecciones de Postman para interactuar con la API de FlowEngine.

## Archivos Incluidos

| Archivo | Descripcion |
|---------|-------------|
| `FlowEngine_API.postman_collection.json` | Coleccion general con todos los endpoints |
| `FlowEngine_MinTrabajo_Collection.json` | **NUEVO** - Coleccion completa para el flujo MinTrabajo (HU-006) |
| `FlowEngine_Environment.postman_environment.json` | Environment para desarrollo local |
| `FlowEngine_Environment.json` | Environment alternativo |

---

## Flujo MinTrabajo (HU-006) - NUEVO

### Proyecto
- **Cliente:** POSITIVA
- **Proyecto:** SGDEA DOCUM - POSITIVA
- **Historia de Usuario:** HU-006_clonacion_de_Procesos_Min_Trabajo

### Como usar esta coleccion

1. Importar `FlowEngine_MinTrabajo_Collection.json` en Postman
2. Importar `FlowEngine_Environment.json` como environment
3. Ejecutar en orden:
   - `01 - Setup / Get Auth Token`
   - `01 - Setup / Create Workflow from YAML`
   - Seguir el flujo deseado

### Escenarios Disponibles

| Carpeta | Descripcion |
|---------|-------------|
| 01 - Setup | Configuracion inicial (auth, crear workflow) |
| 02 - Flujo Principal | Happy path completo: Radicacion -> Aprobacion |
| 03 - Flujo con Rechazo | Escenario de rechazo y subsanacion |
| 04 - Proceso de Clonacion | Flujo completo de clonacion (HU-006) |
| 05 - Reclasificacion | Enviar a PQRD o Entes de Control |
| 06 - Consultas | Endpoints de consulta |

### Diagrama de Estados MinTrabajo

```
radicado -> por_asignar -> en_asignacion -> para_gestion -> en_edicion
                |               |                              |
                v               v                              v
          reclasificado   reclasificado                  por_revisar
             (FINAL)         (FINAL)                          |
                                                    +---------+---------+
                                                    v                   v
                                             revision_aprobada   revision_rechazada
                                                    |                   |
                                                    v                   |
                                              por_aprobar               |
                                                    |                   |
                                                    v                   |
                                               aprobado <---------------+
                                                (FINAL)      (subsanar)
```

---

## 🚀 Instalación en Postman

### Opción 1: Importar Archivos (Recomendado)

1. Abre Postman
2. Click en **Import** (botón superior izquierdo)
3. Arrastra los dos archivos JSON o selecciónalos:
   - `FlowEngine_API.postman_collection.json`
   - `FlowEngine_Environment.postman_environment.json`
4. Click en **Import**
5. Selecciona el environment "FlowEngine - Local" en el dropdown superior derecho

### Opción 2: Importar desde URL

Si el proyecto está en GitHub, puedes importar directamente la URL raw de los archivos.

## 🎯 Estructura de la Colección

### 1. Health Check
- **Health Check** - Verificar que el servidor está corriendo

### 2. Workflows
- **Create Workflow - Simple Approval** - Crear workflow simple de aprobación
- **Create Workflow - Complex Purchase Order** - Crear workflow complejo de orden de compra
- **List All Workflows** - Listar todos los workflows
- **Get Workflow by ID** - Obtener detalles de un workflow

### 3. Instances
- **Create Instance** - Crear nueva instancia de workflow
- **Create Instance - Purchase Order** - Crear instancia de orden de compra
- **List All Instances** - Listar todas las instancias
- **List Instances by Workflow** - Filtrar instancias por workflow
- **Get Instance by ID** - Obtener detalles de una instancia

### 4. Transitions
- **Submit for Review** - Transición: draft → review
- **Approve** - Transición: review → approved
- **Reject** - Transición: review → rejected
- **Manager Approve (PO)** - Aprobación de gerente
- **Finance Approve (PO)** - Aprobación final de finanzas

### 5. Complete Workflow Examples
- **Example 1: Full Approval Flow** - Flujo completo paso a paso
  1. Create Workflow
  2. Create Instance
  3. Submit for Review
  4. Approve
  5. Get Final State

## 🔧 Variables de Environment

La colección usa las siguientes variables que se auto-configuran:

| Variable | Descripción | Valor Inicial |
|----------|-------------|---------------|
| `base_url` | URL base del API | `http://localhost:8080` |
| `workflow_id` | ID del workflow (auto-guardado) | _(vacío)_ |
| `instance_id` | ID de la instancia (auto-guardado) | _(vacío)_ |
| `actor_id` | ID del actor que ejecuta acciones | `550e8400-e29b-41d4-a716-446655440000` |

### ✨ Auto-guardado de IDs

Los requests de creación incluyen **scripts automáticos** que guardan los IDs en las variables:

```javascript
// Al crear un workflow, el ID se guarda automáticamente
if (pm.response.code === 201) {
    var jsonData = pm.response.json();
    pm.environment.set("workflow_id", jsonData.id);
}
```

Esto permite ejecutar los requests en secuencia sin copiar/pegar IDs manualmente.

## 📖 Cómo Usar

### Flujo Rápido (Recomendado)

1. **Inicia el servidor**:
   ```bash
   make run
   # O: go run cmd/api/main.go
   ```

2. **Ejecuta "Example 1: Full Approval Flow"**:
   - Abre la carpeta "Complete Workflow Examples"
   - Ejecuta los 5 requests en orden (tienen números)
   - Los IDs se auto-configuran entre requests

### Flujo Manual

1. **Health Check**:
   - Ejecuta "Health Check" para verificar que el servidor está corriendo

2. **Crear Workflow**:
   - Ejecuta "Create Workflow - Simple Approval"
   - El `workflow_id` se guarda automáticamente

3. **Crear Instancia**:
   - Ejecuta "Create Instance"
   - El `instance_id` se guarda automáticamente

4. **Ejecutar Transiciones**:
   - Ejecuta "Submit for Review" (draft → review)
   - Ejecuta "Approve" (review → approved)

5. **Ver Estado Final**:
   - Ejecuta "Get Instance by ID" para ver el historial completo

## 🎨 Características Especiales

### 1. Scripts de Pre-request
Configuran valores por defecto si no están definidos:
```javascript
if (!pm.environment.get("actor_id")) {
    pm.environment.set("actor_id", "550e8400-e29b-41d4-a716-446655440000");
}
```

### 2. Scripts de Test
Auto-guardan IDs y muestran mensajes en la consola:
```javascript
if (pm.response.code === 201) {
    var jsonData = pm.response.json();
    pm.environment.set("workflow_id", jsonData.id);
    console.log("✅ Workflow created: " + jsonData.id);
}
```

### 3. Validación de Respuestas
Los scripts de test validan automáticamente las respuestas y muestran el estado en la consola de Postman.

## 📊 Ejemplos de Workflows Incluidos

### 1. Simple Approval Workflow
**Estados**: Draft → Review → Approved/Rejected

**Eventos**:
- `submit`: Enviar para revisión
- `approve`: Aprobar
- `reject`: Rechazar

### 2. Purchase Order Workflow
**Estados**: Draft → Manager Review → Finance Review → Approved/Rejected/Cancelled

**Eventos**:
- `submit`: Enviar para aprobación
- `manager_approve`: Aprobación de gerente
- `manager_reject`: Rechazo de gerente
- `finance_approve`: Aprobación de finanzas
- `finance_reject`: Rechazo de finanzas
- `cancel`: Cancelar orden

## 🔍 Tips de Uso

### Ver IDs Actuales
1. Click en el icono del "ojo" 👁️ en la esquina superior derecha
2. Verás todas las variables del environment activo

### Limpiar IDs
Si quieres empezar de nuevo:
1. Hover sobre la variable en el environment
2. Click en el icono de reset ↻

### Console de Postman
- Abre la consola: View → Show Postman Console (Alt+Ctrl+C)
- Verás logs útiles de cada request:
  ```
  ✅ Workflow created: a1b2c3d4-e5f6-7890-abcd-ef1234567890
  ✅ Instance created: b2c3d4e5-f6a7-8901-bcde-f12345678901
  ✅ Transition to review successful
  ```

## 🧪 Testing Avanzado

### Test de Carga
Usa el **Collection Runner** para ejecutar múltiples iteraciones:
1. Click en la colección → Run
2. Selecciona las iteraciones
3. Ejecuta y revisa las estadísticas

### Data-Driven Testing
Puedes crear un CSV con datos y usar el Collection Runner para ejecutar requests con diferentes datos.

## 🐛 Troubleshooting

### Error: "base_url is not defined"
**Solución**: Asegúrate de tener el environment "FlowEngine - Local" seleccionado.

### Error: "workflow_id is not defined"
**Solución**: Ejecuta primero un request de "Create Workflow" para generar el ID.

### Error: Connection refused
**Solución**: Verifica que el servidor esté corriendo en `http://localhost:8080`
```bash
curl http://localhost:8080/health
```

## 📚 Recursos Adicionales

- **API Documentation**: `../docs/api_quickstart.md`
- **Examples**: `../examples/*.json`
- **Server Code**: `../cmd/api/main.go`

## 🎉 Happy Testing!

Si encuentras problemas o tienes sugerencias, abre un issue en el repositorio.
