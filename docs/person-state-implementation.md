# Implementación en Go: Sistema de Estados de Personas

## Overview

Este documento contiene la implementación en Go de los componentes clave del sistema de estados para personas (documentos/solicitudes).

---

## 1. Domain Layer

### 1.1 SubState Value Object

```go
// internal/domain/instance/substate.go
package instance

import "errors"

// SubState representa un subestado dentro de un estado principal
type SubState string

const (
    SubStateEmpty                     SubState = ""
    SubStateWorking                   SubState = "working"
    SubStateEscalatedAwaitingResponse SubState = "escalated_awaiting_response"
    SubStateEscalationResponded       SubState = "escalation_responded"
)

var validSubStates = map[SubState]bool{
    SubStateEmpty:                     true,
    SubStateWorking:                   true,
    SubStateEscalatedAwaitingResponse: true,
    SubStateEscalationResponded:       true,
}

// NewSubState creates a validated SubState
func NewSubState(value string) (SubState, error) {
    ss := SubState(value)
    if !validSubStates[ss] && value != "" {
        return SubStateEmpty, errors.New("invalid sub-state")
    }
    return ss, nil
}

// String returns the string representation
func (s SubState) String() string {
    return string(s)
}

// IsEmpty checks if the substate is empty
func (s SubState) IsEmpty() bool {
    return s == SubStateEmpty
}

// Equals compares two substates
func (s SubState) Equals(other SubState) bool {
    return s == other
}
```

### 1.2 Instance Aggregate - Extended

```go
// internal/domain/instance/instance.go (extensión)
package instance

import (
    "context"
    "errors"
    "time"

    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
)

type Instance struct {
    // Campos existentes...
    id              shared.ID
    workflowID      workflow.ID
    parentID        *shared.ID
    currentState    workflow.State
    previousState   *workflow.State
    version         Version
    status          Status
    data            Data
    variables       Variables
    history         []Transition
    domainEvents    []event.DomainEvent
    createdAt       shared.Timestamp
    updatedAt       shared.Timestamp
    completedAt     *shared.Timestamp

    // Nuevos campos para subestados
    currentSubState  SubState
    previousSubState *SubState
}

// SetSubState cambia el subestado de la instancia
func (i *Instance) SetSubState(substate SubState) error {
    // Validar que el estado actual soporta subestados
    if !i.currentState.SupportsSubStates() {
        return errors.New("current state does not support sub-states")
    }

    // Guardar subestado anterior
    if !i.currentSubState.IsEmpty() {
        previous := i.currentSubState
        i.previousSubState = &previous
    }

    // Actualizar subestado
    i.currentSubState = substate
    i.version = i.version.Increment()
    i.updatedAt = shared.Now()

    // Generar evento de dominio
    evt := event.NewSubStateChanged(
        i.id.String(),
        i.currentState.ID(),
        i.previousSubState,
        substate,
        time.Now(),
    )
    i.addDomainEvent(evt)

    return nil
}

// CurrentSubState returns the current substate
func (i *Instance) CurrentSubState() SubState {
    return i.currentSubState
}

// PreviousSubState returns the previous substate (if any)
func (i *Instance) PreviousSubState() *SubState {
    return i.previousSubState
}

// TransitionWithSubState ejecuta una transición con cambio de subestado
func (i *Instance) TransitionWithSubState(
    ctx context.Context,
    wf *workflow.Workflow,
    eventName string,
    actorID string,
    targetSubState *SubState,
) error {
    // Ejecutar transición normal
    if err := i.Transition(ctx, wf, eventName, actorID); err != nil {
        return err
    }

    // Si se especifica subestado objetivo, aplicarlo
    if targetSubState != nil && !targetSubState.IsEmpty() {
        if err := i.SetSubState(*targetSubState); err != nil {
            return err
        }
    }

    return nil
}

// AddFeedback agrega feedback a las variables (usado en rechazos)
func (i *Instance) AddFeedback(reason, feedback, rejectedBy string) {
    // Obtener o crear array de feedbacks
    var feedbacks []map[string]interface{}
    if existing, ok := i.variables.Get("feedbacks"); ok {
        if arr, ok := existing.([]map[string]interface{}); ok {
            feedbacks = arr
        }
    }

    // Agregar nuevo feedback
    feedbacks = append(feedbacks, map[string]interface{}{
        "reason":      reason,
        "feedback":    feedback,
        "rejected_by": rejectedBy,
        "rejected_at": time.Now(),
    })

    i.variables.Set("feedbacks", feedbacks)

    // Incrementar contador de rechazos
    rejectionCount := 0
    if count, ok := i.variables.Get("rejection_count"); ok {
        if intCount, ok := count.(int); ok {
            rejectionCount = intCount
        }
    }
    i.variables.Set("rejection_count", rejectionCount+1)

    i.version = i.version.Increment()
}
```

