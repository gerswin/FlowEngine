# FlowEngine - Documento de Diseño Técnico

> **Ultima actualizacion**: 2026-03-21

## 1. Overview del Sistema

### 1.1 Propósito

**FlowEngine** es una librería/servicio de orquestación de workflows con máquina de estados finitos (FSM) que proporciona:

#### Implementado
- ✅ **Persistencia híbrida** (Redis cache + PostgreSQL + In-Memory para desarrollo)
- ✅ **Múltiples instancias** de workflows ejecutándose en paralelo
- ✅ **Subprocesos jerárquicos** (procesos padre-hijo, clonación)
- ✅ **Workflows configurables** vía JSON:API y YAML
- ✅ **REST API completa** con JSON:API 1.0, JWT auth, paginación DB-level
- ✅ **Motor de Guards** — 14 guards built-in que validan condiciones pre-transición
- ✅ **Motor de Actions** — 10 actions built-in que ejecutan efectos post-transición
- ✅ **Required Data** — Campos obligatorios por transición, validados automáticamente
- ✅ **Sistema de eventos** — MultiDispatcher (InMemory + Log + Webhook)
- ✅ **Webhook delivery** — Entrega asíncrona con HMAC signing y reintentos
- ✅ **Timers y scheduler** con retry policy (backoff exponencial)
- ✅ **Subestados jerárquicos** para tracking detallado (R17)
- ✅ **Metadata extendida** en transiciones con validación (R23)
- ✅ **Arquitectura hexagonal** con código limpio y testeable
- ✅ **Mapeo de errores de dominio** a HTTP status codes (JSON:API)
- ✅ **331 tests** (domain, handler, webhook dispatcher)

#### Planificado (no implementado aún)
- ⬚ **RBAC** — JWT auth existe pero no se validan roles en endpoints/transiciones
- ⬚ **Sistema de actores** — Dominio actor/ está vacío
- ⬚ **Prometheus metrics** / OpenTelemetry tracing
- ⬚ **RabbitMQ / Kafka** dispatchers (MultiDispatcher listo para agregarlos)
- ⬚ **Config management** centralizado
- ⬚ **Diagrama Mermaid** del workflow (R1.2.4)

### 1.2 Tecnologías Principales

- **Lenguaje**: Go 1.24+
- **Framework HTTP**: Gin
- **Base de datos**: PostgreSQL 15+
- **Caché**: Redis 7+
- **Event System**: MultiDispatcher (InMemory + LogDispatcher + WebhookDispatcher)
- **Testing**: testify, mockery
- **Logging**: slog/JSON estructurado
- **Deployment**: Docker, Docker Compose

### 1.3 Casos de Uso Principales

1. **Flujo de Radicación de Documentos**
   - 6 etapas: Radicar → Asignar → Gestionar → Revisar → Aprobar → Enviar
   - Múltiples actores con roles diferentes
   - Escalamientos automáticos con timeout de 2 horas
   - Rechazos que regresan a estados anteriores

2. **Workflows Configurables Dinámicos**
   - Definir workflows vía YAML/JSON sin código
   - Validadores y acciones customizables
   - Hot-reload de configuraciones

3. **Orquestación de Procesos Complejos**
   - Subprocesos con espera/join
   - Contexto compartido entre padre-hijo
   - Manejo de errores y compensación

4. **Auditoría y Compliance**
   - Historial completo de transiciones
   - Tracking de actores y timestamps
   - Eventos de dominio para integraciones

5. **Alta Concurrencia**
   - Optimistic locking para bajo contention
   - Distributed locks para operaciones críticas
   - 10K+ transiciones/segundo

### 1.4 Requerimientos Especiales

- **Performance**: Cache en Redis con 99% hit rate, sub-segundo latency
- **Consistencia**: Optimistic locking con versionado
- **Escalabilidad**: Stateless API servers, horizontal scaling
- **Observabilidad**: Logging estructurado, métricas Prometheus, tracing distribuido
- **Testabilidad**: >80% code coverage, arquitectura hexagonal

---

## 2. Arquitectura del Sistema

### 2.1 Diagrama de Arquitectura (Hexagonal)

```mermaid
graph TB
    subgraph "External Clients"
        HTTP[HTTP REST Clients / Bruno]
        CLI[CLI Tools / Emulator]
    end

    subgraph "Infrastructure Layer"
        subgraph "HTTP Adapters"
            GIN[Gin Router + JSON:API]
            HAND[Handlers: Workflow, Instance]
            MW[Middleware: JWT Auth, CORS, Logger, RequestID]
        end

        subgraph "Persistence Adapters"
            PG[(PostgreSQL + Cache)]
            MEM[(In-Memory Repos)]
        end

        subgraph "Messaging Adapters"
            MULTI[MultiDispatcher]
            LOG_D[LogDispatcher]
            WH_D[WebhookDispatcher]
        end

        subgraph "Parser"
            YAML[YAML Parser]
        end

        subgraph "Scheduler"
            TIMER_W[Timer Worker + Retry Policy]
        end
    end

    subgraph "Application Layer"
        subgraph "Use Cases"
            UC1[Create/Get/List Workflow]
            UC2[Create/Get/List Instance]
            UC3[Transition Instance]
            UC4[Clone Instance]
            UC5[Transition History]
        end
    end

    subgraph "Domain Layer"
        subgraph "Aggregates"
            WF[Workflow: States, Events, Guards, Actions, RequiredData]
            INST[Instance: Data, Variables, Transitions, SubStates]
        end

        subgraph "Engine"
            ENG[Guard/Action Engine: 14 guards + 10 actions]
        end

        subgraph "Value Objects"
            STATE[State, Event, Version, GuardConfig, ActionConfig]
            VO[ID, Timestamp, ListQuery, TransitionMetadata]
        end

        subgraph "Domain Events"
            DE[InstanceCreated, StateChanged, SubStateChanged, Completed, Canceled, Failed, Paused, Resumed]
        end
    end

    HTTP --> GIN
    CLI --> GIN

    GIN --> MW --> HAND
    HAND --> UC1 & UC2 & UC3 & UC4 & UC5

    UC3 --> ENG
    UC1 & UC2 & UC3 & UC4 --> PG
    UC1 & UC2 & UC3 & UC4 --> MEM

    UC3 --> MULTI
    MULTI --> LOG_D
    MULTI --> WH_D

    UC1 --> WF
    UC3 --> INST & WF
    ENG --> INST

    YAML --> WF

    TIMER_W --> UC3
```

### 2.1.1 Flujo de Ejecución de una Transición

```
Request POST /instances/:id/transitions
    │
    ▼
┌─ JWT Auth Middleware ─┐
│  Validate Bearer token │
└───────────┬───────────┘
            ▼
┌─ TransitionInstanceUseCase ─────────────────┐
│  1. Parse IDs + validate command             │
│  2. Load instance from repo                  │
│  3. Load workflow from repo                  │
│  4. Validate event exists + can transition   │
│  5. Apply transition data (cmd.Data)         │
│  6. ► Evaluate Guards (fail → 400)           │
│  7. ► Validate RequiredData (fail → 400)     │
│  8. Execute domain transition                │
│  9. ► Execute Actions (post-transition)      │
│  10. Save instance                           │
│  11. Dispatch domain events                  │
│      → MultiDispatcher                       │
│        → InMemoryDispatcher (collect)        │
│        → LogDispatcher (audit log)           │
│        → WebhookDispatcher (async HTTP POST) │
└──────────────────────────────────────────────┘
```

### 2.2 Diagrama de Flujo de una Transición

```mermaid
sequenceDiagram
    participant Client
    participant Handler
    participant UseCase
    participant Engine
    participant Repository
    participant Instance
    participant EventBus

    Client->>Handler: POST /instances/{id}/transitions
    Note over Handler: JWT Auth Middleware validates token
    Handler->>UseCase: Execute(TransitionCommand)

    UseCase->>Repository: FindByID(instanceID)
    Note over Repository: Cache read-through (Redis → PostgreSQL)
    Repository-->>UseCase: Instance aggregate

    UseCase->>Repository: FindByID(workflowID)
    Repository-->>UseCase: Workflow (with guards, actions, required_data)

    UseCase->>Instance: UpdateData(cmd.Data)
    Note over Instance: Apply transition data first

    UseCase->>Engine: EvaluateGuards(instance, guards)
    alt Guard fails
        Engine-->>UseCase: Error (field_equals, has_role, etc.)
        UseCase-->>Handler: 400 INVALID_INPUT
        Handler-->>Client: JSON:API error response
    end
    Engine-->>UseCase: Guards passed

    UseCase->>Instance: Transition(toState, event, actor, metadata, requiredData)
    Note over Instance: Validate required_data fields
    Instance->>Instance: Change state + increment version
    Instance->>Instance: Record domain events
    Instance-->>UseCase: Success

    UseCase->>Engine: ExecuteActions(instance, actions)
    Note over Engine: set_metadata, increment_field, mark_as_approved, etc.

    UseCase->>Repository: Save(instance)
    Note over Repository: Optimistic locking (version check)
    Repository-->>UseCase: OK

    UseCase->>EventBus: DispatchBatch(domainEvents)
    Note over EventBus: MultiDispatcher → Log + Webhook (async)

    UseCase-->>Handler: TransitionResult
    Handler-->>Client: 200 OK (JSON:API resource)
```

### 2.3 Diagrama de Datos (PostgreSQL)

```mermaid
erDiagram
    WORKFLOWS ||--o{ WORKFLOW_INSTANCES : defines
    WORKFLOWS ||--o{ WORKFLOWS : "template-derived"
    WORKFLOW_INSTANCES ||--o{ WORKFLOW_INSTANCES : "parent-child"
    WORKFLOW_INSTANCES ||--o{ WORKFLOW_TRANSITIONS : has
    WORKFLOW_INSTANCES ||--o{ WORKFLOW_TIMERS : has
    WORKFLOW_INSTANCES ||--o{ WORKFLOW_ESCALATIONS : has
    WORKFLOWS ||--o{ WEBHOOKS : configures

    WORKFLOWS {
        varchar id PK
        varchar name
        text description
        varchar version
        jsonb config
        boolean is_template
        uuid template_id FK
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    WORKFLOW_INSTANCES {
        uuid id PK
        varchar workflow_id FK
        uuid parent_id FK
        varchar current_state
        varchar previous_state
        varchar current_sub_state
        varchar previous_sub_state
        bigint version
        varchar status
        jsonb data
        jsonb variables
        varchar current_actor
        varchar current_role
        timestamp created_at
        timestamp updated_at
        timestamp completed_at
        varchar locked_by
        timestamp locked_at
        timestamp lock_expires_at
    }

    WORKFLOW_TRANSITIONS {
        uuid id PK
        uuid instance_id FK
        varchar event
        varchar from_state
        varchar to_state
        varchar from_sub_state
        varchar to_sub_state
        varchar actor
        varchar actor_role
        jsonb data
        text reason
        text feedback
        jsonb metadata
        integer duration_ms
        timestamp created_at
    }

    WORKFLOW_TIMERS {
        uuid id PK
        uuid instance_id FK
        varchar state
        varchar event_on_timeout
        timestamp created_at
        timestamp expires_at
        timestamp fired_at
    }

    WORKFLOW_ESCALATIONS {
        uuid id PK
        uuid instance_id FK
        varchar department_id
        text reason
        varchar escalated_by
        timestamp escalated_at
        varchar status
        text response
        varchar responded_by
        timestamp responded_at
        varchar closed_by
        timestamp closed_at
    }

    WEBHOOKS {
        uuid id PK
        varchar workflow_id FK
        text url
        text[] events
        varchar secret
        jsonb headers
        jsonb retry_config
        boolean active
        timestamp created_at
        timestamp updated_at
    }

    EXTERNAL_EVENTS {
        uuid id PK
        uuid instance_id FK
        varchar event_type
        jsonb payload
        timestamp processed_at
        text error_message
        timestamp created_at
    }
```

---

## 3. Componentes Principales

### 3.1 Domain Layer (Núcleo de Negocio)

#### 3.1.1 Workflow Aggregate

**Responsabilidades**:
- Definir estados y transiciones permitidas
- Validar coherencia del workflow
- Proveer métodos para verificar transiciones válidas

**Invariantes**:
- Debe tener al menos un estado inicial
- Todos los eventos deben referenciar estados existentes
- No puede haber ciclos infinitos obligatorios

**Interfaz Pública**:
```go
type Workflow struct {
    id           ID
    name         Name
    version      Version
    initialState State
    states       []State
    events       []Event
}

func NewWorkflow(name Name, initialState State) (*Workflow, error)
func (w *Workflow) AddState(state State) error
func (w *Workflow) AddEvent(event Event) error
func (w *Workflow) CanTransition(from State, event Event) bool
func (w *Workflow) FindEvent(name string) (Event, error)
```

