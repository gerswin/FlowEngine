# FlowEngine Domain Layer Demo

Demo interactiva que muestra todas las capacidades del motor de workflows FlowEngine (Fases 1-5).

## 🚀 Ejecutar la Demo

```bash
# Desde la raíz del proyecto
go run cmd/demo/main.go

# O compilar y ejecutar
go build -o demo cmd/demo/main.go
./demo
```

## 📋 Qué Demuestra

### 1. **Definición de Workflow** (Fase 3)
- Creación de workflow "Document Approval"
- 5 estados: draft, under_review, approved, rejected, needs_revisions
- 5 eventos de transición: submit, approve, reject, request_revisions, resubmit
- Estados finales marcados (approved, rejected)

### 2. **Creación y Ejecución de Instancias** (Fase 4)
- Crear instancia de workflow
- Asignar datos y variables
- Ejecutar transiciones de estado
- Validaciones de reglas de negocio

### 3. **Sub-Estados (R17)**
- Crear sub-estados: qa_check, compliance_check
- Transiciones con sub-estados
- Cambios de sub-estado sin afectar el estado principal
- Eventos SubStateChanged

### 4. **Metadata de Transición (R23)**
- Agregar reason y feedback a transiciones
- Metadata estructurada con campos personalizados
- Validación de metadata
- Captura de información contextual

### 5. **Sistema de Eventos del Dominio** (Fase 5)
- 11 tipos de eventos implementados
- Tracking automático de eventos
- Event dispatcher (NullDispatcher, InMemoryDispatcher)
- Resumen de eventos generados

### 6. **Operaciones de Ciclo de Vida**
- **Pause**: Pausar instancia con razón
- **Resume**: Reanudar instancia pausada
- **Complete**: Completar instancia en estado final
- **Cancel**: Cancelar instancia con razón

### 7. **Validación y Manejo de Errores**
- Validación de transiciones inválidas
- Prevención de operaciones en instancias completadas
- Errores de dominio tipados
- Mensajes de error descriptivos

### 8. **Historial de Transiciones**
- Registro completo de todas las transiciones
- Metadata asociada a cada transición
- Timestamps precisos
- Sub-states en historial

### 9. **Optimistic Locking**
- Versionado automático con cada operación
- Incremento de versión en cada cambio
- Tracking de versión desde v1 hasta vN

## 📊 Estadísticas de la Demo

```
Workflows creados:     1
Instancias creadas:    2
Transiciones:          7
Eventos de dominio:    17
Características:       9/9 (100%)
```

## 🎯 Casos de Uso Demostrados

### Caso 1: Aprobación Completa (Happy Path)
```
draft → under_review → needs_revisions → under_review → approved → COMPLETED
```
- Incluye sub-estados para QA y Compliance
- Metadata detallada en cada paso
- Pause/Resume durante el proceso

### Caso 2: Cancelación (Cancel Path)
```
draft → under_review → CANCELED
```
- Creación de segunda instancia
- Cancelación por razón de negocio
- Evento InstanceCanceled generado

## 🔍 Salida de la Demo

La demo produce output visual con:
- ✅ Checkmarks para operaciones exitosas
- ❌ X para errores esperados (validación)
- ℹ️  Info para datos descriptivos
- 📊 Resumen de eventos
- 🎉 Estadísticas finales

### Colores en Terminal
- 🟢 Verde: Operaciones exitosas
- 🔴 Rojo: Errores
- 🔵 Azul: Headers
- 🟡 Amarillo: Sub-headers
- 🟣 Morado: Log de eventos
- 🔵 Cyan: Info general

## 📝 Ver Eventos Detallados

Para ver el payload completo de cada evento, descomenta esta línea en `main.go`:

```go
// tracker.PrintDetailed()  // <-- Descomentar esta línea
```

Esto mostrará el payload JSON completo de todos los eventos generados.

## 🛠️ Modificar la Demo

### Agregar Nuevo Estado
```go
customState, _ := workflow.NewState("custom", "Custom State")
approvalWorkflow.AddState(customState)
```

### Agregar Nueva Transición
```go
customEvent, _ := workflow.NewEvent("custom_transition",
    []workflow.State{sourceState},
    customState)
approvalWorkflow.AddEvent(customEvent)
```

### Agregar Metadata Personalizada
```go
metadata := instance.NewTransitionMetadata(
    "Reason for transition",
    "Optional feedback text",
    map[string]interface{}{
        "custom_field": "value",
        "priority": 1,
        "tags": []string{"urgent", "review"},
    },
)
```

## 🧪 Testing

La demo incluye:
- Validación de reglas de negocio
- Manejo de errores
- Demostración de casos edge

Para ejecutar tests relacionados:
```bash
# Tests del dominio completo
go test ./internal/domain/... -v

# Tests con coverage
go test ./internal/domain/... -cover

# Tests de eventos específicamente
go test ./internal/domain/event/... -v
```

## 📚 Recursos Adicionales

- **Requisitos**: `/home/user/FlowEngine/requirements.md`
- **Diseño**: `/home/user/FlowEngine/design.md`
- **Plan**: `/home/user/FlowEngine/task.md`
- **Tests**: `/home/user/FlowEngine/internal/domain/*/`

## 🎓 Próximos Pasos

Después de entender la demo:

1. **Fase 6**: Implementar PostgreSQL persistence
2. **Fase 7**: Implementar Redis caching
3. **Fase 8**: Implementar Application layer (use cases)
4. **Fase 9**: Implementar REST API con Gin
5. **Fase 10**: Sistema de Eventos (MultiDispatcher + WebhookDispatcher)

## 💡 Tips

- La demo usa colores ANSI para mejor visualización
- Funciona mejor en terminales con soporte de color
- Los IDs se muestran truncados (primeros 8 caracteres)
- Los timestamps están en UTC
- La demo es completamente self-contained (no requiere BD)

## ❓ Troubleshooting

### Demo no compila
```bash
# Asegurarse de estar en la raíz del proyecto
cd /home/user/FlowEngine
go mod tidy
go run cmd/demo/main.go
```

### Sin colores en Windows
Los colores ANSI no funcionan bien en cmd.exe. Usar:
- PowerShell
- Windows Terminal
- Git Bash
- WSL

### Error de imports
```bash
# Regenerar go.mod si es necesario
go mod init github.com/LaFabric-LinkTIC/FlowEngine
go mod tidy
```

---

**Versión**: 1.0.0
**Última actualización**: 2025-11-10
**Autor**: LaFabric-LinkTIC
**Licencia**: TBD