### 1.3 Workflow State - Extended

```go
// internal/domain/workflow/state.go (extensión)
package workflow

type State struct {
    id            string
    name          string
    description   string
    timeout       *time.Duration
    onTimeout     *string
    isFinal       bool
    allowedRoles  []string
    subStates     []SubStateDefinition  // Nuevo campo
}

type SubStateDefinition struct {
    ID          string
    Name        string
    Description string
    IsDefault   bool
}

// SupportsSubStates verifica si el estado soporta subestados
func (s State) SupportsSubStates() bool {
    return len(s.subStates) > 0
}

// SubStates retorna los subestados definidos
func (s State) SubStates() []SubStateDefinition {
    return s.subStates
}

// WithSubStates agrega subestados a la definición
func (s State) WithSubStates(substates []SubStateDefinition) State {
    s.subStates = substates
    return s
}

// AllowedRoles retorna los roles permitidos
func (s State) AllowedRoles() []string {
    return s.allowedRoles
}
```

---

## 2. Guards de Permisos

### 2.1 Guard Interface

```go
// internal/domain/workflow/guard.go
package workflow

import (
    "context"

    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/actor"
    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
)

// Guard valida si una transición es permitida
type Guard interface {
    Validate(ctx context.Context, inst *instance.Instance, actor *actor.Actor) error
}

// GuardFunc es un adapter para usar funciones como Guards
type GuardFunc func(ctx context.Context, inst *instance.Instance, actor *actor.Actor) error

func (f GuardFunc) Validate(ctx context.Context, inst *instance.Instance, actor *actor.Actor) error {
    return f(ctx, inst, actor)
}
```

### 2.2 Role Guard

```go
// internal/domain/workflow/guard_role.go
package workflow

import (
    "context"
    "fmt"

    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/actor"
    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
)

// RoleGuard verifica que el actor tenga un rol específico
type RoleGuard struct {
    requiredRole actor.Role
}

// NewRoleGuard crea un guard de rol
func NewRoleGuard(role actor.Role) Guard {
    return &RoleGuard{requiredRole: role}
}

func (g *RoleGuard) Validate(ctx context.Context, inst *instance.Instance, actor *actor.Actor) error {
    if !actor.HasRole(g.requiredRole) {
        return fmt.Errorf("actor does not have required role: %s", g.requiredRole)
    }
    return nil
}
```

### 2.3 Assignment Guard

```go
// internal/domain/workflow/guard_assignment.go
package workflow

import (
    "context"
    "errors"

    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/actor"
    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
)

// AssignmentGuard verifica que el documento esté asignado al actor
type AssignmentGuard struct {
    checkGroup bool  // Si TRUE, también verifica pertenencia a grupo
}

// NewAssignmentGuard crea un guard de asignación
func NewAssignmentGuard(checkGroup bool) Guard {
    return &AssignmentGuard{checkGroup: checkGroup}
}

func (g *AssignmentGuard) Validate(ctx context.Context, inst *instance.Instance, actor *actor.Actor) error {
    currentActor := inst.CurrentActor()
    if currentActor == "" {
        return errors.New("instance has no assigned actor")
    }

    // Verificar asignación directa
    if currentActor == actor.ID().String() {
        return nil
    }

    // Verificar pertenencia a grupo (si enabled)
    if g.checkGroup {
        if actor.BelongsToGroup(currentActor) {
            return nil
        }
    }

    return errors.New("instance is not assigned to actor")
}
```