#### 3.1.2 Instance Aggregate

**Responsabilidades**:
- Ejecutar transiciones de estado
- Mantener historial de transiciones
- Generar eventos de dominio
- Gestionar lifecycle (pause, resume, cancel)
- Validar permisos de actores

**Invariantes**:
- Version debe incrementarse en cada cambio
- No se puede transicionar si status != Running
- Estados finales no permiten más transiciones
- Historial es append-only (inmutable)

**Interfaz Pública**:
```go
type Instance struct {
    id               ID
    workflowID       workflow.ID
    currentState     workflow.State
    currentSubState  *workflow.SubState      // R17: Subestados
    previousSubState *workflow.SubState      // R17: Subestados
    version          Version
    status           Status
    data             Data
    variables        Variables
    history          []Transition
    domainEvents     []event.DomainEvent
}

func NewInstance(workflowID workflow.ID, initialState workflow.State, data Data) (*Instance, error)
func (i *Instance) Transition(ctx context.Context, wf *workflow.Workflow, eventName string, actorID string) error
func (i *Instance) TransitionWithMetadata(ctx context.Context, wf *workflow.Workflow, eventName string, actorID string, reason, feedback string, metadata map[string]interface{}) error  // R23
func (i *Instance) Pause() error
func (i *Instance) Resume() error
func (i *Instance) Cancel() error
func (i *Instance) SetVariable(key string, value interface{})
func (i *Instance) GetVariable(key string) (interface{}, bool)
func (i *Instance) DomainEvents() []event.DomainEvent

// R17: Métodos de subestados
func (i *Instance) TransitionSubState(ctx context.Context, wf *workflow.Workflow, newSubState workflow.SubState) error
func (i *Instance) CurrentSubState() *workflow.SubState
func (i *Instance) PreviousSubState() *workflow.SubState

// R19: Reclasificación
func (i *Instance) Reclassify(newType string, reason string, actor string) error
```

#### 3.1.3 Escalation Aggregate (R18)

**Responsabilidades**:
- Gestionar el ciclo de vida de un escalamiento
- Validar transiciones de estado
- Registrar respuestas y cierres
- Generar eventos de dominio

**Invariantes**:
- Solo escalamientos "pending" pueden ser cancelados
- Una vez cerrado, no se puede reabrir
- Respuesta es obligatoria para estado "responded"
- DepartmentId debe ser válido

**Interfaz Pública**:
```go
type Escalation struct {
    id           ID
    instanceID   instance.ID
    departmentID DepartmentID
    reason       string
    escalatedBy  string
    escalatedAt  time.Time
    status       EscalationStatus
    response     *string
    respondedBy  *string
    respondedAt  *time.Time
    closedBy     *string
    closedAt     *time.Time
    domainEvents []event.DomainEvent
}

func NewEscalation(instanceID instance.ID, departmentID DepartmentID, reason string, escalatedBy string) (*Escalation, error)
func (e *Escalation) Reply(response string, respondedBy string) error
func (e *Escalation) Close(closedBy string) error
func (e *Escalation) Cancel() error
func (e *Escalation) Status() EscalationStatus
func (e *Escalation) CanCancel() bool
func (e *Escalation) DomainEvents() []event.DomainEvent
```

#### 3.1.4 Value Objects

**State**:
```go
type State struct {
    id          string
    name        string
    description string
    timeout     *time.Duration
    onTimeout   *string
    isFinal     bool
}

func NewState(id, name string) (State, error)
func (s State) WithTimeout(timeout time.Duration, onTimeout string) State
func (s State) AsFinal() State
func (s State) Equals(other State) bool
func (s State) Validate() error
```

**Version** (para optimistic locking):
```go
type Version struct {
    value int64
}

func NewVersion() Version
func (v Version) Increment() Version
func (v Version) Value() int64
func (v Version) Equals(other Version) bool
```

**Data** (datos del workflow):
```go
type Data struct {
    values map[string]interface{}
}

func NewData() Data
func NewDataFromMap(m map[string]interface{}) Data
func (d Data) Get(key string) (interface{}, bool)
func (d Data) Set(key string, value interface{})
func (d Data) ToMap() map[string]interface{}
```

**SubState** (R17 - subestados):
```go
type SubState struct {
    id          string
    name        string
    description string
}

func NewSubState(id, name string) (SubState, error)
func (s SubState) ID() string
func (s SubState) Name() string
func (s SubState) Equals(other SubState) bool
func (s SubState) Validate() error
```

**TransitionMetadata** (R23 - metadata extendida):
```go
type TransitionMetadata struct {
    reason   string
    feedback string
    metadata map[string]interface{}
}

func NewTransitionMetadata(reason, feedback string, metadata map[string]interface{}) TransitionMetadata
func (tm TransitionMetadata) Reason() string
func (tm TransitionMetadata) Feedback() string
func (tm TransitionMetadata) Metadata() map[string]interface{}
func (tm TransitionMetadata) Validate(schema *MetadataSchema) error
```

**EscalationStatus** (R18 - escalamientos):
```go
type EscalationStatus string

const (
    EscalationPending   EscalationStatus = "pending"
    EscalationResponded EscalationStatus = "responded"
    EscalationClosed    EscalationStatus = "closed"
    EscalationCanceled  EscalationStatus = "canceled"
)

func (es EscalationStatus) IsValid() bool
func (es EscalationStatus) CanTransitionTo(newStatus EscalationStatus) bool
```

### 3.2 Application Layer (Casos de Uso)

#### 3.2.1 TriggerEventUseCase

**Responsabilidades**:
- Validar comando de entrada
- Adquirir lock distribuido
- Cargar instancia y workflow
- Ejecutar transición (domain logic)
- Persistir cambios con optimistic locking
- Publicar eventos de dominio

**Flujo**:
```go
type TriggerEventUseCase struct {
    instanceRepo  instance.Repository
    workflowRepo  workflow.Repository
    eventBus      event.Dispatcher
    locker        ports.Locker
    logger        ports.Logger
}

type TriggerEventCommand struct {
    InstanceID string
    EventName  string
    ActorID    string
    Data       map[string]interface{}
}

type TriggerEventResult struct {
    InstanceID    string
    PreviousState string
    CurrentState  string
    Version       int64
}

func (uc *TriggerEventUseCase) Execute(ctx context.Context, cmd TriggerEventCommand) (*TriggerEventResult, error) {
    // 1. Validar command
    // 2. Adquirir lock
    // 3. Cargar instance
    // 4. Cargar workflow
    // 5. Ejecutar transición
    // 6. Persistir con version check
    // 7. Publicar domain events
    // 8. Release lock
    // 9. Retornar resultado
}
```

#### 3.2.2 CreateInstanceUseCase

```go
type CreateInstanceCommand struct {
    WorkflowID string
    ActorID    string
    ActorRole  string
    Data       map[string]interface{}
}

type CreateInstanceResult struct {
    InstanceID   string
    InitialState string
    CreatedAt    time.Time
}

func (uc *CreateInstanceUseCase) Execute(ctx context.Context, cmd CreateInstanceCommand) (*CreateInstanceResult, error)
```

#### 3.2.3 QueryInstancesUseCase

```go
type QueryInstancesQuery struct {
    WorkflowID string
    States     []string
    Actors     []string
    FromDate   *time.Time
    ToDate     *time.Time
    Status     *string
    Limit      int
    Offset     int
}

type QueryInstancesResult struct {
    Instances []*InstanceDTO
    Total     int
    Limit     int
    Offset    int
}

func (uc *QueryInstancesUseCase) Execute(ctx context.Context, query QueryInstancesQuery) (*QueryInstancesResult, error)
```

#### 3.2.4 EscalateInstanceUseCase (R18)

```go
type EscalateInstanceCommand struct {
    InstanceID   string
    DepartmentID string
    Reason       string
    ActorID      string
}

type EscalateInstanceResult struct {
    EscalationID string
    InstanceID   string
    DepartmentID string
    Status       string
    EscalatedAt  time.Time
}

func (uc *EscalateInstanceUseCase) Execute(ctx context.Context, cmd EscalateInstanceCommand) (*EscalateInstanceResult, error)
// 1. Validar que la instancia exista y permita escalamientos
// 2. Validar permisos del actor
// 3. Crear agregado Escalation
// 4. Cambiar subestado de la instancia
// 5. Persistir escalamiento y actualizar instancia
// 6. Publicar evento DocumentEscalated
```

#### 3.2.5 ReplyEscalationUseCase (R18)

```go
type ReplyEscalationCommand struct {
    EscalationID string
    Response     string
    RespondedBy  string
}

type ReplyEscalationResult struct {
    EscalationID string
    Status       string
    RespondedAt  time.Time
}

func (uc *ReplyEscalationUseCase) Execute(ctx context.Context, cmd ReplyEscalationCommand) (*ReplyEscalationResult, error)
```

#### 3.2.6 ReclassifyInstanceUseCase (R19)

```go
type ReclassifyInstanceCommand struct {
    InstanceID string
    NewType    string
    Reason     string
    ActorID    string
}

type ReclassifyInstanceResult struct {
    InstanceID   string
    FromType     string
    ToType       string
    CurrentState string
    Version      int64
}

func (uc *ReclassifyInstanceUseCase) Execute(ctx context.Context, cmd ReclassifyInstanceCommand) (*ReclassifyInstanceResult, error)
// 1. Validar que la instancia permita reclasificación
// 2. Validar permisos del actor (requires_senior?)
// 3. Ejecutar reclasificación en el agregado
// 4. Persistir cambios
// 5. Publicar evento DocumentReclassified
```

#### 3.2.7 CloneWorkflowFromTemplateUseCase (R21)

```go
type CloneWorkflowCommand struct {
    TemplateID  string
    NewID       *string
    Name        string
    Description string
    Overrides   map[string]interface{}
}

type CloneWorkflowResult struct {
    WorkflowID string
    Name       string
    TemplateID string
}

func (uc *CloneWorkflowFromTemplateUseCase) Execute(ctx context.Context, cmd CloneWorkflowCommand) (*CloneWorkflowResult, error)
// 1. Cargar template desde repositorio
// 2. Validar que sea template
// 3. Clonar configuración
// 4. Aplicar overrides
// 5. Validar workflow resultante
// 6. Persistir nuevo workflow
```

#### 3.2.8 ImportWorkflowUseCase (R22)

```go
type ImportWorkflowCommand struct {
    Content       []byte
    Format        string // "yaml" | "json"
    Mode          string // "create" | "update" | "force"
    GenerateNewID bool
}

type ImportWorkflowResult struct {
    WorkflowID string
    Name       string
    Version    string
    Created    bool
}

func (uc *ImportWorkflowUseCase) Execute(ctx context.Context, cmd ImportWorkflowCommand) (*ImportWorkflowResult, error)
// 1. Parsear YAML/JSON
// 2. Validar schema version y compatibilidad
// 3. Validar checksum
// 4. Validar workflow configuration
// 5. Verificar si existe (según mode)
// 6. Persistir workflow
```

### 3.3 Infrastructure Layer (Adaptadores)

#### 3.3.1 PostgreSQL Repository

**Implementa**: `instance.Repository`, `workflow.Repository`

**Responsabilidades**:
- Persistencia durable en PostgreSQL
- Mapeo domain ↔ DB models
- Optimistic locking con version check
- Queries complejas con filtros
- Transacciones

**Schema principal**:
```sql
CREATE TABLE workflow_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES workflow_instances(id),

    current_state VARCHAR(100) NOT NULL,
    previous_state VARCHAR(100),
    version BIGINT NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'running',

    data JSONB,
    variables JSONB,

    current_actor VARCHAR(255),
    current_role VARCHAR(100),

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,

    locked_by VARCHAR(255),
    locked_at TIMESTAMP,
    lock_expires_at TIMESTAMP
);

CREATE INDEX idx_instances_workflow ON workflow_instances(workflow_id);
CREATE INDEX idx_instances_state ON workflow_instances(current_state);
CREATE INDEX idx_instances_status ON workflow_instances(status);
CREATE INDEX idx_instances_actor ON workflow_instances(current_actor);
CREATE INDEX idx_instances_version ON workflow_instances(id, version);
```

**Implementación de SaveWithVersion** (optimistic locking):
```go
func (r *PostgresInstanceRepository) SaveWithVersion(
    ctx context.Context,
    inst *instance.Instance,
    expectedVersion instance.Version,
) error {
    model := r.mapper.ToModel(inst)

    query := `
        UPDATE workflow_instances SET
            current_state = $1,
            previous_state = $2,
            version = $3,
            status = $4,
            data = $5,
            variables = $6,
            updated_at = $7,
            completed_at = $8
        WHERE id = $9 AND version = $10
    `

    result, err := r.db.ExecContext(ctx, query,
        model.CurrentState,
        model.PreviousState,
        model.Version,
        model.Status,
        model.Data,
        model.Variables,
        model.UpdatedAt,
        model.CompletedAt,
        model.ID,
        expectedVersion.Value(),
    )

    if err != nil {
        return fmt.Errorf("failed to update: %w", err)
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        return instance.ErrVersionConflict
    }

    return nil
}
```

