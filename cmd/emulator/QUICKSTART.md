# 🎮 Guía Rápida - CLI Interactivo

## Para ejecutar el emulador:

```bash
go run cmd/emulator/main.go
```

---

## 📚 Tutorial Paso a Paso

### Sesión de Ejemplo: Crear Workflow de Aprobación

```
> create-workflow

  Workflow name: Aprobacion Simple
  Initial state ID: draft
  Initial state name: Borrador

  Add additional states (enter empty ID to finish):
    State ID: review
    State name: En Revision
    Is final state? (y/n): n
    ✅ Added state: review

    State ID: approved
    State name: Aprobado
    Is final state? (y/n): y
    ✅ Added state: approved

    State ID: rejected
    State name: Rechazado
    Is final state? (y/n): y
    ✅ Added state: rejected

    State ID: [Enter para terminar]

  Add events/transitions (enter empty event name to finish):
    Event name: submit
    From state ID: draft
    To state ID: review
    ✅ Added event: submit (draft → review)

    Event name: approve
    From state ID: review
    To state ID: approved
    ✅ Added event: approve (review → approved)

    Event name: reject
    From state ID: review
    To state ID: rejected
    ✅ Added event: reject (review → rejected)

    Event name: [Enter para terminar]

✅ Workflow 'Aprobacion Simple' created successfully!

> list-workflows
📋 Available Workflows:
  • Aprobacion Simple [ID: 4ed77ad0, States: 4, Events: 3]

> show-workflow Aprobacion Simple
📋 Workflow: Aprobacion Simple
  ID: 4ed77ad0-1234-5678-9abc-def012345678
  Version: 1.0.0
  Initial State: draft

  States:
    • draft - Borrador
    • review - En Revision
    • approved - Aprobado [FINAL]
    • rejected - Rechazado [FINAL]

  Events:
    • submit: [draft] → review
    • approve: [review] → approved
    • reject: [review] → rejected

> create-instance Aprobacion Simple
✅ Instance created: a5618cc4
  Workflow: Aprobacion Simple
  State: draft
  Status: RUNNING

> list-instances
🎬 Active Instances:
  • a5618cc4: Aprobacion Simple [State: draft, Status: RUNNING]

> show-instance a5618cc4
🎬 Instance: a5618cc4
  Workflow: Aprobacion Simple [4ed77ad0]
  Current State: draft
  Status: RUNNING
  Version: v1
  Transitions: 0
  Created: 2025-11-10 19:43:25

> transition a5618cc4 submit
✅ Transition executed: submit
  New State: review
  Status: RUNNING
  Version: v2

> transition a5618cc4 approve
✅ Transition executed: approve
  New State: approved
  Status: RUNNING
  Version: v3

> complete a5618cc4
✅ Instance completed: a5618cc4

> history a5618cc4
📜 Transition History for a5618cc4 (2 transitions):

  [1] submit
      draft → review
      Time: 19:43:26
      Reason: Manual transition: submit

  [2] approve
      review → approved
      Time: 19:43:27
      Reason: Manual transition: approve

> events
📊 Domain Events (5 total):
  • workflow.created              : 1
  • instance.created              : 1
  • instance.state_changed        : 2
  • instance.completed            : 1

> help
📚 Available Commands:

  Workflow Management:
    create-workflow (cw)           - Create a new workflow interactively
    list-workflows (lw)            - List all workflows
    show-workflow <name> (sw)      - Show workflow details

  Instance Management:
    create-instance <workflow> (ci) - Create a new instance
    list-instances (li)            - List all instances
    show-instance <id> (si)        - Show instance details

  Instance Operations:
    transition <id> <event> (t)    - Execute a transition
    pause <id>                     - Pause an instance
    resume <id>                    - Resume a paused instance
    complete <id>                  - Complete an instance
    cancel <id>                    - Cancel an instance
    history <id>                   - Show transition history

  Events & Utilities:
    events                         - Show all domain events
    clear                          - Clear screen
    help (h)                       - Show this help
    exit (quit, q)                 - Exit emulator

> exit
👋 ¡Hasta luego!
```

---

## 🎯 Comandos Más Usados

### Atajos Cortos
```
cw    = create-workflow
lw    = list-workflows
sw    = show-workflow
ci    = create-instance
li    = list-instances
si    = show-instance
t     = transition
h     = help
q     = quit
```

### Flujo Típico
```bash
1. cw                              # Crear workflow
2. lw                              # Ver workflows disponibles
3. ci MiWorkflow                   # Crear instancia
4. li                              # Ver instancias
5. t <id> <evento>                 # Ejecutar transición
6. history <id>                    # Ver historial
7. events                          # Ver eventos
```

---

## 💡 Tips

### Para usar IDs cortos:
- El CLI muestra IDs de 8 caracteres (ej: `a5618cc4`)
- Usa estos IDs cortos en los comandos
- No necesitas el UUID completo

### Auto-completado:
- Presiona Tab para... bueno, no hay auto-completado aún 😅
- Pero puedes usar `list-workflows` y `list-instances` para ver opciones

### Limpiar pantalla:
```
> clear
```

### Ver todo sobre una instancia:
```
> show-instance <id>
> history <id>
```

---

## 🚀 Ahora Sí, ¡A Probar!

Ejecuta:
```bash
go run cmd/emulator/main.go
```

Y sigue el tutorial de arriba para crear tu primer workflow!

---

## 🐛 Si algo no funciona

- **Error: workflow not found** → Usa el nombre exacto (case-sensitive)
- **Error: instance not found** → Usa el ID corto de 8 caracteres
- **Error: event not found** → Verifica con `show-workflow <name>`
- **Error: invalid transition** → Verifica el estado actual con `show-instance <id>`