### 2.4 Custom Reclassification Guard

```go
// internal/domain/workflow/guard_reclassify.go
package workflow

import (
    "context"
    "errors"
    "fmt"

    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/actor"
    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
)

// ReclassificationGuard verifica si se puede reclasificar
type ReclassificationGuard struct {
    targetType string
}

// NewReclassificationGuard crea un guard de reclasificación
func NewReclassificationGuard(targetType string) Guard {
    return &ReclassificationGuard{targetType: targetType}
}

func (g *ReclassificationGuard) Validate(ctx context.Context, inst *instance.Instance, actor *actor.Actor) error {
    // Obtener tipo actual
    currentTypeRaw, ok := inst.Data().Get("current_type")
    if !ok {
        return errors.New("instance does not have current_type")
    }

    currentType, ok := currentTypeRaw.(string)
    if !ok {
        return errors.New("current_type is not a string")
    }

    // No puede reclasificar al mismo tipo
    if currentType == g.targetType {
        return fmt.Errorf("cannot reclassify to same type: %s", currentType)
    }

    // Solo gestionadores senior pueden reclasificar
    if !actor.HasRole(actor.RoleGestionador) {
        return errors.New("only gestionador role can reclassify")
    }

    // Validación adicional: senior level
    if !actor.IsSenior() {
        return errors.New("only senior gestionadores can reclassify")
    }

    return nil
}
```

---

## 3. Domain Services

### 3.1 Escalation Service

```go
// internal/domain/escalation/service.go
package escalation

import (
    "context"
    "errors"

    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
    "github.com/google/uuid"
)

// Escalation representa un escalamiento
type Escalation struct {
    id             uuid.UUID
    instanceID     instance.ID
    departmentID   string
    reason         string
    escalatedBy    string
    escalatedAt    time.Time
    response       *string
    respondedBy    *string
    respondedAt    *time.Time
    status         Status
    isAuto         bool
}

type Status string

const (
    StatusPending   Status = "pending"
    StatusResponded Status = "responded"
    StatusClosed    Status = "closed"
    StatusCancelled Status = "cancelled"
)

// NewEscalation crea un nuevo escalamiento
func NewEscalation(
    instanceID instance.ID,
    departmentID string,
    reason string,
    escalatedBy string,
    isAuto bool,
) (*Escalation, error) {
    if departmentID == "" {
        return nil, errors.New("department_id is required")
    }
    if reason == "" {
        return nil, errors.New("reason is required")
    }

    return &Escalation{
        id:           uuid.New(),
        instanceID:   instanceID,
        departmentID: departmentID,
        reason:       reason,
        escalatedBy:  escalatedBy,
        escalatedAt:  time.Now(),
        status:       StatusPending,
        isAuto:       isAuto,
    }, nil
}

// Respond registra una respuesta al escalamiento
func (e *Escalation) Respond(response string, respondedBy string) error {
    if e.status != StatusPending {
        return errors.New("escalation is not pending")
    }
    if response == "" {
        return errors.New("response is required")
    }

    now := time.Now()
    e.response = &response
    e.respondedBy = &respondedBy
    e.respondedAt = &now
    e.status = StatusResponded

    return nil
}

// Repository port
type Repository interface {
    Save(ctx context.Context, escalation *Escalation) error
    FindByID(ctx context.Context, id uuid.UUID) (*Escalation, error)
    FindPendingByInstance(ctx context.Context, instanceID instance.ID) ([]*Escalation, error)
    FindPendingByDepartment(ctx context.Context, departmentID string) ([]*Escalation, error)
}
```