#### 3.3.2 Redis Cache

**Implementa**: `Cache` interface

**Responsabilidades**:
- Cache L1 para instancias activas
- Distributed locking
- TTL management
- Invalidación selectiva

**Implementación**:
```go
type InstanceCache struct {
    client     *redis.Client
    ttl        time.Duration
    serializer *Serializer
}

func (c *InstanceCache) Get(ctx context.Context, id instance.ID) (*instance.Instance, error) {
    key := fmt.Sprintf("instance:%s", id.String())

    data, err := c.client.Get(ctx, key).Bytes()
    if err == redis.Nil {
        return nil, instance.ErrNotFound
    }
    if err != nil {
        return nil, err
    }

    return c.serializer.DeserializeInstance(data)
}

func (c *InstanceCache) Set(ctx context.Context, inst *instance.Instance) error {
    key := fmt.Sprintf("instance:%s", inst.ID().String())
    data, err := c.serializer.SerializeInstance(inst)
    if err != nil {
        return err
    }

    return c.client.Set(ctx, key, data, c.ttl).Err()
}
```

**Distributed Lock** (Redis):
```go
type DistributedLocker struct {
    client *redis.Client
}

func (l *DistributedLocker) Lock(ctx context.Context, key string, ttl time.Duration) (*Lock, error) {
    lockKey := fmt.Sprintf("lock:%s", key)
    lockValue := uuid.New().String()

    // Redis SET NX (set if not exists)
    ok, err := l.client.SetNX(ctx, lockKey, lockValue, ttl).Result()
    if err != nil {
        return nil, err
    }
    if !ok {
        return nil, ErrLockAlreadyHeld
    }

    return &Lock{
        key:    lockKey,
        value:  lockValue,
        client: l.client,
    }, nil
}

func (l *Lock) Unlock(ctx context.Context) error {
    // Lua script para atomic check-and-delete
    script := `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        else
            return 0
        end
    `

    return l.client.Eval(ctx, script, []string{l.key}, l.value).Err()
}
```

#### 3.3.3 Hybrid Repository

**Combina**: Redis (cache) + PostgreSQL (durable)

**Estrategia**:
```go
type HybridInstanceRepository struct {
    cache   *redis.InstanceCache
    durable *postgres.InstanceRepository
    config  HybridConfig
}

type HybridConfig struct {
    CacheTTL      time.Duration
    WriteThrough  bool  // Escribir a cache y DB simultáneamente
    ReadThrough   bool  // Poblar cache en miss
    AsyncWrite    bool  // Escribir a DB async
}

func (r *HybridInstanceRepository) FindByID(ctx context.Context, id instance.ID) (*instance.Instance, error) {
    // 1. Try cache first (Redis)
    inst, err := r.cache.Get(ctx, id)
    if err == nil {
        return inst, nil // Cache hit!
    }

    // 2. Cache miss: read from DB
    inst, err = r.durable.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // 3. Populate cache (read-through)
    if r.config.ReadThrough {
        _ = r.cache.Set(ctx, inst)
    }

    return inst, nil
}

func (r *HybridInstanceRepository) Save(ctx context.Context, inst *instance.Instance) error {
    // 1. Write to cache (fast)
    if err := r.cache.Set(ctx, inst); err != nil {
        return err
    }

    // 2. Write to DB
    if r.config.AsyncWrite {
        go r.asyncWrite(ctx, inst)
        return nil
    }

    // Sync write
    if err := r.durable.Save(ctx, inst); err != nil {
        r.cache.Delete(ctx, inst.ID()) // Rollback cache
        return err
    }

    return nil
}
```

#### 3.3.4 HTTP Handlers

**InstanceHandler**:
```go
type InstanceHandler struct {
    createUseCase  *application.CreateInstanceUseCase
    triggerUseCase *application.TriggerEventUseCase
    queryUseCase   *application.QueryInstancesUseCase
    logger         ports.Logger
}

// POST /api/v1/instances/{id}/events
func (h *InstanceHandler) TriggerEvent(c *gin.Context) {
    instanceID := c.Param("id")

    var req TriggerEventRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse{Error: "Invalid request"})
        return
    }

    cmd := application.TriggerEventCommand{
        InstanceID: instanceID,
        EventName:  req.Event,
        ActorID:    req.Actor,
        Data:       req.Data,
    }

    result, err := h.triggerUseCase.Execute(c.Request.Context(), cmd)
    if err != nil {
        h.handleError(c, err)
        return
    }

    c.JSON(200, TriggerEventResponse{
        InstanceID:    result.InstanceID,
        PreviousState: result.PreviousState,
        CurrentState:  result.CurrentState,
        Version:       result.Version,
    })
}
```

#### 3.3.5 Event Dispatcher (RabbitMQ)

```go
type RabbitMQEventDispatcher struct {
    conn     *amqp.Connection
    channel  *amqp.Channel
    exchange string
}

func (d *RabbitMQEventDispatcher) Dispatch(ctx context.Context, evt event.DomainEvent) error {
    body, err := json.Marshal(evt)
    if err != nil {
        return err
    }

    return d.channel.PublishWithContext(ctx,
        d.exchange,        // exchange
        evt.Type(),        // routing key
        false,             // mandatory
        false,             // immediate
        amqp.Publishing{
            ContentType:  "application/json",
            Body:         body,
            DeliveryMode: amqp.Persistent,
            Timestamp:    time.Now(),
            Type:         evt.Type(),
        },
    )
}
```

#### 3.3.6 Workflow Loader (YAML)

```go
type YAMLWorkflowLoader struct {
    validator *WorkflowValidator
}

func (l *YAMLWorkflowLoader) LoadFromFile(path string) (*workflow.Workflow, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var config WorkflowConfig
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, err
    }

    if err := l.validator.Validate(&config); err != nil {
        return nil, err
    }

    return l.buildWorkflow(&config)
}

func (l *YAMLWorkflowLoader) buildWorkflow(config *WorkflowConfig) (*workflow.Workflow, error) {
    // Construir states (R17: con subestados)
    states := []workflow.State{}
    for _, sc := range config.Workflow.States {
        state, err := workflow.NewState(sc.ID, sc.Name)
        if err != nil {
            return nil, err
        }

        if sc.Timeout != "" {
            timeout, _ := time.ParseDuration(sc.Timeout)
            state = state.WithTimeout(timeout, sc.OnTimeout)
        }

        if sc.Final {
            state = state.AsFinal()
        }

        // R17: Agregar subestados si existen
        if len(sc.Substates) > 0 {
            substates := []workflow.SubState{}
            for _, ssc := range sc.Substates {
                substate, _ := workflow.NewSubState(ssc.ID, ssc.Name)
                substates = append(substates, substate)
            }
            state = state.WithSubstates(substates)
        }

        // R18: Configurar escalamientos
        if sc.AllowEscalation {
            state = state.WithEscalationEnabled(sc.EscalationGuard)
        }

        states = append(states, state)
    }

    // Construir events (R20: con guards avanzados, R23: con metadata schema)
    events := []workflow.Event{}
    for _, ec := range config.Workflow.Events {
        event := workflow.NewEvent(ec.Name, ec.From, ec.To)

        // R20: Configurar guards
        for _, guardConfig := range ec.Guards {
            guard, err := l.guardRegistry.Get(guardConfig)
            if err != nil {
                return nil, err
            }
            event = event.WithGuard(guard)
        }

        // R23: Configurar metadata schema
        if ec.RequireReason {
            event = event.WithRequiredReason()
        }
        if ec.RequireFeedback {
            event = event.WithRequiredFeedback()
        }
        if ec.MetadataSchema != nil {
            schema, _ := l.buildMetadataSchema(ec.MetadataSchema)
            event = event.WithMetadataSchema(schema)
        }

        // ... configurar validators, actions, etc
        events = append(events, event)
    }

    // Crear workflow (R21: con soporte para templates)
    initialState := states[0]
    wf, err := workflow.NewWorkflow(
        workflow.Name(config.Workflow.Name),
        initialState,
    )

    // Agregar estados y eventos
    for _, state := range states[1:] {
        wf.AddState(state)
    }

    for _, event := range events {
        wf.AddEvent(event)
    }

    // R19: Configurar reclasificación
    if config.Reclassification != nil && config.Reclassification.Enabled {
        wf.EnableReclassification(config.Reclassification)
    }

    return wf, nil
}
```

#### 3.3.7 Ejemplo Completo de Configuración YAML (Con Nuevas Características)

```yaml
# config/workflows/person_document_flow.yaml
workflow:
  id: person_document_flow
  name: "Flujo de Documentos de Persona"
  version: "2.0"
  description: "Workflow con subestados, escalamientos, guards avanzados y metadata"

  # R17: Estados con subestados
  states:
    - id: filed
      name: "Radicado"
      timeout: "24h"
      on_timeout: "auto_escalate"

    - id: assigned
      name: "Asignado"
      timeout: "12h"

    - id: in_progress
      name: "En Progreso"
      timeout: "72h"
      # R17: Subestados
      substates:
        - id: working
          name: "Trabajando"
          description: "Gestionador trabajando normalmente"
        - id: escalated_awaiting_response
          name: "Escalado - Esperando respuesta"
          description: "Escalado a departamento externo"
        - id: additional_info_required
          name: "Información Adicional Requerida"
          description: "Esperando más información del solicitante"
      # R18: Permitir escalamientos
      allow_escalation: true
      escalation_guard: has_role:gestionador

    - id: in_review
      name: "En Revisión"
      timeout: "24h"
      substates:
        - id: initial_review
          name: "Revisión Inicial"
        - id: senior_review
          name: "Revisión Senior"

    - id: approved
      name: "Aprobado"
      final: true

    - id: rejected
      name: "Rechazado"
      final: true

  # R20: Eventos con guards avanzados y R23: metadata schema
  events:
    - name: assign_document
      from: [filed]
      to: assigned
      # R20: Guards avanzados
      guards:
        - has_role:asignador
        - field_not_empty:data.document_type
        - instance_age_less_than:24h

    - name: start_processing
      from: [assigned]
      to: in_progress
      guards:
        - has_role:gestionador
        - is_assigned_to_actor

    - name: request_additional_info
      from: [in_progress]
      to: in_progress  # Reentrada al mismo estado
      # R23: Metadata requerida
      require_reason: true
      require_feedback: false
      metadata_schema:
        required:
          reason: string
          requested_fields:
            type: array
            items: string
        optional:
          contact_method:
            type: enum
            values: [email, phone, portal]

    - name: escalate
      from: [in_progress]
      to: in_progress  # Permanece en mismo estado, cambia subestado
      # R23: Metadata extendida
      require_reason: true
      require_feedback: false
      metadata_schema:
        required:
          reason: string
          department_id: string
        optional:
          urgency:
            type: enum
            values: [low, medium, high, critical]
          estimated_response_days: number

    - name: submit_for_review
      from: [in_progress]
      to: in_review
      guards:
        - has_role:gestionador
        - field_not_empty:data.completed_work
        - instance_age_more_than:1h  # Al menos 1 hora trabajando

    - name: approve
      from: [in_review]
      to: approved
      guards:
        - has_role:revisor
        - or:
            - has_role:senior_revisor
            - field_equals:data.risk_level:low
      # R23: Metadata
      require_feedback: true
      metadata_schema:
        required:
          feedback: string
          approval_notes: string
        optional:
          quality_score:
            type: number
            min: 1
            max: 10

    - name: reject
      from: [in_review]
      to: in_progress  # Vuelve a estado previo
      # R23: Metadata obligatoria
      require_reason: true
      require_feedback: true
      metadata_schema:
        required:
          reason: string
          feedback: string
          correction_notes: string
        optional:
          priority:
            type: enum
            values: [low, medium, high]

    - name: reclassify
      from: [in_progress, in_review]
      to: in_progress  # Loop/stay
      # R19: Reclasificación
      guards:
        - has_role:gestionador_senior
      require_reason: true
      metadata_schema:
        required:
          reason: string
          from_type: string
          to_type: string

# R19: Configuración de reclasificación
reclassification:
  enabled: true
  allowed_types:
    - PQRD
    - Control
    - Queja
    - Reclamo
    - Sugerencia
    - Felicitación
  allowed_from_states:
    - in_progress
    - in_review
  required_role: gestionador
  requires_senior: true

# R20: Guards personalizados (ejemplo de cómo se registrarían)
custom_guards:
  - name: has_attachments_with_size
    type: custom
    implementation: internal.guards.HasAttachmentsWithSize
    config:
      min_size_kb: 10
      max_size_kb: 5120
```

