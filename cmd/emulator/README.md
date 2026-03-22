# 🎮 FlowEngine Emulator & Testing Tools

Tres formas de probar y emular workflows en FlowEngine.

---

## 🚀 Opción 1: Quick Test (Modificar código y ejecutar)

**Archivo**: `cmd/quick-test/main.go`

### Uso:
```bash
# Modificar el archivo con tu workflow
nano cmd/quick-test/main.go

# Ejecutar
go run cmd/quick-test/main.go
```

### Ventajas:
- ✅ Rápido para pruebas simples
- ✅ Código fácil de modificar
- ✅ Perfecto para aprender la API

### Ejemplo de modificación:

```go
// Crear tus estados
draft, _ := workflow.NewState("draft", "Borrador")
review, _ := workflow.NewState("review", "En Revisión")

// Crear workflow
wf, _ := workflow.NewWorkflow("Mi Workflow", draft, actor)

// Agregar transiciones
inst.Transition(review.ID(), "submit", actor, metadata)
```

---

## 🎮 Opción 2: CLI Interactivo (REPL)

**Archivo**: `cmd/emulator/main.go`

### Uso:
```bash
go run cmd/emulator/main.go
```

### Comandos disponibles:

#### Workflows:
```
create-workflow (cw)        - Crear workflow paso a paso
list-workflows (lw)         - Listar todos los workflows
show-workflow <name> (sw)   - Ver detalles de un workflow
```

#### Instancias:
```
create-instance <workflow> (ci)  - Crear instancia
list-instances (li)              - Listar instancias
show-instance <id> (si)          - Ver detalles
```

#### Operaciones:
```
transition <id> <event> (t)  - Ejecutar transición
pause <id>                   - Pausar instancia
resume <id>                  - Reanudar instancia
complete <id>                - Completar instancia
cancel <id>                  - Cancelar instancia
history <id>                 - Ver historial
events                       - Ver todos los eventos
```

### Ejemplo de sesión:

```bash
> create-workflow
  Workflow name: Aprobación de Documentos
  Initial state ID: draft
  Initial state name: Borrador

  Add additional states:
    State ID: review
    State name: En Revisión
    Is final state? (y/n): n
    ✅ Added state: review

    State ID: approved
    State name: Aprobado
    Is final state? (y/n): y
    ✅ Added state: approved

    State ID: (presionar Enter para terminar)

  Add events/transitions:
    Event name: submit
    From state ID: draft
    To state ID: review
    ✅ Added event: submit (draft → review)

    Event name: approve
    From state ID: review
    To state ID: approved
    ✅ Added event: approve (review → approved)

✅ Workflow 'Aprobación de Documentos' created!

> create-instance Aprobación de Documentos
✅ Instance created: a5618cc4

> transition a5618cc4 submit
✅ Transition executed: submit
  New State: review

> transition a5618cc4 approve
✅ Transition executed: approve
  New State: approved

> complete a5618cc4
✅ Instance completed: a5618cc4

> history a5618cc4
📜 Transition History (2 transitions):
  [1] submit
      draft → review
      Time: 19:43:25
  [2] approve
      review → approved
      Time: 19:43:26

> events
📊 Domain Events (5 total):
  • workflow.created          : 1
  • instance.created          : 1
  • instance.state_changed    : 2
  • instance.completed        : 1
```

---

## 🎯 Opción 3: Demo Completa (Ver funcionamiento)

**Archivo**: `cmd/demo/main.go`

### Uso:
```bash
go run cmd/demo/main.go
```

### Qué muestra:
- Workflow de aprobación de documentos completo
- Sub-estados (R17)
- Metadata de transiciones (R23)
- Pause/Resume
- Historial completo
- Resumen de eventos

---

## 📚 Ejemplos de Workflows para Probar

### 1. Workflow de Tickets de Soporte

```bash
> create-workflow
  Workflow name: Ticket Support

Estados:
  - new (Nuevo)
  - in_progress (En Progreso)
  - waiting_customer (Esperando Cliente)
  - resolved (Resuelto) [FINAL]
  - closed (Cerrado) [FINAL]

Eventos:
  - assign: new → in_progress
  - request_info: in_progress → waiting_customer
  - respond: waiting_customer → in_progress
  - resolve: in_progress → resolved
  - close: resolved → closed
```