---

## 4. Application Layer - Use Cases

### 4.1 Escalate Use Case

```go
// internal/application/instance/escalate.go
package instance

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/escalation"
    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
    "github.com/LaFabric-LinkTIC/FlowEngine/pkg/ports"
)

type EscalateCommand struct {
    InstanceID   string
    DepartmentID string
    Reason       string
    ActorID      string
    IsAuto       bool
}

type EscalateResult struct {
    EscalationID string
    InstanceID   string
    SubState     instance.SubState
}

type EscalateUseCase struct {
    instanceRepo   instance.Repository
    escalationRepo escalation.Repository
    eventBus       event.Dispatcher
    locker         ports.Locker
    logger         ports.Logger
}

func NewEscalateUseCase(
    instanceRepo instance.Repository,
    escalationRepo escalation.Repository,
    eventBus event.Dispatcher,
    locker ports.Locker,
    logger ports.Logger,
) *EscalateUseCase {
    return &EscalateUseCase{
        instanceRepo:   instanceRepo,
        escalationRepo: escalationRepo,
        eventBus:       eventBus,
        locker:         locker,
        logger:         logger,
    }
}

func (uc *EscalateUseCase) Execute(ctx context.Context, cmd EscalateCommand) (*EscalateResult, error) {
    log := uc.logger.With(
        "use_case", "escalate",
        "instance_id", cmd.InstanceID,
        "department_id", cmd.DepartmentID,
        "actor", cmd.ActorID,
    )

    log.Info("executing escalate use case")

    // 1. Validar comando
    if err := uc.validateCommand(cmd); err != nil {
        return nil, fmt.Errorf("invalid command: %w", err)
    }

    // 2. Adquirir distributed lock
    lock, err := uc.locker.Lock(ctx, cmd.InstanceID, 30*time.Second)
    if err != nil {
        return nil, fmt.Errorf("failed to acquire lock: %w", err)
    }
    defer func() {
        if err := lock.Unlock(ctx); err != nil {
            log.Error("failed to release lock", "error", err)
        }
    }()

    // 3. Cargar instance
    instID := instance.ParseID(cmd.InstanceID)
    inst, err := uc.instanceRepo.FindByID(ctx, instID)
    if err != nil {
        return nil, fmt.Errorf("failed to load instance: %w", err)
    }

    // 4. Validar que está en estado apropiado
    if inst.CurrentState().ID() != "in_progress" {
        return nil, errors.New("can only escalate from in_progress state")
    }

    // 5. Crear registro de escalamiento
    esc, err := escalation.NewEscalation(
        inst.ID(),
        cmd.DepartmentID,
        cmd.Reason,
        cmd.ActorID,
        cmd.IsAuto,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create escalation: %w", err)
    }

    if err := uc.escalationRepo.Save(ctx, esc); err != nil {
        return nil, fmt.Errorf("failed to save escalation: %w", err)
    }

    // 6. Cambiar substate a escalated_awaiting_response
    if err := inst.SetSubState(instance.SubStateEscalatedAwaitingResponse); err != nil {
        return nil, fmt.Errorf("failed to set substate: %w", err)
    }

    // 7. Persistir cambios
    if err := uc.instanceRepo.Save(ctx, inst); err != nil {
        return nil, fmt.Errorf("failed to save instance: %w", err)
    }

    // 8. Publicar eventos de dominio
    domainEvents := inst.DomainEvents()
    for _, evt := range domainEvents {
        if err := uc.eventBus.Dispatch(ctx, evt); err != nil {
            log.Error("failed to dispatch event", "event", evt.Type(), "error", err)
        }
    }

    // 9. Publicar evento específico de escalamiento
    escalatedEvent := event.NewDocumentEscalated(
        inst.ID().String(),
        cmd.DepartmentID,
        cmd.Reason,
        cmd.ActorID,
        cmd.IsAuto,
        time.Now(),
    )

    if err := uc.eventBus.Dispatch(ctx, escalatedEvent); err != nil {
        log.Error("failed to dispatch escalated event", "error", err)
    }

    log.Info("escalation completed",
        "escalation_id", esc.ID().String(),
        "substate", inst.CurrentSubState(),
    )

    return &EscalateResult{
        EscalationID: esc.ID().String(),
        InstanceID:   inst.ID().String(),
        SubState:     inst.CurrentSubState(),
    }, nil
}

func (uc *EscalateUseCase) validateCommand(cmd EscalateCommand) error {
    if cmd.InstanceID == "" {
        return errors.New("instance_id is required")
    }
    if cmd.DepartmentID == "" {
        return errors.New("department_id is required")
    }
    if cmd.Reason == "" {
        return errors.New("reason is required")
    }
    if cmd.ActorID == "" && !cmd.IsAuto {
        return errors.New("actor_id is required for manual escalations")
    }
    return nil
}
```