---

## 4. Interfaces y Contratos

### 4.1 Repository Ports (Domain)

```go
// internal/domain/instance/repository.go
package instance

import "context"

type Repository interface {
    // Basic CRUD
    Save(ctx context.Context, instance *Instance) error
    FindByID(ctx context.Context, id ID) (*Instance, error)
    Delete(ctx context.Context, id ID) error

    // Optimistic locking
    SaveWithVersion(ctx context.Context, instance *Instance, expectedVersion Version) error

    // Queries
    FindByWorkflow(ctx context.Context, workflowID string) ([]*Instance, error)
    FindByState(ctx context.Context, state string) ([]*Instance, error)
    FindByActor(ctx context.Context, actorID string) ([]*Instance, error)

    // Complex queries
    Query(ctx context.Context, spec Specification) ([]*Instance, int, error)
}

// Specification pattern
type Specification interface {
    IsSatisfiedBy(instance *Instance) bool
    ToSQL() (query string, args []interface{})
}
```

### 4.2 Event Dispatcher Port

```go
// internal/domain/event/dispatcher.go
package event

import "context"

type Dispatcher interface {
    Dispatch(ctx context.Context, event DomainEvent) error
    DispatchBatch(ctx context.Context, events []DomainEvent) error
}

type DomainEvent interface {
    Type() string
    AggregateID() string
    OccurredAt() time.Time
    Payload() map[string]interface{}
}

// Eventos concretos
type InstanceCreated struct {
    instanceID string
    workflowID string
    state      string
    occurredAt time.Time
}

func (e InstanceCreated) Type() string        { return "instance.created" }
func (e InstanceCreated) AggregateID() string { return e.instanceID }
func (e InstanceCreated) OccurredAt() time.Time { return e.occurredAt }

type StateChanged struct {
    instanceID   string
    fromState    string
    toState      string
    fromSubState *string  // R17
    toSubState   *string  // R17
    event        string
    actor        string
    reason       *string  // R23
    feedback     *string  // R23
    occurredAt   time.Time
}

func (e StateChanged) Type() string { return "state.changed" }

// R18: Eventos de escalamientos
type DocumentEscalated struct {
    instanceID   string
    escalationID string
    departmentID string
    escalatedBy  string
    reason       string
    occurredAt   time.Time
}

func (e DocumentEscalated) Type() string { return "document.escalated" }

type EscalationReplied struct {
    escalationID string
    instanceID   string
    respondedBy  string
    response     string
    occurredAt   time.Time
}

func (e EscalationReplied) Type() string { return "escalation.replied" }

type EscalationClosed struct {
    escalationID string
    instanceID   string
    closedBy     string
    occurredAt   time.Time
}

func (e EscalationClosed) Type() string { return "escalation.closed" }

// R19: Evento de reclasificación
type DocumentReclassified struct {
    instanceID string
    fromType   string
    toType     string
    reason     string
    actor      string
    occurredAt time.Time
}

func (e DocumentReclassified) Type() string { return "document.reclassified" }
```

### 4.3 Locker Port

```go
// pkg/ports/lock.go
package ports

import (
    "context"
    "time"
)

type Locker interface {
    Lock(ctx context.Context, key string, ttl time.Duration) (Lock, error)
}

type Lock interface {
    Unlock(ctx context.Context) error
    Refresh(ctx context.Context, ttl time.Duration) error
}
```

### 4.4 Logger Port

```go
// pkg/ports/logger.go
package ports

type Logger interface {
    Debug(msg string, keysAndValues ...interface{})
    Info(msg string, keysAndValues ...interface{})
    Warn(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
    With(keysAndValues ...interface{}) Logger
}
```

### 4.5 REST API Endpoints

#### Workflows

| Método | Endpoint | Descripción | Request | Response |
|--------|----------|-------------|---------|----------|
| GET | `/api/v1/workflows` | Listar workflows | Query: `is_template`, `template_id` | `WorkflowListResponse` |
| POST | `/api/v1/workflows` | Crear workflow | `CreateWorkflowRequest` | `WorkflowResponse` |
| GET | `/api/v1/workflows/:id` | Obtener workflow | - | `WorkflowResponse` |
| PUT | `/api/v1/workflows/:id` | Actualizar workflow | `UpdateWorkflowRequest` | `WorkflowResponse` |
| DELETE | `/api/v1/workflows/:id` | Eliminar workflow | Query: `force` | `204 No Content` |
| GET | `/api/v1/workflows/:id/visualize` | Visualizar (mermaid) | - | `VisualizationResponse` |
| POST | `/api/v1/workflows/from-template` | Clonar desde plantilla (R21) | `CloneFromTemplateRequest` | `WorkflowResponse` |
| GET | `/api/v1/workflows/:id/export` | Exportar workflow (R22) | Query: `format=yaml\|json` | File download |
| POST | `/api/v1/workflows/export-batch` | Exportar múltiples (R22) | `ExportBatchRequest` | ZIP download |
| GET | `/api/v1/workflows/export-all` | Exportar todos (R22) | - | ZIP download |
| POST | `/api/v1/workflows/import` | Importar workflow (R22) | Multipart/JSON | `WorkflowResponse` |
| POST | `/api/v1/workflows/import-bulk` | Importar bulk (R22) | ZIP file | `ImportBulkResponse` |

#### Instances

| Método | Endpoint | Descripción | Request | Response |
|--------|----------|-------------|---------|----------|
| POST | `/api/v1/instances` | Crear instancia | `CreateInstanceRequest` | `InstanceResponse` |
| GET | `/api/v1/instances` | Listar instancias | Query params | `InstanceListResponse` |
| GET | `/api/v1/instances/:id` | Obtener instancia | - | `InstanceResponse` |
| POST | `/api/v1/instances/:id/events` | Trigger evento | `TriggerEventRequest` | `TransitionResponse` |
| GET | `/api/v1/instances/:id/history` | Ver historial | Query: `event`, `has_feedback` | `HistoryResponse` |
| POST | `/api/v1/instances/:id/pause` | Pausar instancia | - | `InstanceResponse` |
| POST | `/api/v1/instances/:id/resume` | Reanudar instancia | - | `InstanceResponse` |
| DELETE | `/api/v1/instances/:id` | Cancelar instancia | - | `204 No Content` |
| POST | `/api/v1/instances/:id/escalate` | Escalar a departamento (R18) | `EscalateRequest` | `EscalationResponse` |
| POST | `/api/v1/instances/:id/escalation-reply` | Responder escalamiento (R18) | `EscalationReplyRequest` | `EscalationResponse` |
| GET | `/api/v1/instances/:id/escalations` | Listar escalamientos (R18) | - | `EscalationListResponse` |
| POST | `/api/v1/instances/:id/reclassify` | Reclasificar instancia (R19) | `ReclassifyRequest` | `ReclassificationResponse` |
| GET | `/api/v1/instances/:id/reclassifications` | Historial reclasif. (R19) | - | `ReclassificationListResponse` |

#### Escalations (R18)

| Método | Endpoint | Descripción | Request | Response |
|--------|----------|-------------|---------|----------|
| GET | `/api/v1/escalations/:id` | Obtener escalamiento | - | `EscalationResponse` |
| POST | `/api/v1/escalations/:id/close` | Cerrar escalamiento | - | `EscalationResponse` |
| DELETE | `/api/v1/escalations/:id` | Cancelar escalamiento | - | `204 No Content` |
| GET | `/api/v1/queries/escalations` | Query escalamientos | Query: `department_id`, `status` | `EscalationListResponse` |

#### Queries

| Método | Endpoint | Descripción | Request | Response |
|--------|----------|-------------|---------|----------|
| POST | `/api/v1/queries/instances` | Query avanzado | `QueryRequest` (con `sub_states`) | `QueryResponse` |
| GET | `/api/v1/queries/statistics` | Estadísticas | Query params | `StatisticsResponse` |
| GET | `/api/v1/queries/actors/:id/workload` | Carga de trabajo | - | `WorkloadResponse` |
| POST | `/api/v1/queries/transitions` | Query transiciones (R23) | `TransitionsQueryRequest` | `TransitionsQueryResponse` |

### 4.6 Request/Response DTOs

```go
// CreateInstanceRequest
type CreateInstanceRequest struct {
    WorkflowID string                 `json:"workflow_id" binding:"required"`
    ActorID    string                 `json:"actor_id" binding:"required"`
    ActorRole  string                 `json:"actor_role" binding:"required"`
    Data       map[string]interface{} `json:"data"`
}

// InstanceResponse
type InstanceResponse struct {
    ID               string                 `json:"id"`
    WorkflowID       string                 `json:"workflow_id"`
    CurrentState     string                 `json:"current_state"`
    PreviousState    *string                `json:"previous_state,omitempty"`
    CurrentSubState  *string                `json:"current_sub_state,omitempty"`   // R17
    PreviousSubState *string                `json:"previous_sub_state,omitempty"`  // R17
    Status           string                 `json:"status"`
    Version          int64                  `json:"version"`
    Data             map[string]interface{} `json:"data"`
    Variables        map[string]interface{} `json:"variables"`
    CurrentActor     string                 `json:"current_actor,omitempty"`
    CreatedAt        time.Time              `json:"created_at"`
    UpdatedAt        time.Time              `json:"updated_at"`
    CompletedAt      *time.Time             `json:"completed_at,omitempty"`
}

// TriggerEventRequest
type TriggerEventRequest struct {
    Event    string                 `json:"event" binding:"required"`
    Actor    string                 `json:"actor" binding:"required"`
    Data     map[string]interface{} `json:"data"`
    Reason   string                 `json:"reason,omitempty"`     // R23
    Feedback string                 `json:"feedback,omitempty"`   // R23
    Metadata map[string]interface{} `json:"metadata,omitempty"`   // R23
}

// TransitionResponse
type TransitionResponse struct {
    InstanceID    string    `json:"instance_id"`
    PreviousState string    `json:"previous_state"`
    CurrentState  string    `json:"current_state"`
    Version       int64     `json:"version"`
    TransitionedAt time.Time `json:"transitioned_at"`
}

// QueryRequest
type QueryRequest struct {
    WorkflowID string     `json:"workflow_id,omitempty"`
    States     []string   `json:"states,omitempty"`
    Actors     []string   `json:"actors,omitempty"`
    Status     *string    `json:"status,omitempty"`
    FromDate   *time.Time `json:"from_date,omitempty"`
    ToDate     *time.Time `json:"to_date,omitempty"`
    Limit      int        `json:"limit" binding:"min=1,max=100"`
    Offset     int        `json:"offset" binding:"min=0"`
}

// ErrorResponse
type ErrorResponse struct {
    Error   string `json:"error"`
    Code    string `json:"code,omitempty"`
    Details string `json:"details,omitempty"`
}

// R18: Escalamientos - Request/Response
type EscalateRequest struct {
    DepartmentId string `json:"department_id" binding:"required"`
    Reason       string `json:"reason" binding:"required"`
}

type EscalationReplyRequest struct {
    EscalationId string `json:"escalation_id" binding:"required"`
    Response     string `json:"response" binding:"required"`
}

type EscalationResponse struct {
    ID           string     `json:"id"`
    InstanceID   string     `json:"instance_id"`
    DepartmentID string     `json:"department_id"`
    Reason       string     `json:"reason"`
    EscalatedBy  string     `json:"escalated_by"`
    EscalatedAt  time.Time  `json:"escalated_at"`
    Status       string     `json:"status"`
    Response     *string    `json:"response,omitempty"`
    RespondedBy  *string    `json:"responded_by,omitempty"`
    RespondedAt  *time.Time `json:"responded_at,omitempty"`
    ClosedBy     *string    `json:"closed_by,omitempty"`
    ClosedAt     *time.Time `json:"closed_at,omitempty"`
}

type EscalationListResponse struct {
    Escalations []EscalationResponse `json:"escalations"`
    Total       int                  `json:"total"`
}

// R19: Reclasificación - Request/Response
type ReclassifyRequest struct {
    NewType string `json:"new_type" binding:"required"`
    Reason  string `json:"reason" binding:"required"`
}

type ReclassificationResponse struct {
    InstanceID     string    `json:"instance_id"`
    FromType       string    `json:"from_type"`
    ToType         string    `json:"to_type"`
    CurrentState   string    `json:"current_state"`
    Version        int64     `json:"version"`
    ReclassifiedAt time.Time `json:"reclassified_at"`
}

type ReclassificationListResponse struct {
    Reclassifications []TransitionResponse `json:"reclassifications"`
    Total             int                  `json:"total"`
    UniqueTypes       []string             `json:"unique_types"`
}

// R21: Plantillas - Request/Response
type CloneFromTemplateRequest struct {
    TemplateID  string                 `json:"template_id" binding:"required"`
    NewID       *string                `json:"new_id,omitempty"`
    Name        string                 `json:"name" binding:"required"`
    Description string                 `json:"description"`
    Overrides   map[string]interface{} `json:"overrides,omitempty"`
}

type WorkflowResponse struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Version     string                 `json:"version"`
    Config      map[string]interface{} `json:"config"`
    IsTemplate  bool                   `json:"is_template"`
    TemplateID  *string                `json:"template_id,omitempty"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