### 2. Workflow de Pedidos E-commerce

```bash
Estados:
  - pending (Pendiente)
  - payment_confirmed (Pago Confirmado)
  - preparing (Preparando)
  - shipped (Enviado)
  - delivered (Entregado) [FINAL]
  - canceled (Cancelado) [FINAL]

Eventos:
  - confirm_payment: pending → payment_confirmed
  - start_preparing: payment_confirmed → preparing
  - ship: preparing → shipped
  - deliver: shipped → delivered
  - cancel: pending/payment_confirmed → canceled
```

### 3. Workflow de Contratación

```bash
Estados:
  - applied (Aplicado)
  - screening (Filtrado)
  - interview (Entrevista)
  - offer (Oferta)
  - hired (Contratado) [FINAL]
  - rejected (Rechazado) [FINAL]

Eventos:
  - screen: applied → screening
  - schedule_interview: screening → interview
  - make_offer: interview → offer
  - hire: offer → hired
  - reject: * → rejected
```

### 4. Workflow de Vacaciones

```bash
Estados:
  - requested (Solicitado)
  - manager_approval (Aprobación Manager)
  - hr_approval (Aprobación RRHH)
  - approved (Aprobado) [FINAL]
  - rejected (Rechazado) [FINAL]

Eventos:
  - manager_approve: requested → manager_approval
  - hr_approve: manager_approval → hr_approval
  - final_approve: hr_approval → approved
  - reject: * → rejected
```

---

## 🧪 Casos de Prueba Sugeridos

### Test 1: Happy Path
```bash
1. Crear workflow simple
2. Crear instancia
3. Ejecutar todas las transiciones hasta estado final
4. Completar instancia
5. Verificar historial
```

### Test 2: Validaciones
```bash
1. Intentar transición inválida
2. Pausar y reanudar
3. Intentar operar instancia completada
4. Ver errores descriptivos
```

### Test 3: Sub-Estados
```bash
1. Crear instancia
2. Usar transiciones con sub-estados
3. Cambiar solo sub-estado
4. Verificar eventos SubStateChanged
```

### Test 4: Metadata
```bash
1. Agregar metadata con reason/feedback
2. Agregar campos personalizados
3. Ver metadata en historial
4. Verificar en eventos
```

### Test 5: Ciclos y Loops
```bash
1. Crear workflow con ciclo (A → B → A)
2. Ejecutar múltiples veces el ciclo
3. Verificar versionado
4. Ver historial completo
```

---

## 💡 Tips

### Emulator CLI:
- Usa IDs cortos (primeros 8 caracteres)
- Los workflows se crean en memoria (no persisten)
- `clear` para limpiar pantalla
- `help` siempre disponible

### Quick Test:
- Modifica directamente el código
- Experimenta con diferentes escenarios
- Usa `fmt.Println` para debug
- Corre múltiples veces sin problemas

### Demo:
- No requiere interacción
- Perfecto para presentaciones
- Muestra todas las features
- Código comentado para aprender

---

## 🐛 Troubleshooting

### Error: "workflow not found"
- Verifica el nombre exacto (case-sensitive)
- Usa `list-workflows` para ver disponibles

### Error: "instance not found"
- Usa el ID corto de 8 caracteres
- Usa `list-instances` para ver IDs

### Error: "event not found"
- Verifica el nombre del evento
- Usa `show-workflow` para ver eventos disponibles

### Error: "invalid transition"
- Verifica que la instancia esté en el estado correcto
- El evento debe tener el estado actual como source

---

## 📖 Siguientes Pasos

Después de probar workflows:

1. **Fase 6**: Implementar persistencia PostgreSQL
2. **Fase 7**: Agregar caching con Redis
3. **Fase 8**: Application layer (use cases)
4. **Fase 9**: REST API
5. **Fase 10**: Sistema de Eventos (MultiDispatcher + WebhookDispatcher)

---

## 🎓 Recursos

- **Código**: Ver implementación en archivos
- **Tests**: `internal/domain/*/` para ejemplos
- **Docs**: `requirements.md`, `design.md`, `task.md`

---

**Happy Testing! 🚀**