### 4.2 Reject Use Case

```go
// internal/application/instance/reject.go
package instance

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
    "github.com/LaFabric-LinkTIC/FlowEngine/pkg/ports"
)

type RejectCommand struct {
    InstanceID string
    Reason     string
    Feedback   string
    ActorID    string
}

type RejectResult struct {
    InstanceID       string
    PreviousState    string
    CurrentState     string
    CurrentSubState  instance.SubState
    RejectionCount   int
}

type RejectUseCase struct {
    instanceRepo instance.Repository
    workflowRepo workflow.Repository
    eventBus     event.Dispatcher
    locker       ports.Locker
    logger       ports.Logger
}

func NewRejectUseCase(
    instanceRepo instance.Repository,
    workflowRepo workflow.Repository,
    eventBus event.Dispatcher,
    locker ports.Locker,
    logger ports.Logger,
) *RejectUseCase {
    return &RejectUseCase{
        instanceRepo: instanceRepo,
        workflowRepo: workflowRepo,
        eventBus:     eventBus,
        locker:       locker,
        logger:       logger,
    }
}

func (uc *RejectUseCase) Execute(ctx context.Context, cmd RejectCommand) (*RejectResult, error) {
    log := uc.logger.With(
        "use_case", "reject",
        "instance_id", cmd.InstanceID,
        "actor", cmd.ActorID,
    )

    log.Info("executing reject use case")

    // 1. Validar comando
    if err := uc.validateCommand(cmd); err != nil {
        return nil, fmt.Errorf("invalid command: %w", err)
    }

    // 2. Adquirir lock
    lock, err := uc.locker.Lock(ctx, cmd.InstanceID, 30*time.Second)
    if err != nil {
        return nil, fmt.Errorf("failed to acquire lock: %w", err)
    }
    defer lock.Unlock(ctx)

    // 3. Cargar instance
    instID := instance.ParseID(cmd.InstanceID)
    inst, err := uc.instanceRepo.FindByID(ctx, instID)
    if err != nil {
        return nil, fmt.Errorf("failed to load instance: %w", err)
    }

    // 4. Validar estado actual
    if inst.CurrentState().ID() != "in_review" {
        return nil, errors.New("can only reject from in_review state")
    }

    previousState := inst.CurrentState().ID()

    // 5. Cargar workflow
    wf, err := uc.workflowRepo.FindByID(ctx, inst.WorkflowID())
    if err != nil {
        return nil, fmt.Errorf("failed to load workflow: %w", err)
    }

    // 6. Agregar feedback a la instancia
    inst.AddFeedback(cmd.Reason, cmd.Feedback, cmd.ActorID)

    // 7. Ejecutar transición de rechazo (in_review → in_progress)
    targetSubState := instance.SubStateWorking
    if err := inst.TransitionWithSubState(ctx, wf, "reject", cmd.ActorID, &targetSubState); err != nil {
        return nil, fmt.Errorf("failed to execute transition: %w", err)
    }

    // 8. Persistir cambios
    if err := uc.instanceRepo.Save(ctx, inst); err != nil {
        return nil, fmt.Errorf("failed to save instance: %w", err)
    }

    // 9. Publicar eventos
    domainEvents := inst.DomainEvents()
    for _, evt := range domainEvents {
        if err := uc.eventBus.Dispatch(ctx, evt); err != nil {
            log.Error("failed to dispatch event", "event", evt.Type(), "error", err)
        }
    }

    // Evento específico de rechazo
    rejectedEvent := event.NewDocumentRejected(
        inst.ID().String(),
        cmd.Reason,
        cmd.Feedback,
        cmd.ActorID,
        time.Now(),
    )

    if err := uc.eventBus.Dispatch(ctx, rejectedEvent); err != nil {
        log.Error("failed to dispatch rejected event", "error", err)
    }

    // Obtener contador de rechazos
    rejectionCount := 0
    if count, ok := inst.Variables().Get("rejection_count"); ok {
        if intCount, ok := count.(int); ok {
            rejectionCount = intCount
        }
    }

    log.Info("rejection completed",
        "previous_state", previousState,
        "current_state", inst.CurrentState().ID(),
        "rejection_count", rejectionCount,
    )

    return &RejectResult{
        InstanceID:      inst.ID().String(),
        PreviousState:   previousState,
        CurrentState:    inst.CurrentState().ID(),
        CurrentSubState: inst.CurrentSubState(),
        RejectionCount:  rejectionCount,
    }, nil
}

func (uc *RejectUseCase) validateCommand(cmd RejectCommand) error {
    if cmd.InstanceID == "" {
        return errors.New("instance_id is required")
    }
    if cmd.Reason == "" {
        return errors.New("reason is required")
    }
    if cmd.Feedback == "" {
        return errors.New("feedback is required")
    }
    if cmd.ActorID == "" {
        return errors.New("actor_id is required")
    }
    return nil
}
```