// R22: Import/Export - Request/Response
type ExportBatchRequest struct {
    WorkflowIDs []string `json:"workflow_ids" binding:"required"`
}

type ImportBulkResponse struct {
    TotalProcessed int               `json:"total_processed"`
    Successful     int               `json:"successful"`
    Failed         int               `json:"failed"`
    Errors         []ImportError     `json:"errors,omitempty"`
    ImportedIDs    []string          `json:"imported_ids"`
}

type ImportError struct {
    WorkflowID string `json:"workflow_id"`
    Error      string `json:"error"`
    Line       *int   `json:"line,omitempty"`
}

// R23: Metadata Extendida - Request/Response
type TransitionsQueryRequest struct {
    WorkflowID     string                 `json:"workflow_id,omitempty"`
    Events         []string               `json:"events,omitempty"`
    FromDate       *time.Time             `json:"from_date,omitempty"`
    ToDate         *time.Time             `json:"to_date,omitempty"`
    HasFeedback    *bool                  `json:"has_feedback,omitempty"`
    MetadataFilter map[string]interface{} `json:"metadata_filter,omitempty"`
    Limit          int                    `json:"limit" binding:"min=1,max=100"`
    Offset         int                    `json:"offset" binding:"min=0"`
}

type TransitionsQueryResponse struct {
    Transitions []TransitionDetailResponse `json:"transitions"`
    Total       int                        `json:"total"`
    Limit       int                        `json:"limit"`
    Offset      int                        `json:"offset"`
}