---

## 5. Domain Events

### 5.1 SubState Changed Event

```go
// internal/domain/event/substate_changed.go
package event

import (
    "time"

    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
)

type SubStateChanged struct {
    instanceID      string
    state           string
    fromSubState    *instance.SubState
    toSubState      instance.SubState
    occurredAt      time.Time
}

func NewSubStateChanged(
    instanceID string,
    state string,
    fromSubState *instance.SubState,
    toSubState instance.SubState,
    occurredAt time.Time,
) DomainEvent {
    return &SubStateChanged{
        instanceID:   instanceID,
        state:        state,
        fromSubState: fromSubState,
        toSubState:   toSubState,
        occurredAt:   occurredAt,
    }
}

func (e *SubStateChanged) Type() string {
    return "substate.changed"
}

func (e *SubStateChanged) AggregateID() string {
    return e.instanceID
}

func (e *SubStateChanged) OccurredAt() time.Time {
    return e.occurredAt
}

func (e *SubStateChanged) Payload() map[string]interface{} {
    payload := map[string]interface{}{
        "instance_id": e.instanceID,
        "state":       e.state,
        "to_substate": e.toSubState.String(),
    }

    if e.fromSubState != nil {
        payload["from_substate"] = e.fromSubState.String()
    }

    return payload
}
```

### 5.2 Document Escalated Event