type TransitionDetailResponse struct {
    ID           string                 `json:"id"`
    InstanceID   string                 `json:"instance_id"`
    Event        string                 `json:"event"`
    FromState    string                 `json:"from_state"`
    ToState      string                 `json:"to_state"`
    FromSubState *string                `json:"from_sub_state,omitempty"`
    ToSubState   *string                `json:"to_sub_state,omitempty"`
    Actor        string                 `json:"actor"`
    ActorRole    string                 `json:"actor_role"`
    Data         map[string]interface{} `json:"data,omitempty"`
    Reason       *string                `json:"reason,omitempty"`
    Feedback     *string                `json:"feedback,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
    DurationMs   int                    `json:"duration_ms"`
    CreatedAt    time.Time              `json:"created_at"`
}
```

---

## 5. Modelos de Datos

### 5.1 Schema PostgreSQL Completo

```sql
-- Workflows (definiciones)
CREATE TABLE workflows (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version VARCHAR(50) NOT NULL,
    config JSONB NOT NULL,

    -- R21: Plantillas de workflows
    is_template BOOLEAN NOT NULL DEFAULT FALSE,
    template_id UUID REFERENCES workflows(id),

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_workflows_name ON workflows(name);
CREATE INDEX idx_workflows_deleted ON workflows(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_workflows_template ON workflows(template_id) WHERE template_id IS NOT NULL;
CREATE INDEX idx_workflows_is_template ON workflows(is_template) WHERE is_template = TRUE;

-- Workflow instances (ejecuciones)
CREATE TABLE workflow_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id VARCHAR(255) NOT NULL REFERENCES workflows(id),
    parent_id UUID REFERENCES workflow_instances(id),

    -- Estado
    current_state VARCHAR(100) NOT NULL,
    previous_state VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'running',

    -- R17: Subestados
    current_sub_state VARCHAR(100),
    previous_sub_state VARCHAR(100),

    -- Optimistic locking
    version BIGINT NOT NULL DEFAULT 1,

    -- Datos
    data JSONB,
    variables JSONB,

    -- Actores
    current_actor VARCHAR(255),
    current_role VARCHAR(100),

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,

    -- Distributed locking
    locked_by VARCHAR(255),
    locked_at TIMESTAMP,
    lock_expires_at TIMESTAMP,

    CONSTRAINT check_status CHECK (status IN ('running', 'paused', 'completed', 'canceled', 'failed'))
);

-- Indices para queries frecuentes
CREATE INDEX idx_instances_workflow ON workflow_instances(workflow_id);
CREATE INDEX idx_instances_state ON workflow_instances(current_state);
CREATE INDEX idx_instances_status ON workflow_instances(status);
CREATE INDEX idx_instances_actor ON workflow_instances(current_actor);
CREATE INDEX idx_instances_parent ON workflow_instances(parent_id);
CREATE INDEX idx_instances_created ON workflow_instances(created_at DESC);
CREATE INDEX idx_instances_version ON workflow_instances(id, version);

-- Índice compuesto para queries complejas
CREATE INDEX idx_instances_query ON workflow_instances(workflow_id, status, current_state, created_at DESC);

-- R17: Índice para subestados
CREATE INDEX idx_instances_substates ON workflow_instances(current_state, current_sub_state, status) WHERE current_sub_state IS NOT NULL;

-- Transiciones (historial)
CREATE TABLE workflow_transitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,

    event VARCHAR(100) NOT NULL,
    from_state VARCHAR(100) NOT NULL,
    to_state VARCHAR(100) NOT NULL,

    -- R17: Subestados
    from_sub_state VARCHAR(100),
    to_sub_state VARCHAR(100),

    actor VARCHAR(255),
    actor_role VARCHAR(100),

    data JSONB,

    -- R23: Metadata extendida
    reason TEXT,
    feedback TEXT,
    metadata JSONB,

    duration_ms INTEGER,

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transitions_instance ON workflow_transitions(instance_id, created_at DESC);
CREATE INDEX idx_transitions_event ON workflow_transitions(event);
CREATE INDEX idx_transitions_actor ON workflow_transitions(actor);

-- R23: Índices para metadata
CREATE INDEX idx_transitions_has_feedback ON workflow_transitions(instance_id) WHERE feedback IS NOT NULL;
CREATE INDEX idx_transitions_metadata ON workflow_transitions USING GIN (metadata) WHERE metadata IS NOT NULL;

-- Timers activos
CREATE TABLE workflow_timers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,

    state VARCHAR(100) NOT NULL,
    event_on_timeout VARCHAR(100) NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    fired_at TIMESTAMP,

    CONSTRAINT check_fired CHECK (fired_at IS NULL OR fired_at >= created_at)
);

CREATE INDEX idx_timers_expires ON workflow_timers(expires_at) WHERE fired_at IS NULL;
CREATE INDEX idx_timers_instance ON workflow_timers(instance_id);

-- Webhooks configurados
CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id VARCHAR(255) REFERENCES workflows(id),

    url TEXT NOT NULL,
    events TEXT[] NOT NULL,
    secret VARCHAR(255),

    headers JSONB,
    retry_config JSONB,

    active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT check_url CHECK (url ~ '^https?://')
);

CREATE INDEX idx_webhooks_workflow ON webhooks(workflow_id);
CREATE INDEX idx_webhooks_active ON webhooks(active) WHERE active = TRUE;

-- Eventos externos (queue)
CREATE TABLE external_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID REFERENCES workflow_instances(id),

    event_type VARCHAR(255) NOT NULL,
    payload JSONB,

    processed_at TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_external_events_processed ON external_events(processed_at);
CREATE INDEX idx_external_events_pending ON external_events(created_at) WHERE processed_at IS NULL;
CREATE INDEX idx_external_events_instance ON external_events(instance_id);

-- R18: Escalamientos manuales
CREATE TABLE workflow_escalations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,

    department_id VARCHAR(100) NOT NULL,
    reason TEXT NOT NULL,

    escalated_by VARCHAR(255) NOT NULL,
    escalated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    status VARCHAR(50) NOT NULL DEFAULT 'pending',

    response TEXT,
    responded_by VARCHAR(255),
    responded_at TIMESTAMP,

    closed_by VARCHAR(255),
    closed_at TIMESTAMP,

    CONSTRAINT check_escalation_status CHECK (status IN ('pending', 'responded', 'closed', 'canceled'))
);

CREATE INDEX idx_escalations_instance ON workflow_escalations(instance_id, escalated_at DESC);
CREATE INDEX idx_escalations_department ON workflow_escalations(department_id, status);
CREATE INDEX idx_escalations_status ON workflow_escalations(status) WHERE status IN ('pending', 'responded');
CREATE INDEX idx_escalations_escalated_by ON workflow_escalations(escalated_by);

-- Auditoría (opcional)
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    actor VARCHAR(255),
    changes JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_entity ON audit_log(entity_type, entity_id, created_at DESC);
CREATE INDEX idx_audit_actor ON audit_log(actor, created_at DESC);
```

### 5.2 Migraciones

**001_initial.up.sql**: Schema completo arriba

**001_initial.down.sql**:
```sql
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS external_events;
DROP TABLE IF EXISTS webhooks;
DROP TABLE IF EXISTS workflow_timers;
DROP TABLE IF EXISTS workflow_transitions;
DROP TABLE IF EXISTS workflow_instances;
DROP TABLE IF EXISTS workflows;
```

### 5.3 Datos de Ejemplo

```sql
-- Workflow: radicacion
INSERT INTO workflows (id, name, description, version, config) VALUES (
    'radicacion',
    'Flujo de Radicación',
    'Proceso completo de radicación de documentos',
    '1.0',
    '{
        "initial_state": "radicar",
        "states": [
            {"id": "radicar", "name": "Radicación", "timeout": "24h"},
            {"id": "asignar", "name": "Asignación", "timeout": "12h"},
            {"id": "gestionar", "name": "Gestión", "timeout": "48h"},
            {"id": "revisar", "name": "Revisión", "timeout": "12h"},
            {"id": "aprobar", "name": "Aprobación", "timeout": "12h"},
            {"id": "enviar", "name": "Envío", "final": true}
        ],
        "events": [
            {"name": "generar_radicado", "from": ["radicar"], "to": "asignar"},
            {"name": "asignar_gestor", "from": ["asignar"], "to": "gestionar"},
            {"name": "rechazar_asignacion", "from": ["asignar"], "to": "radicar"}
        ]
    }'::jsonb
);

-- Instancia de ejemplo
INSERT INTO workflow_instances (
    id, workflow_id, current_state, status, data
) VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    'radicacion',
    'gestionar',
    'running',
    '{
        "tipo": "PQRD",
        "remitente": "Juan Pérez",
        "asunto": "Solicitud de información",
        "numeroRadicado": "RAD-2024-001234"
    }'::jsonb
);
```

---

## 6. Manejo de Errores

### 6.1 Jerarquía de Errores

```go
// pkg/common/errors/errors.go
package errors

import "errors"

// Base errors
var (
    ErrNotFound        = errors.New("not found")
    ErrAlreadyExists   = errors.New("already exists")
    ErrInvalidInput    = errors.New("invalid input")
    ErrUnauthorized    = errors.New("unauthorized")
    ErrForbidden       = errors.New("forbidden")
    ErrConflict        = errors.New("conflict")
    ErrInternal        = errors.New("internal error")
)

// Domain errors
var (
    // Workflow
    ErrInvalidWorkflow     = errors.New("invalid workflow")
    ErrStateNotFound       = errors.New("state not found")
    ErrEventNotFound       = errors.New("event not found")

    // Instance
    ErrInvalidTransition   = errors.New("invalid transition")
    ErrVersionConflict     = errors.New("version conflict")
    ErrInstanceNotRunning  = errors.New("instance not running")
    ErrInstanceCompleted   = errors.New("instance already completed")

    // Actor
    ErrInsufficientPermissions = errors.New("insufficient permissions")
    ErrActorNotAssigned    = errors.New("actor not assigned")

    // Lock
    ErrLockAlreadyHeld     = errors.New("lock already held")
    ErrLockExpired         = errors.New("lock expired")
)

// Wrapped errors con contexto
type DomainError struct {
    Code    string
    Message string
    Cause   error
    Context map[string]interface{}
}

func (e *DomainError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Cause)
    }
    return e.Message
}

func (e *DomainError) Unwrap() error {
    return e.Cause
}

// Helpers
func NewDomainError(code, message string, cause error) *DomainError {
    return &DomainError{
        Code:    code,
        Message: message,
        Cause:   cause,
        Context: make(map[string]interface{}),
    }
}

func (e *DomainError) WithContext(key string, value interface{}) *DomainError {
    e.Context[key] = value
    return e
}
```

### 6.2 Estrategia por Capa

**Domain Layer**:
- Retorna errores de dominio puros (`ErrInvalidTransition`, etc)
- No conoce detalles de HTTP o DB
- Errores son valores, no excepciones

```go
func (i *Instance) Transition(...) error {
    if i.status != StatusRunning {
        return ErrInstanceNotRunning
    }

    if !wf.CanTransition(i.currentState, evt) {
        return ErrInvalidTransition
    }

    // ...
}
```

**Application Layer**:
- Wrappea errores de dominio con contexto
- Maneja transacciones y rollback
- Propaga errores a infrastructure

```go
func (uc *TriggerEventUseCase) Execute(...) error {
    inst, err := uc.instanceRepo.FindByID(ctx, instanceID)
    if err != nil {
        if errors.Is(err, instance.ErrNotFound) {
            return NewDomainError("INSTANCE_NOT_FOUND", "Instance not found", err).
                WithContext("instance_id", instanceID)
        }
        return fmt.Errorf("failed to load instance: %w", err)
    }

    if err := inst.Transition(ctx, wf, eventName, actorID); err != nil {
        if errors.Is(err, instance.ErrInvalidTransition) {
            return NewDomainError("INVALID_TRANSITION", "Invalid transition", err).
                WithContext("current_state", inst.CurrentState().ID()).
                WithContext("event", eventName)
        }
        return err
    }

    // ...
}
```

**Infrastructure Layer (HTTP)**:
- Mapea errores a HTTP status codes
- Genera responses JSON apropiados
- Log de errores

```go
func (h *InstanceHandler) handleError(c *gin.Context, err error) {
    var domainErr *errors.DomainError

    if errors.As(err, &domainErr) {
        switch domainErr.Code {
        case "INSTANCE_NOT_FOUND":
            c.JSON(404, ErrorResponse{
                Error: domainErr.Message,
                Code:  domainErr.Code,
            })
        case "INVALID_TRANSITION":
            c.JSON(409, ErrorResponse{
                Error: domainErr.Message,
                Code:  domainErr.Code,
                Details: fmt.Sprintf("Cannot execute %v from state %v",
                    domainErr.Context["event"],
                    domainErr.Context["current_state"]),
            })
        case "VERSION_CONFLICT":
            c.JSON(409, ErrorResponse{
                Error: "Optimistic lock conflict, please retry",
                Code:  "CONFLICT",
            })
        default:
            h.logger.Error("domain error", "error", err, "context", domainErr.Context)
            c.JSON(500, ErrorResponse{Error: "Internal error"})
        }
        return
    }

    // Unknown error
    h.logger.Error("unexpected error", "error", err)
    c.JSON(500, ErrorResponse{Error: "Internal server error"})
}
```

### 6.3 Códigos de Error HTTP

| HTTP Code | Significado | Cuándo usar |
|-----------|-------------|-------------|
| 400 | Bad Request | Request inválido, validación fallida |
| 401 | Unauthorized | No autenticado |
| 403 | Forbidden | Sin permisos |
| 404 | Not Found | Recurso no existe |
| 409 | Conflict | Version conflict, invalid transition |
| 422 | Unprocessable Entity | Validación de negocio fallida |
| 429 | Too Many Requests | Rate limit excedido |
| 500 | Internal Server Error | Error inesperado |
| 503 | Service Unavailable | DB/Redis down |

---

## 7. Estrategia de Testing

### 7.1 Pirámide de Tests

```
       /\
      /  \     E2E Tests (5%)
     /----\
    /      \   Integration Tests (15%)
   /--------\
  /          \ Unit Tests (80%)
 /____________\
```

### 7.2 Unit Tests (Domain Layer)

**Objetivo**: Testear lógica de negocio aislada

**Características**:
- Sin dependencias externas
- Rápidos (<10ms cada uno)
- Deterministicos
- Coverage >90%

**Ejemplo**:
```go
// internal/domain/instance/instance_test.go
func TestInstance_Transition_Success(t *testing.T) {
    // Given
    initialState, _ := workflow.NewState("pending", "Pending")
    approvedState, _ := workflow.NewState("approved", "Approved")

    inst, _ := instance.NewInstance(
        workflow.NewID("test"),
        initialState,
        instance.NewData(),
    )

    wf := createTestWorkflow(initialState, approvedState)

    // When
    err := inst.Transition(context.Background(), wf, "approve", "user123")

    // Then
    assert.NoError(t, err)
    assert.Equal(t, "approved", inst.CurrentState().ID())
    assert.Equal(t, int64(2), inst.Version().Value())
    assert.Len(t, inst.History(), 1)

    events := inst.DomainEvents()
    assert.Len(t, events, 2) // Created + StateChanged
    assert.Equal(t, "state.changed", events[1].Type())
}

func TestInstance_Transition_InvalidEvent(t *testing.T) {
    // Given
    state, _ := workflow.NewState("pending", "Pending")
    inst, _ := instance.NewInstance(
        workflow.NewID("test"),
        state,
        instance.NewData(),
    )
    wf := createTestWorkflow(state)

    // When
    err := inst.Transition(context.Background(), wf, "invalid", "user123")

    // Then
    assert.Error(t, err)
    assert.ErrorIs(t, err, instance.ErrEventNotFound)
}

// Table-driven tests
func TestState_Validation(t *testing.T) {
    tests := []struct {
        name    string
        id      string
        stName  string
        wantErr bool
    }{
        {"valid", "pending", "Pending", false},
        {"empty id", "", "Pending", true},
        {"invalid id", "Invalid-ID", "Pending", true},
        {"empty name", "pending", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            state, err := workflow.NewState(tt.id, tt.stName)

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.id, state.ID())
            }
        })
    }
}
```

### 7.3 Integration Tests (Application + Infrastructure)

**Objetivo**: Testear interacción entre capas con dependencias reales

**Características**:
- PostgreSQL real (testcontainers)
- Redis real (testcontainers)
- Transacciones con rollback
- Más lentos (~100-500ms)

**Setup**:
```go
//go:build integration

package integration_test

import (
    "context"
    "testing"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/modules/redis"
)

type TestSuite struct {
    pgContainer    *postgres.PostgresContainer
    redisContainer *redis.RedisContainer
    db             *sql.DB
    redisClient    *redis.Client
    container      *di.Container
}

func SetupSuite(t *testing.T) *TestSuite {
    ctx := context.Background()

    // Start PostgreSQL
    pgContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
    )
    require.NoError(t, err)

    pgURL, _ := pgContainer.ConnectionString(ctx)
    db, _ := sql.Open("postgres", pgURL)

    // Run migrations
    runMigrations(t, db)

    // Start Redis
    redisContainer, err := redis.RunContainer(ctx,
        testcontainers.WithImage("redis:7-alpine"),
    )
    require.NoError(t, err)

    redisURL, _ := redisContainer.ConnectionString(ctx)
    redisClient := redis.NewClient(&redis.Options{Addr: redisURL})

    // Create DI container
    container := createTestContainer(db, redisClient)

    return &TestSuite{
        pgContainer:    pgContainer,
        redisContainer: redisContainer,
        db:             db,
        redisClient:    redisClient,
        container:      container,
    }
}

func (s *TestSuite) Teardown(t *testing.T) {
    s.pgContainer.Terminate(context.Background())
    s.redisContainer.Terminate(context.Background())
}
```

**Test Example**:
```go
func TestInstance_CreateAndTransition_Integration(t *testing.T) {
    suite := SetupSuite(t)
    defer suite.Teardown(t)

    ctx := context.Background()

    // Given: workflow existe en DB
    workflowID := "test-workflow"
    seedWorkflow(t, suite.db, workflowID)

    // When: creamos instancia
    createCmd := application.CreateInstanceCommand{
        WorkflowID: workflowID,
        ActorID:    "user123",
        ActorRole:  "admin",
        Data:       map[string]interface{}{"test": "value"},
    }

    result, err := suite.container.CreateInstanceUseCase().Execute(ctx, createCmd)
    require.NoError(t, err)

    instanceID := result.InstanceID

    // Then: instancia existe en DB
    var count int
    suite.db.QueryRow("SELECT COUNT(*) FROM workflow_instances WHERE id = $1", instanceID).Scan(&count)
    assert.Equal(t, 1, count)

    // And: instancia existe en cache
    cached, err := suite.redisClient.Get(ctx, "instance:"+instanceID).Result()
    assert.NoError(t, err)
    assert.NotEmpty(t, cached)

    // When: ejecutamos transición
    triggerCmd := application.TriggerEventCommand{
        InstanceID: instanceID,
        EventName:  "approve",
        ActorID:    "user123",
    }

    _, err = suite.container.TriggerEventUseCase().Execute(ctx, triggerCmd)
    require.NoError(t, err)

    // Then: estado actualizado en DB
    var currentState string
    suite.db.QueryRow("SELECT current_state FROM workflow_instances WHERE id = $1", instanceID).Scan(&currentState)
    assert.Equal(t, "approved", currentState)

    // And: transición registrada en historial
    suite.db.QueryRow("SELECT COUNT(*) FROM workflow_transitions WHERE instance_id = $1", instanceID).Scan(&count)
    assert.Equal(t, 1, count)
}
```

### 7.4 E2E Tests (API completa)

**Objetivo**: Testear flujos completos desde HTTP

**Características**:
- HTTP server real
- DB y Redis reales
- Flujos de usuario completos
- Más lentos (~1-5s)

**Ejemplo**:
```go
func TestRadicacionWorkflow_E2E(t *testing.T) {
    suite := SetupSuite(t)
    defer suite.Teardown(t)

    // Start HTTP server
    server := startTestServer(suite.container)
    defer server.Shutdown(context.Background())

    baseURL := fmt.Sprintf("http://localhost:%d/api/v1", server.Port)

    // Crear instancia
    createResp := httpPost(t, baseURL+"/instances", map[string]interface{}{
        "workflow_id": "radicacion",
        "actor_id":    "radicador1",
        "actor_role":  "radicador",
        "data": map[string]interface{}{
            "tipo":      "PQRD",
            "remitente": "Juan Pérez",
        },
    })

    instanceID := createResp["id"].(string)
    assert.Equal(t, "radicar", createResp["current_state"])

    // Flujo completo
    transitions := []struct {
        event         string
        expectedState string
    }{
        {"generar_radicado", "asignar"},
        {"asignar_gestor", "gestionar"},
        {"enviar_revision", "revisar"},
        {"aprobar_revision", "aprobar"},
        {"aprobar_documento", "enviar"},
    }

    for _, tr := range transitions {
        resp := httpPost(t, baseURL+"/instances/"+instanceID+"/events", map[string]interface{}{
            "event": tr.event,
            "actor": "user123",
        })

        assert.Equal(t, tr.expectedState, resp["current_state"],
            "Event %s failed", tr.event)
    }

    // Verificar estado final
    finalResp := httpGet(t, baseURL+"/instances/"+instanceID)
    assert.Equal(t, "enviar", finalResp["current_state"])
    assert.Equal(t, "completed", finalResp["status"])

    // Verificar historial
    historyResp := httpGet(t, baseURL+"/instances/"+instanceID+"/history")
    transitions := historyResp["transitions"].([]interface{})
    assert.Len(t, transitions, 5)
}
```

### 7.5 Performance Tests (k6)

```javascript
// test/performance/load_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
    stages: [
        { duration: '1m', target: 50 },   // Ramp up
        { duration: '3m', target: 50 },   // Stay at 50 users
        { duration: '1m', target: 100 },  // Spike
        { duration: '2m', target: 100 },  // Stay at 100
        { duration: '1m', target: 0 },    // Ramp down
    ],
    thresholds: {
        http_req_duration: ['p(95)<500', 'p(99)<1000'], // 95% < 500ms, 99% < 1s
        http_req_failed: ['rate<0.01'],  // Error rate < 1%
    },
};

const BASE_URL = 'http://localhost:8080/api/v1';

export default function () {
    // Crear instancia
    let createResp = http.post(`${BASE_URL}/instances`, JSON.stringify({
        workflow_id: 'radicacion',
        actor_id: `user-${__VU}`,
        actor_role: 'radicador',
        data: { test: 'value' },
    }), {
        headers: { 'Content-Type': 'application/json' },
    });

    check(createResp, {
        'create status 201': (r) => r.status === 201,
        'create has id': (r) => JSON.parse(r.body).id !== undefined,
    });

    let instanceID = JSON.parse(createResp.body).id;

    // Trigger evento
    let triggerResp = http.post(`${BASE_URL}/instances/${instanceID}/events`, JSON.stringify({
        event: 'generar_radicado',
        actor: `user-${__VU}`,
    }), {
        headers: { 'Content-Type': 'application/json' },
    });

    check(triggerResp, {
        'trigger status 200': (r) => r.status === 200,
        'trigger changed state': (r) => JSON.parse(r.body).current_state === 'asignar',
    });

    sleep(1);
}
```

**Ejecutar**:
```bash
k6 run test/performance/load_test.js
```

### 7.6 Comandos de Testing

```makefile
# Makefile
.PHONY: test test-unit test-integration test-e2e test-coverage test-performance

test: test-unit test-integration

test-unit:
	go test -v -race -short ./internal/domain/... ./internal/application/...

test-integration:
	go test -v -race -tags=integration ./test/integration/...

test-e2e:
	go test -v -tags=e2e ./test/e2e/...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-performance:
	k6 run test/performance/load_test.js

test-all: test-unit test-integration test-e2e
```

---

## 8. Optimizaciones de Performance

### 8.1 Base de Datos

**Índices Estratégicos**:
```sql
-- Query frecuente: instancias por workflow y estado
CREATE INDEX idx_instances_query ON workflow_instances(
    workflow_id, status, current_state, created_at DESC
);

-- Query frecuente: instancias por actor
CREATE INDEX idx_instances_actor_state ON workflow_instances(
    current_actor, current_state, status
);

-- Optimistic locking
CREATE INDEX idx_instances_version ON workflow_instances(id, version);

-- Historial paginado
CREATE INDEX idx_transitions_instance_time ON workflow_transitions(
    instance_id, created_at DESC
);
```

**Connection Pooling**:
```go
func NewPostgresDB(dsn string) (*sql.DB, error) {
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, err
    }

    // Pool config
    db.SetMaxOpenConns(25)                  // Max connections
    db.SetMaxIdleConns(5)                   // Idle connections
    db.SetConnMaxLifetime(5 * time.Minute)  // Connection lifetime
    db.SetConnMaxIdleTime(1 * time.Minute)  // Idle timeout

    return db, db.Ping()
}
```

**Prepared Statements**:
```go
type PostgresInstanceRepository struct {
    db    *sql.DB
    stmts map[string]*sql.Stmt
}

func (r *PostgresInstanceRepository) init() error {
    stmts := map[string]string{
        "findByID": `SELECT id, workflow_id, current_state, ... FROM workflow_instances WHERE id = $1`,
        "save":     `INSERT INTO workflow_instances (...) VALUES (...) ON CONFLICT (id) DO UPDATE ...`,
    }

    r.stmts = make(map[string]*sql.Stmt)
    for name, query := range stmts {
        stmt, err := r.db.Prepare(query)
        if err != nil {
            return err
        }
        r.stmts[name] = stmt
    }

    return nil
}

func (r *PostgresInstanceRepository) FindByID(ctx context.Context, id instance.ID) (*instance.Instance, error) {
    stmt := r.stmts["findByID"]
    // Use prepared statement
    row := stmt.QueryRowContext(ctx, id.String())
    // ...
}
```

**Batch Inserts** (transiciones):
```go
func (r *PostgresInstanceRepository) SaveTransitionsBatch(ctx context.Context, transitions []Transition) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO workflow_transitions (id, instance_id, event, from_state, to_state, actor, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `)
    if err != nil {
        return err
    }
    defer stmt.Close()

    for _, t := range transitions {
        _, err := stmt.ExecContext(ctx,
            t.ID(), t.InstanceID(), t.Event(), t.From(), t.To(), t.Actor(), t.Timestamp(),
        )
        if err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

### 8.2 Redis Cache

**Pipeline** (batch operations):
```go
func (c *InstanceCache) GetBatch(ctx context.Context, ids []instance.ID) ([]*instance.Instance, error) {
    pipe := c.client.Pipeline()

    cmds := make([]*redis.StringCmd, len(ids))
    for i, id := range ids {
        key := fmt.Sprintf("instance:%s", id.String())
        cmds[i] = pipe.Get(ctx, key)
    }

    _, err := pipe.Exec(ctx)
    if err != nil && err != redis.Nil {
        return nil, err
    }

    instances := make([]*instance.Instance, 0, len(ids))
    for _, cmd := range cmds {
        data, err := cmd.Bytes()
        if err == redis.Nil {
            continue
        }
        if err != nil {
            return nil, err
        }

        inst, _ := c.serializer.DeserializeInstance(data)
        instances = append(instances, inst)
    }

    return instances, nil
}
```

**Compression** (para payloads grandes):
```go
type Serializer struct {
    compress bool
}

func (s *Serializer) SerializeInstance(inst *instance.Instance) ([]byte, error) {
    data, err := json.Marshal(inst)
    if err != nil {
        return nil, err
    }

    if s.compress && len(data) > 1024 {
        var buf bytes.Buffer
        w := gzip.NewWriter(&buf)
        w.Write(data)
        w.Close()
        return buf.Bytes(), nil
    }

    return data, nil
}
```

**TTL Sliding** (renovar TTL en accesos):
```go
func (c *InstanceCache) Get(ctx context.Context, id instance.ID) (*instance.Instance, error) {
    key := fmt.Sprintf("instance:%s", id.String())

    pipe := c.client.Pipeline()
    getCmd := pipe.Get(ctx, key)
    pipe.Expire(ctx, key, c.ttl) // Renovar TTL

    _, err := pipe.Exec(ctx)
    if err != nil {
        return nil, err
    }

    data, err := getCmd.Bytes()
    if err != nil {
        return nil, err
    }

    return c.serializer.DeserializeInstance(data)
}
```

### 8.3 Aplicación

**Goroutine Pool** (para webhooks):
```go
type WebhookDispatcher struct {
    workers   int
    queue     chan *WebhookTask
    wg        sync.WaitGroup
}

func NewWebhookDispatcher(workers int) *WebhookDispatcher {
    d := &WebhookDispatcher{
        workers: workers,
        queue:   make(chan *WebhookTask, 1000),
    }

    for i := 0; i < workers; i++ {
        d.wg.Add(1)
        go d.worker()
    }

    return d
}

func (d *WebhookDispatcher) worker() {
    defer d.wg.Done()

    for task := range d.queue {
        d.processWebhook(task)
    }
}

func (d *WebhookDispatcher) Dispatch(task *WebhookTask) {
    d.queue <- task
}
```

**Context Timeout**:
```go
func (uc *TriggerEventUseCase) Execute(ctx context.Context, cmd Command) error {
    // Timeout global para el use case
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    // ...
}
```

**Eager Loading** (evitar N+1):
```go
func (r *PostgresInstanceRepository) FindByWorkflowWithHistory(
    ctx context.Context,
    workflowID string,
) ([]*instance.Instance, error) {
    // Single query con JOIN
    query := `
        SELECT
            i.*,
            COALESCE(json_agg(t.*) FILTER (WHERE t.id IS NOT NULL), '[]') as transitions
        FROM workflow_instances i
        LEFT JOIN workflow_transitions t ON t.instance_id = i.id
        WHERE i.workflow_id = $1
        GROUP BY i.id
        ORDER BY i.created_at DESC
    `

    // Parse results
    // ...
}
```

### 8.4 Observabilidad

**Prometheus Metrics**:
```go
var (
    transitionDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "flowengine_transition_duration_seconds",
            Help:    "Duration of state transitions",
            Buckets: prometheus.DefBuckets,
        },
        []string{"workflow_id", "event"},
    )

    lockWaitDuration = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "flowengine_lock_wait_duration_seconds",
            Help:    "Time spent waiting for locks",
            Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
        },
    )

    cacheHitRate = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "flowengine_cache_requests_total",
            Help: "Cache hit/miss count",
        },
        []string{"result"}, // "hit" or "miss"
    )
)

func init() {
    prometheus.MustRegister(transitionDuration, lockWaitDuration, cacheHitRate)
}

// Usage
func (uc *TriggerEventUseCase) Execute(ctx context.Context, cmd Command) error {
    start := time.Now()
    defer func() {
        duration := time.Since(start).Seconds()
        transitionDuration.WithLabelValues(cmd.WorkflowID, cmd.EventName).Observe(duration)
    }()

    // ...
}
```

**Structured Logging**:
```go
func (uc *TriggerEventUseCase) Execute(ctx context.Context, cmd Command) error {
    log := uc.logger.With(
        "use_case", "trigger_event",
        "instance_id", cmd.InstanceID,
        "event", cmd.EventName,
        "actor", cmd.ActorID,
    )

    log.Info("executing transition")

    // ...

    if err != nil {
        log.Error("transition failed", "error", err)
        return err
    }

    log.Info("transition completed",
        "previous_state", previousState,
        "current_state", currentState,
        "duration_ms", time.Since(start).Milliseconds(),
    )

    return nil
}
```

---

## 9. Configuración y Deployment

### 9.1 Variables de Entorno

```bash
# config/.env.example

# Application
APP_NAME=flowengine
APP_ENV=production
APP_PORT=8080
LOG_LEVEL=info

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=flowengine
POSTGRES_USER=flowengine
POSTGRES_PASSWORD=secret
POSTGRES_SSL_MODE=require
POSTGRES_MAX_CONNS=25
POSTGRES_MAX_IDLE_CONNS=5

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_CACHE_TTL=5m
REDIS_MAX_RETRIES=3

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_EXCHANGE=flowengine.events
RABBITMQ_QUEUE_PREFIX=flowengine

# Security
JWT_SECRET=your-secret-key
API_KEYS=key1,key2,key3

# Performance
RATE_LIMIT_RPS=100
REQUEST_TIMEOUT=30s
SHUTDOWN_TIMEOUT=30s

# Observability
METRICS_ENABLED=true
METRICS_PORT=9090
TRACING_ENABLED=true
JAEGER_ENDPOINT=http://localhost:14268/api/traces

# Features
ASYNC_DB_WRITES=true
DISTRIBUTED_LOCKS=true
```

### 9.2 Configuración Estructurada

```go
// config/config.go
package config

import (
    "time"

    "github.com/spf13/viper"
)

type Config struct {
    App         AppConfig
    Postgres    PostgresConfig
    Redis       RedisConfig
    RabbitMQ    RabbitMQConfig
    Security    SecurityConfig
    Performance PerformanceConfig
    Observability ObservabilityConfig
}

type AppConfig struct {
    Name        string
    Environment string
    Port        int
    LogLevel    string
}

type PostgresConfig struct {
    Host        string
    Port        int
    Database    string
    User        string
    Password    string
    SSLMode     string
    MaxConns    int
    MaxIdleConns int
}

type RedisConfig struct {
    Addr       string
    Password   string
    DB         int
    CacheTTL   time.Duration
    MaxRetries int
}

type RabbitMQConfig struct {
    URL          string
    Exchange     string
    QueuePrefix  string
}

type SecurityConfig struct {
    JWTSecret string
    APIKeys   []string
}

type PerformanceConfig struct {
    RateLimitRPS    int
    RequestTimeout  time.Duration
    ShutdownTimeout time.Duration
    AsyncDBWrites   bool
}

type ObservabilityConfig struct {
    MetricsEnabled bool
    MetricsPort    int
    TracingEnabled bool
    JaegerEndpoint string
}

func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("./config")
    viper.AddConfigPath(".")

    viper.AutomaticEnv()
    viper.SetEnvPrefix("FLOWENGINE")

    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }

    return &config, nil
}
```

### 9.3 Dockerfile

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o flowengine ./cmd/api

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary
COPY --from=builder /app/flowengine .

# Copy configs
COPY --from=builder /app/config ./config

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

CMD ["./flowengine"]
```

### 9.4 Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  flowengine:
    build: .
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - APP_ENV=production
      - POSTGRES_HOST=postgres
      - REDIS_ADDR=redis:6379
      - RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
    depends_on:
      - postgres
      - redis
      - rabbitmq
    restart: unless-stopped
    networks:
      - flowengine-network

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: flowengine
      POSTGRES_USER: flowengine
      POSTGRES_PASSWORD: secret
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - flowengine-network

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    networks:
      - flowengine-network

  rabbitmq:
    image: rabbitmq:3-management-alpine
    ports:
      - "5672:5672"
      - "15672:15672"
    environment:
      RABBITMQ_DEFAULT_USER: guest
      RABBITMQ_DEFAULT_PASS: guest
    volumes:
      - rabbitmq-data:/var/lib/rabbitmq
    networks:
      - flowengine-network

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./deployments/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    networks:
      - flowengine-network

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin
    volumes:
      - grafana-data:/var/lib/grafana
    networks:
      - flowengine-network

volumes:
  postgres-data:
  redis-data:
  rabbitmq-data:
  prometheus-data:
  grafana-data:

networks:
  flowengine-network:
    driver: bridge
```

### 9.5 Kubernetes Deployment

```yaml
# deployments/k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flowengine
  labels:
    app: flowengine
spec:
  replicas: 3
  selector:
    matchLabels:
      app: flowengine
  template:
    metadata:
      labels:
        app: flowengine
    spec:
      containers:
      - name: flowengine
        image: flowengine:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: APP_ENV
          value: "production"
        - name: POSTGRES_HOST
          valueFrom:
            configMapKeyRef:
              name: flowengine-config
              key: postgres_host
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: flowengine-secrets
              key: postgres_password
        - name: REDIS_ADDR
          valueFrom:
            configMapKeyRef:
              name: flowengine-config
              key: redis_addr
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /api/v1/health
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /api/v1/health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10

---
apiVersion: v1
kind: Service
metadata:
  name: flowengine
spec:
  selector:
    app: flowengine
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
  type: LoadBalancer

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: flowengine-config
data:
  postgres_host: "postgres-service"
  redis_addr: "redis-service:6379"
  log_level: "info"

---
apiVersion: v1
kind: Secret
metadata:
  name: flowengine-secrets
type: Opaque
stringData:
  postgres_password: "secret"
  jwt_secret: "your-jwt-secret"
```

---

## 10. Seguridad

### 10.1 Autenticación y Autorización

**JWT Middleware**:
```go
func JWTAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, ErrorResponse{Error: "Missing authorization header"})
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")

        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method")
            }
            return []byte(jwtSecret), nil
        })

        if err != nil || !token.Valid {
            c.JSON(401, ErrorResponse{Error: "Invalid token"})
            c.Abort()
            return
        }

        claims := token.Claims.(jwt.MapClaims)
        c.Set("user_id", claims["user_id"])
        c.Set("roles", claims["roles"])

        c.Next()
    }
}
```

**Role-based Access Control**:
```go
func RequireRole(requiredRole string) gin.HandlerFunc {
    return func(c *gin.Context) {
        roles, _ := c.Get("roles")
        roleList := roles.([]string)

        hasRole := false
        for _, role := range roleList {
            if role == requiredRole {
                hasRole = true
                break
            }
        }

        if !hasRole {
            c.JSON(403, ErrorResponse{Error: "Insufficient permissions"})
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### 10.2 Rate Limiting

```go
func RateLimit(limit int) gin.HandlerFunc {
    limiter := rate.NewLimiter(rate.Limit(limit), limit*2)

    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.JSON(429, ErrorResponse{Error: "Rate limit exceeded"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### 10.3 Input Validation

```go
type CreateInstanceRequest struct {
    WorkflowID string                 `json:"workflow_id" binding:"required,alphanum,max=100"`
    ActorID    string                 `json:"actor_id" binding:"required,max=255"`
    ActorRole  string                 `json:"actor_role" binding:"required,oneof=radicador asignador gestionador revisor aprobador"`
    Data       map[string]interface{} `json:"data" binding:"required"`
}

func (r *CreateInstanceRequest) Validate() error {
    if len(r.Data) > 100 {
        return fmt.Errorf("data fields exceed limit")
    }

    // Sanitize data
    for key, value := range r.Data {
        if str, ok := value.(string); ok {
            r.Data[key] = sanitize(str)
        }
    }

    return nil
}
```

---

## 11. Ejemplo Completo: Flujo de Radicación

### 11.1 Workflow YAML

Ver `config/templates/radicacion.yaml` en la sección 3.3.6

### 11.2 Uso Completo (cURL)

```bash
# 1. Crear instancia
curl -X POST http://localhost:8080/api/v1/instances \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "workflow_id": "radicacion",
    "actor_id": "user123",
    "actor_role": "radicador",
    "data": {
      "tipo": "PQRD",
      "remitente": "Juan Pérez",
      "email": "juan@example.com",
      "asunto": "Solicitud de información"
    }
  }'

# Response:
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "workflow_id": "radicacion",
  "current_state": "radicar",
  "status": "running",
  "version": 1,
  "created_at": "2025-01-15T10:00:00Z"
}

# 2. Generar radicado
curl -X POST http://localhost:8080/api/v1/instances/550e8400-e29b-41d4-a716-446655440000/events \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "event": "generar_radicado",
    "actor": "user123"
  }'

# Response:
{
  "instance_id": "550e8400-e29b-41d4-a716-446655440000",
  "previous_state": "radicar",
  "current_state": "asignar",
  "version": 2
}

# 3. Asignar gestor
curl -X POST http://localhost:8080/api/v1/instances/550e8400-e29b-41d4-a716-446655440000/events \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "event": "asignar_gestor",
    "actor": "user456",
    "data": {
      "gestor_asignado": "user789"
    }
  }'

# 4. Ver historial
curl http://localhost:8080/api/v1/instances/550e8400-e29b-41d4-a716-446655440000/history \
  -H "Authorization: Bearer $TOKEN"

# Response:
{
  "transitions": [
    {
      "event": "generar_radicado",
      "from_state": "radicar",
      "to_state": "asignar",
      "actor": "user123",
      "created_at": "2025-01-15T10:01:00Z"
    },
    {
      "event": "asignar_gestor",
      "from_state": "asignar",
      "to_state": "gestionar",
      "actor": "user456",
      "created_at": "2025-01-15T10:05:00Z"
    }
  ]
}

# 5. Query: todas las radicaciones en gestión
curl -X POST http://localhost:8080/api/v1/queries/instances \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "workflow_id": "radicacion",
    "states": ["gestionar"],
    "limit": 50
  }'
```

---

## 12. Métricas y Monitoreo

### 12.1 Métricas Clave

| Métrica | Tipo | Descripción |
|---------|------|-------------|
| `flowengine_transition_duration_seconds` | Histogram | Duración de transiciones |
| `flowengine_lock_wait_duration_seconds` | Histogram | Tiempo esperando locks |
| `flowengine_cache_requests_total{result="hit\|miss"}` | Counter | Cache hit/miss rate |
| `flowengine_instances_total{status="running\|completed\|failed"}` | Gauge | Instancias activas por estado |
| `flowengine_http_requests_total{method,path,status}` | Counter | Requests HTTP |
| `flowengine_http_request_duration_seconds` | Histogram | Latencia HTTP |
| `flowengine_db_connections{state="open\|idle"}` | Gauge | Conexiones DB |
| `flowengine_event_dispatched_total{type}` | Counter | Eventos publicados |

### 12.2 Prometheus Config

```yaml
# deployments/prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'flowengine'
    static_configs:
      - targets: ['flowengine:9090']
```

### 12.3 Grafana Dashboard

**Panel: Transiciones por Segundo**
```promql
rate(flowengine_transition_duration_seconds_count[1m])
```

**Panel: Latencia p95/p99**
```promql
histogram_quantile(0.95, rate(flowengine_transition_duration_seconds_bucket[5m]))
histogram_quantile(0.99, rate(flowengine_transition_duration_seconds_bucket[5m]))
```

**Panel: Cache Hit Rate**
```promql
rate(flowengine_cache_requests_total{result="hit"}[5m]) /
rate(flowengine_cache_requests_total[5m]) * 100
```

---

## Apéndices

### A. Glosario

- **Workflow**: Definición de un proceso con estados y transiciones
- **Instance**: Ejecución específica de un workflow
- **State**: Punto en el workflow donde una instancia puede estar
- **Event**: Trigger que causa una transición de estado
- **Transition**: Cambio de un estado a otro
- **Aggregate**: Cluster de objetos de dominio tratados como una unidad
- **Value Object**: Objeto inmutable identificado por sus valores
- **Entity**: Objeto con identidad única
- **Repository**: Abstracción de persistencia
- **Use Case**: Caso de uso de la aplicación (application layer)
- **Port**: Interface que define un contrato
- **Adapter**: Implementación de un port
- **Optimistic Locking**: Estrategia de concurrencia basada en versiones

### B. Referencias

- [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/)
- [Domain-Driven Design](https://www.domainlanguage.com/ddd/)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [looplab/fsm](https://github.com/looplab/fsm)
- [Go Testing](https://go.dev/doc/tutorial/add-a-test)

### C. Decisiones de Arquitectura (ADR)

Ver `docs/architecture/adr/` para Architecture Decision Records detallados.

---

**Version**: 3.0
**Fecha**: 2026-03-21
**Autores**: FlowEngine Team

**Changelog v3.0** (2026-03-21):
- Motor de Guards (14 built-in) y Actions (10 built-in) con registry extensible
- Required Data: campos obligatorios por transicion validados en dominio
- JSON:API 1.0 en todas las respuestas
- JWT authentication en todos los endpoints protegidos
- Mapeo de errores de dominio a HTTP status (DomainError → JSON:API errors)
- Paginacion DB-level con COUNT(*) OVER() (reemplaza paginacion in-memory)
- Endpoint GET /instances/:id/history para historial de transiciones
- WebhookDispatcher con HMAC signing, reintentos y entrega asincrona
- MultiDispatcher + LogDispatcher para composicion de event bus
- Timer retry policy con backoff exponencial (30s, 60s, 120s...)
- Timer repository in-memory (scheduler funciona sin Postgres)
- Postgres workflow repo completo (FindByName, FindAll, etc. — eran stubs)
- Cache hydration en workflow FindByID (era no-op)
- JOIN workflows para resolver workflow name en instancias (era "unknown")
- Refactoring: instanceRow/workflowRow, columnConstants, error handling
- 331 tests (40 handler tests, 10 webhook tests, 281 domain/infra tests)
- Coleccion Bruno con 29 requests cubriendo los 12 endpoints
- Documentacion README y API quickstart actualizados

**Changelog v2.0** (2025-11-10):
- R17: Agregado sistema de subestados jerarquicos
- R18: Agregado sistema de escalamientos manuales
- R19: Agregado soporte para reclasificacion de instancias
- R20: Agregado sistema de guards de transicion avanzados
- R21: Agregado sistema de plantillas de workflows
- R22: Agregado import/export de workflows en YAML/JSON
- R23: Agregado metadata extendida en transiciones con validacion

---

## Apendice: Estado de Implementacion

| Componente | Estado | Archivos |
|------------|--------|----------|
| **Domain: Workflow** (States, Events, Version) | ✅ Completo | `internal/domain/workflow/` |
| **Domain: Instance** (Transitions, Data, SubStates) | ✅ Completo | `internal/domain/instance/` |
| **Domain: Guards/Actions Engine** | ✅ Completo | `internal/domain/instance/engine.go` |
| **Domain: Events** (8 event types + Dispatcher interface) | ✅ Completo | `internal/domain/event/` |
| **Domain: Timer** (con retry policy) | ✅ Completo | `internal/domain/timer/` |
| **Domain: Shared** (ID, Timestamp, Pagination, Errors) | ✅ Completo | `internal/domain/shared/` |
| **Domain: Actor/Roles** | ❌ Vacio | `internal/domain/actor/` |
| **App: Create/Get/List Workflow** | ✅ Completo | `internal/application/workflow/` |
| **App: Create/Get/List/Transition/Clone Instance** | ✅ Completo | `internal/application/instance/` |
| **Infra: HTTP Handlers + JSON:API** | ✅ Completo | `internal/infrastructure/http/handler/` |
| **Infra: JWT Auth Middleware** | ✅ Completo | `internal/infrastructure/http/middleware/` |
| **Infra: Postgres Repos** (Instance + Workflow + Timer) | ✅ Completo | `internal/infrastructure/persistence/postgres/` |
| **Infra: Memory Repos** (Instance + Workflow + Timer) | ✅ Completo | `internal/infrastructure/persistence/memory/` |
| **Infra: Redis Cache** | ✅ Completo | `internal/infrastructure/cache/` |
| **Infra: YAML Parser** | ✅ Completo | `internal/infrastructure/parser/yaml/` |
| **Infra: MultiDispatcher** | ✅ Completo | `internal/infrastructure/messaging/multi_dispatcher.go` |
| **Infra: WebhookDispatcher** | ✅ Completo | `internal/infrastructure/messaging/webhook_dispatcher.go` |
| **Infra: LogDispatcher** | ✅ Completo | `internal/infrastructure/messaging/log_dispatcher.go` |
| **Infra: Scheduler Worker** (con retry) | ✅ Completo | `internal/infrastructure/scheduler/worker.go` |
| **Infra: RBAC** | ❌ No implementado | — |
| **Infra: Prometheus/OpenTelemetry** | ❌ No implementado | — |
| **Infra: RabbitMQ/Kafka Dispatcher** | ❌ No implementado | MultiDispatcher listo para agregar |
| **Infra: Config Management** | ❌ Env vars directo | — |
| **Testing** | ✅ 331 tests | handler, webhook, domain, parser |
| **Bruno Collection** | ✅ 29 requests | `bruno/FlowEngine/` |