```go
// internal/domain/event/document_escalated.go
package event

import "time"

type DocumentEscalated struct {
    instanceID   string
    departmentID string
    reason       string
    escalatedBy  string
    isAuto       bool
    occurredAt   time.Time
}

func NewDocumentEscalated(
    instanceID string,
    departmentID string,
    reason string,
    escalatedBy string,
    isAuto bool,
    occurredAt time.Time,
) DomainEvent {
    return &DocumentEscalated{
        instanceID:   instanceID,
        departmentID: departmentID,
        reason:       reason,
        escalatedBy:  escalatedBy,
        isAuto:       isAuto,
        occurredAt:   occurredAt,
    }
}

func (e *DocumentEscalated) Type() string {
    return "document.escalated"
}

func (e *DocumentEscalated) AggregateID() string {
    return e.instanceID
}

func (e *DocumentEscalated) OccurredAt() time.Time {
    return e.occurredAt
}

func (e *DocumentEscalated) Payload() map[string]interface{} {
    return map[string]interface{}{
        "instance_id":   e.instanceID,
        "department_id": e.departmentID,
        "reason":        e.reason,
        "escalated_by":  e.escalatedBy,
        "is_auto":       e.isAuto,
    }
}
```

### 5.3 Document Rejected Event

```go
// internal/domain/event/document_rejected.go
package event

import "time"

type DocumentRejected struct {
    instanceID string
    reason     string
    feedback   string
    rejectedBy string
    occurredAt time.Time
}

func NewDocumentRejected(
    instanceID string,
    reason string,
    feedback string,
    rejectedBy string,
    occurredAt time.Time,
) DomainEvent {
    return &DocumentRejected{
        instanceID: instanceID,
        reason:     reason,
        feedback:   feedback,
        rejectedBy: rejectedBy,
        occurredAt: occurredAt,
    }
}

func (e *DocumentRejected) Type() string {
    return "document.rejected"
}

func (e *DocumentRejected) AggregateID() string {
    return e.instanceID
}

func (e *DocumentRejected) OccurredAt() time.Time {
    return e.occurredAt
}

func (e *DocumentRejected) Payload() map[string]interface{} {
    return map[string]interface{}{
        "instance_id": e.instanceID,
        "reason":      e.reason,
        "feedback":    e.feedback,
        "rejected_by": e.rejectedBy,
    }
}
```

---

## 6. HTTP Handlers

### 6.1 Escalate Handler

```go
// internal/infrastructure/http/rest/handlers/escalate_handler.go
package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/LaFabric-LinkTIC/FlowEngine/internal/application/instance"
)

type EscalateHandler struct {
    escalateUseCase *instance.EscalateUseCase
    logger          ports.Logger
}

func NewEscalateHandler(escalateUseCase *instance.EscalateUseCase, logger ports.Logger) *EscalateHandler {
    return &EscalateHandler{
        escalateUseCase: escalateUseCase,
        logger:          logger,
    }
}

type EscalateRequest struct {
    DepartmentID string `json:"department_id" binding:"required"`
    Reason       string `json:"reason" binding:"required"`
}

type EscalateResponse struct {
    EscalationID string `json:"escalation_id"`
    InstanceID   string `json:"instance_id"`
    SubState     string `json:"sub_state"`
}

// POST /api/v1/instances/:id/escalate
func (h *EscalateHandler) Escalate(c *gin.Context) {
    instanceID := c.Param("id")
    actorID := c.GetString("user_id")  // Desde JWT middleware

    var req EscalateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
        return
    }

    cmd := instance.EscalateCommand{
        InstanceID:   instanceID,
        DepartmentID: req.DepartmentID,
        Reason:       req.Reason,
        ActorID:      actorID,
        IsAuto:       false,
    }

    result, err := h.escalateUseCase.Execute(c.Request.Context(), cmd)
    if err != nil {
        h.handleError(c, err)
        return
    }

    c.JSON(http.StatusOK, EscalateResponse{
        EscalationID: result.EscalationID,
        InstanceID:   result.InstanceID,
        SubState:     result.SubState.String(),
    })
}

func (h *EscalateHandler) handleError(c *gin.Context, err error) {
    // Similar al InstanceHandler.handleError del diseño original
    h.logger.Error("escalate error", "error", err)
    c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
}
```

---

## 7. Tests

### 7.1 Unit Test - SubState

```go
// internal/domain/instance/substate_test.go
package instance_test

import (
    "testing"

    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
    "github.com/stretchr/testify/assert"
)

func TestNewSubState_Valid(t *testing.T) {
    tests := []struct {
        name     string
        value    string
        expected instance.SubState
        wantErr  bool
    }{
        {
            name:     "empty substate",
            value:    "",
            expected: instance.SubStateEmpty,
            wantErr:  false,
        },
        {
            name:     "working substate",
            value:    "working",
            expected: instance.SubStateWorking,
            wantErr:  false,
        },
        {
            name:     "escalated_awaiting_response",
            value:    "escalated_awaiting_response",
            expected: instance.SubStateEscalatedAwaitingResponse,
            wantErr:  false,
        },
        {
            name:     "invalid substate",
            value:    "invalid_substate",
            expected: instance.SubStateEmpty,
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ss, err := instance.NewSubState(tt.value)

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, ss)
            }
        })
    }
}

func TestSubState_IsEmpty(t *testing.T) {
    assert.True(t, instance.SubStateEmpty.IsEmpty())
    assert.False(t, instance.SubStateWorking.IsEmpty())
}
```

### 7.2 Unit Test - Escalate Use Case

```go
// internal/application/instance/escalate_test.go
package instance_test

import (
    "context"
    "testing"
    "time"

    "github.com/LaFabric-LinkTIC/FlowEngine/internal/application/instance"
    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/escalation"
    "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
    "github.com/LaFabric-LinkTIC/FlowEngine/mocks"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestEscalateUseCase_Execute_Success(t *testing.T) {
    // Setup mocks
    mockInstanceRepo := mocks.NewInstanceRepository(t)
    mockEscalationRepo := mocks.NewEscalationRepository(t)
    mockEventBus := mocks.NewEventDispatcher(t)
    mockLocker := mocks.NewLocker(t)
    mockLogger := mocks.NewLogger(t)

    // Create use case
    useCase := instance.NewEscalateUseCase(
        mockInstanceRepo,
        mockEscalationRepo,
        mockEventBus,
        mockLocker,
        mockLogger,
    )

    // Create test instance
    inst := createTestInstance(t, "in_progress", instance.SubStateWorking)

    // Mock expectations
    mockLock := mocks.NewLock(t)
    mockLocker.On("Lock", mock.Anything, inst.ID().String(), 30*time.Second).
        Return(mockLock, nil)
    mockLock.On("Unlock", mock.Anything).Return(nil)

    mockInstanceRepo.On("FindByID", mock.Anything, inst.ID()).
        Return(inst, nil)

    mockEscalationRepo.On("Save", mock.Anything, mock.AnythingOfType("*escalation.Escalation")).
        Return(nil)

    mockInstanceRepo.On("Save", mock.Anything, inst).
        Return(nil)

    mockEventBus.On("Dispatch", mock.Anything, mock.Anything).
        Return(nil)

    mockLogger.On("With", mock.Anything, mock.Anything).Return(mockLogger)
    mockLogger.On("Info", mock.Anything, mock.Anything)

    // Execute
    cmd := instance.EscalateCommand{
        InstanceID:   inst.ID().String(),
        DepartmentID: "legal",
        Reason:       "Requiere revisión legal",
        ActorID:      "user-001",
        IsAuto:       false,
    }

    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.NotEmpty(t, result.EscalationID)
    assert.Equal(t, inst.ID().String(), result.InstanceID)
    assert.Equal(t, instance.SubStateEscalatedAwaitingResponse, result.SubState)

    // Verify mocks
    mockInstanceRepo.AssertExpectations(t)
    mockEscalationRepo.AssertExpectations(t)
    mockEventBus.AssertExpectations(t)
    mockLocker.AssertExpectations(t)
}
```

---

**Fin de la Implementación en Go**
