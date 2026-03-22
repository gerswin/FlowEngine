# FlowEngine - Plan de Implementación

## Contexto del Proyecto

**Stack Tecnológico**: Go 1.24+, Gin, PostgreSQL 15+, Redis 7+, Event Dispatcher (WebhookDispatcher, LogDispatcher), custom FSM, Docker, Kubernetes

**Componentes Principales**:
- Domain Layer (Workflow, Instance aggregates)
- Application Layer (Use Cases)
- Infrastructure Layer (PostgreSQL, Redis, HTTP, Messaging)
- REST API completa
- Sistema de eventos externos
- Workflows configurables vía YAML/JSON

**Funcionalidades Core**:
- Persistencia híbrida (Redis + PostgreSQL)
- Múltiples instancias en paralelo
- Subprocesos jerárquicos
- Sistema de actores y roles
- Timers y escalamientos
- Optimistic locking para concurrencia

---

## Plan de Implementación

### Fase 1: Setup y Estructura Base del Proyecto

- [ ] 1.1 Configuración inicial del proyecto
  - Ejecutar `go mod init github.com/LaFabric-LinkTIC/FlowEngine`
  - Agregar dependencias principales a `go.mod`:
    - FlowEngine custom FSM (`internal/domain/workflow/fsm.go`)
    - `github.com/gin-gonic/gin`
    - `github.com/redis/go-redis/v9`
    - `github.com/lib/pq`
    - `github.com/google/uuid`
    - `gopkg.in/yaml.v3`
    - `github.com/stretchr/testify`
  - Crear archivo `.gitignore` (binarios, vendor, coverage, .env)
  - Crear archivo `README.md` con descripción básica
  - _Requirements: Arquitectura Hexagonal_

- [ ] 1.2 Estructura de directorios hexagonal
  - Crear directorios según arquitectura:
    - `internal/domain/` (workflow, instance, actor, event, shared, timer)
    - `internal/application/` (workflow, instance, subprocess, query, webhook)
    - `internal/infrastructure/` (persistence, messaging, http, config, scheduler, observability, di)
    - `pkg/` (ports, common, testing)
    - `cmd/` (api, worker, cli)
    - `config/`, `docs/`, `test/`, `scripts/`, `deployments/`
  - Crear archivos `.gitkeep` en directorios vacíos
  - _Requirements: Clean Architecture, Arquitectura Hexagonal_

- [ ] 1.3 Makefile y scripts de desarrollo
  - Crear `Makefile` con comandos:
    - `make build` (compilar binario)
    - `make test` (ejecutar tests unitarios)
    - `make test-integration` (tests de integración)
    - `make test-coverage` (coverage report)
    - `make lint` (golangci-lint)
    - `make run` (ejecutar API server)
    - `make migrate-up` / `make migrate-down`
    - `make docker-build` / `make docker-up`
  - Crear script `scripts/generate-mocks.sh` para mockery
  - _Requirements: Mejores prácticas_

### Fase 2: Domain Layer - Shared Types

- [ ] 2.1 Shared value objects
  - Implementar `internal/domain/shared/id.go`:
    - Type `ID` basado en UUID
    - Constructor `NewID() ID`
    - `ParseID(string) (ID, error)`
    - `String() string`, `IsValid() bool`
  - Implementar `internal/domain/shared/timestamp.go`:
    - Type `Timestamp` wrapeando `time.Time`
    - `Now() Timestamp`, `From(time.Time) Timestamp`
    - Métodos `Before()`, `After()`, `Equal()`
  - _Requirements: DDD Value Objects_

- [ ] 2.2 Shared domain errors
  - Crear `internal/domain/shared/errors.go`:
    - Definir errores base: `ErrNotFound`, `ErrInvalidInput`, `ErrConflict`
    - Type `DomainError struct` con `Code`, `Message`, `Cause`, `Context`
    - Constructor `NewDomainError(code, message, cause) *DomainError`
    - Método `WithContext(key, value) *DomainError`
    - Implementar `Error() string` y `Unwrap() error`
  - _Requirements: Manejo de Errores por Capa_

- [ ]* 2.3 Tests unitarios de shared types
  - Test `internal/domain/shared/id_test.go`:
    - `TestNewID_GeneratesValidUUID`
    - `TestParseID_ValidInput`
    - `TestParseID_InvalidInput`
  - Test `internal/domain/shared/timestamp_test.go`:
    - `TestTimestamp_Comparison`
    - `TestTimestamp_Serialization`
  - Coverage objetivo: >90%
  - _Requirements: Estrategia de Testing_

### Fase 3: Domain Layer - Workflow Aggregate

- [ ] 3.1 Workflow value objects
  - Implementar `internal/domain/workflow/state.go`:
    - Type `State struct` con campos: id, name, description, timeout, onTimeout, isFinal
    - Constructor `NewState(id, name string) (State, error)`
    - Métodos builder: `WithTimeout()`, `AsFinal()`, `WithDescription()`
    - Método `Validate() error` (verificar ID pattern: `^[a-z][a-z0-9_]*$`)
    - Método `Equals(other State) bool` (comparación por ID)
  - Implementar `internal/domain/workflow/event.go`:
    - Type `Event struct` con: name, sources ([]State), destination (State), validators
    - Constructor `NewEvent(name, sources, destination)`
    - Getters: `Name()`, `Sources()`, `Destination()`
  - Implementar `internal/domain/workflow/version.go`:
    - Type `Version struct` con campo semver
    - `NewVersion(major, minor, patch int) Version`
  - _Requirements: DDD Value Objects, Inmutabilidad_

- [ ] 3.2 Workflow aggregate raíz
  - Implementar `internal/domain/workflow/workflow.go`:
    - Type `Workflow struct` con: id, name, version, initialState, states, events, createdAt, updatedAt
    - Constructor `NewWorkflow(name, initialState) (*Workflow, error)`
    - Método `AddState(state State) error` (validar no duplicados)
    - Método `AddEvent(event Event) error` (validar estados existen)
    - Método `CanTransition(from State, event Event) bool` (lógica de validación)
    - Método `FindEvent(name string) (Event, error)`
    - Getters inmutables con copia defensiva: `States()`, `Events()`
  - _Requirements: DDD Aggregates, Invariantes de Negocio_

- [ ] 3.3 Workflow repository port
  - Crear `internal/domain/workflow/repository.go`:
    - Interface `Repository` con métodos:
      - `Save(ctx, *Workflow) error`
      - `FindByID(ctx, ID) (*Workflow, error)`
      - `FindAll(ctx) ([]*Workflow, error)`
      - `Delete(ctx, ID) error`
    - Documentar contratos de cada método
  - _Requirements: Ports & Adapters_

- [ ] 3.4 Workflow domain errors
  - Crear `internal/domain/workflow/errors.go`:
    - `ErrInvalidWorkflow`, `ErrStateNotFound`, `ErrEventNotFound`
    - `ErrStateAlreadyExists`, `ErrInvalidStateID`, `ErrEmptyStateName`
    - `ErrCyclicDependency`, `ErrInvalidTransition`
  - _Requirements: Domain Errors_

- [ ]* 3.5 Tests unitarios de Workflow
  - Test `internal/domain/workflow/workflow_test.go`:
    - `TestNewWorkflow_Success`
    - `TestWorkflow_AddState_Success`
    - `TestWorkflow_AddState_Duplicate_Error`
    - `TestWorkflow_AddEvent_ValidStates_Success`
    - `TestWorkflow_AddEvent_InvalidState_Error`
    - `TestWorkflow_CanTransition_ValidEvent_ReturnsTrue`
    - `TestWorkflow_CanTransition_InvalidEvent_ReturnsFalse`
  - Test `internal/domain/workflow/state_test.go`:
    - Table-driven test para validación de State
    - Test de `Equals()` method
  - Coverage objetivo: >90%
  - _Requirements: Unit Testing, Table-Driven Tests_

### Fase 4: Domain Layer - Instance Aggregate

- [ ] 4.1 Instance value objects
  - Implementar `internal/domain/instance/status.go`:
    - Type `Status` enum: Running, Paused, Completed, Canceled, Failed
    - Métodos `IsActive() bool`, `IsFinal() bool`
    - `String() string` para serialización
  - Implementar `internal/domain/instance/version.go`:
    - Type `Version struct` con campo int64
    - Constructor `NewVersion() Version` (inicia en 1)
    - Método `Increment() Version`
    - Método `Value() int64`, `Equals(other Version) bool`
  - Implementar `internal/domain/instance/data.go`:
    - Type `Data struct` wrapeando `map[string]interface{}`
    - Constructor `NewData()`, `NewDataFromMap(m)`
    - Métodos `Get(key)`, `Set(key, value)`, `ToMap()`
  - Implementar `internal/domain/instance/variables.go`:
    - Similar a Data, pero para variables de workflow
  - Implementar `internal/domain/instance/sub_state.go` (R17):
    - Type `SubState struct` con: id, name, description
    - Constructor `NewSubState(id, name string) (SubState, error)`
    - Método `Validate() error` (verificar ID pattern)
    - Método `Equals(other SubState) bool`
  - Implementar `internal/domain/instance/transition_metadata.go` (R23):
    - Type `TransitionMetadata struct` con: reason, feedback, metadata map
    - Constructor `NewTransitionMetadata(reason, feedback string, metadata map)`
    - Método `Validate(schema *MetadataSchema) error` (validar tipos, required, constraints)
    - Type `MetadataSchema struct` para definir validaciones (required, optional, types)
  - _Requirements: DDD Value Objects, Optimistic Locking, R17, R23_

- [ ] 4.2 Instance transition entity
  - Implementar `internal/domain/instance/transition.go`:
    - Type `Transition struct` con: id, from, to, event, actor, timestamp, data
    - Constructor `NewTransition(from, to, event, actor, timestamp)`
    - Getters: `ID()`, `From()`, `To()`, `Event()`, `Actor()`, `Timestamp()`
    - Método `Duration() time.Duration` (si hay timestamp de fin)
  - _Requirements: DDD Entities_

- [ ] 4.3 Instance aggregate raíz
  - Implementar `internal/domain/instance/instance.go`:
    - Type `Instance struct` con:
      - id, workflowID, parentID (para subprocesos)
      - currentState, previousState
      - currentSubState, previousSubState (*SubState) (R17)
      - version (optimistic locking)
      - status, data, variables
      - history ([]Transition)
      - domainEvents ([]event.DomainEvent)
      - createdAt, updatedAt, completedAt
    - Constructor `NewInstance(workflowID, initialState, data) (*Instance, error)`
    - Método `Transition(ctx, wf, eventName, actorID) error`:
      - Validar status == Running
      - Verificar transición válida con workflow
      - Cambiar estado
      - Incrementar version
      - Agregar a history
      - Si estado final, llamar `complete()`
      - Generar evento de dominio `StateChanged`
    - Método `TransitionWithMetadata(ctx, wf, eventName, actorID, reason, feedback string, metadata map) error` (R23):
      - Validar metadata según schema del evento
      - Ejecutar transición normal
      - Incluir reason, feedback, metadata en transition history
    - Método `TransitionSubState(ctx, wf, newSubState SubState) error` (R17):
      - Cambiar currentSubState sin afectar currentState
      - Guardar previousSubState
      - Incrementar version
      - Agregar a history con from_sub_state y to_sub_state
      - Generar evento SubStateChanged
    - Método `Reclassify(newType, reason, actor string) error` (R19):
      - Actualizar data.tipo
      - Mantener currentState sin cambios
      - Incrementar version
      - Crear transición especial con event="reclassify"
      - Generar evento DocumentReclassified
    - Métodos `Pause()`, `Resume()`, `Cancel()` error
    - Métodos `SetVariable(key, value)`, `GetVariable(key)`
    - Métodos `CurrentSubState()`, `PreviousSubState()` (R17)
    - Método `DomainEvents() []event.DomainEvent` (retorna y limpia)
    - Getters inmutables
  - _Requirements: DDD Aggregates, Domain Events, Business Logic, R17, R19, R23_

- [ ] 4.4 Instance repository port
  - Crear `internal/domain/instance/repository.go`:
    - Interface `Repository` con métodos:
      - `Save(ctx, *Instance) error`
      - `SaveWithVersion(ctx, *Instance, expectedVersion Version) error`
      - `FindByID(ctx, ID) (*Instance, error)`
      - `FindByWorkflow(ctx, workflowID string) ([]*Instance, error)`
      - `FindByState(ctx, state string) ([]*Instance, error)`
      - `FindByActor(ctx, actorID string) ([]*Instance, error)`
      - `Query(ctx, spec Specification) ([]*Instance, int, error)`
      - `Delete(ctx, ID) error`
    - Interface `Specification` para queries complejas
  - _Requirements: Repository Pattern, Specification Pattern_

- [ ] 4.5 Instance domain errors
  - Crear `internal/domain/instance/errors.go`:
    - `ErrNotFound`, `ErrInvalidTransition`, `ErrVersionConflict`
    - `ErrInstanceNotRunning`, `ErrInstanceCompleted`, `ErrAlreadyFinished`
    - `ErrCannotPause`, `ErrCannotResume`
  - _Requirements: Domain Errors_

- [ ]* 4.6 Tests unitarios de Instance
  - Test `internal/domain/instance/instance_test.go`:
    - `TestNewInstance_Success`
    - `TestInstance_Transition_Success` (happy path)
    - `TestInstance_Transition_InvalidEvent_Error`
    - `TestInstance_Transition_WrongState_Error`
    - `TestInstance_Transition_NotRunning_Error`
    - `TestInstance_Transition_FinalState_Completes`
    - `TestInstance_Transition_IncrementsVersion`
    - `TestInstance_Transition_GeneratesDomainEvent`
    - `TestInstance_Pause_Success`
    - `TestInstance_Resume_Success`
    - `TestInstance_Cancel_Success`
    - `TestInstance_Variables_GetSet`
  - Coverage objetivo: >90%
  - _Requirements: Unit Testing, Domain Logic Testing_

### Fase 5: Domain Layer - Event System

- [ ] 5.1 Domain events definitions
  - Crear `internal/domain/event/event.go`:
    - Interface `DomainEvent` con métodos:
      - `Type() string`
      - `AggregateID() string`
      - `OccurredAt() time.Time`
      - `Payload() map[string]interface{}`
  - Implementar eventos concretos:
    - `InstanceCreated` struct
    - `StateChanged` struct
    - `InstancePaused` struct
    - `InstanceResumed` struct
    - `InstanceCompleted` struct
    - `InstanceCanceled` struct
  - Cada evento implementa `DomainEvent` interface
  - _Requirements: Domain Events, Event-Driven Architecture_

- [ ] 5.2 Event dispatcher port
  - Crear `internal/domain/event/dispatcher.go`:
    - Interface `Dispatcher` con métodos:
      - `Dispatch(ctx, event DomainEvent) error`
      - `DispatchBatch(ctx, events []DomainEvent) error`
  - Documentar contratos y semántica (async, retry, etc)
  - _Requirements: Ports & Adapters_

- [ ]* 5.3 Tests de domain events
  - Test `internal/domain/event/event_test.go`:
    - Verificar serialización de cada evento
    - Verificar campos requeridos
    - Test de `Payload()` con datos correctos
  - _Requirements: Unit Testing_

### Fase 6: Infrastructure - PostgreSQL Persistence

- [ ] 6.1 Migraciones de base de datos
  - Crear `internal/infrastructure/persistence/postgres/migrations/001_initial.up.sql`:
    - Tabla `workflows` (id, name, description, version, config JSONB, is_template BOOLEAN DEFAULT FALSE, template_id UUID, created_at, updated_at, deleted_at)
    - Tabla `workflow_instances` (id UUID, workflow_id, parent_id, current_state, previous_state, current_sub_state, previous_sub_state, version, status, data JSONB, variables JSONB, current_actor, current_role, created_at, updated_at, completed_at, locked_by, locked_at, lock_expires_at)
    - Tabla `workflow_transitions` (id, instance_id, event, from_state, to_state, from_sub_state, to_sub_state, actor, actor_role, data JSONB, reason TEXT, feedback TEXT, metadata JSONB, duration_ms, created_at)
    - Tabla `workflow_timers` (id, instance_id, state, event_on_timeout, created_at, expires_at, fired_at)
    - Tabla `workflow_escalations` (id, instance_id, department_id, reason TEXT, escalated_by, escalated_at, status, response TEXT, responded_by, responded_at, closed_by, closed_at)
    - Tabla `webhooks` (id, workflow_id, url, events text[], secret, headers JSONB, retry_config JSONB, active, created_at, updated_at)
    - Tabla `external_events` (id, instance_id, event_type, payload JSONB, processed_at, error_message, retry_count, created_at)
    - Índices optimizados:
      - workflow_instances: idx_instances_workflow, idx_instances_parent, idx_instances_status, idx_instances_actor, idx_instances_substates (R17)
      - workflow_transitions: idx_transitions_instance, idx_transitions_has_feedback (R23), idx_transitions_metadata GIN (R23)
      - workflow_escalations: idx_escalations_instance, idx_escalations_department, idx_escalations_status (R18)
      - workflows: idx_workflows_template, idx_workflows_is_template (R21)
    - Constraints y checks:
      - status CHECK IN ('running', 'paused', 'completed', 'canceled', 'failed')
      - escalation_status CHECK IN ('pending', 'responded', 'closed', 'canceled')
      - workflows.template_id FK REFERENCES workflows(id)
  - Crear `001_initial.down.sql` con DROP tables
  - Crear script `scripts/migrate.sh` usando golang-migrate
  - _Requirements: R17, R18, R21, R23, Schema PostgreSQL, Índices, Migraciones_

- [ ] 6.2 PostgreSQL connection y configuración
  - Implementar `internal/infrastructure/persistence/postgres/connection.go`:
    - Función `NewPostgresDB(config PostgresConfig) (*sql.DB, error)`
    - Configurar connection pool (MaxOpenConns=25, MaxIdleConns=5, ConnMaxLifetime=5m)
    - Health check con `db.Ping()`
    - Context support
  - _Requirements: Connection Pooling, Performance_

- [ ] 6.3 Instance repository adapter - Mappers
  - Crear `internal/infrastructure/persistence/postgres/instance_mapper.go`:
    - Type `InstanceMapper struct`
    - Método `ToModel(inst *domain.Instance) *InstanceModel`
    - Método `ToDomain(model *InstanceModel) (*domain.Instance, error)`
    - Type `InstanceModel struct` matching DB schema
    - Serializar/deserializar JSONB (data, variables)
    - Manejar nullable fields (previous_state, completed_at, parent_id)
  - _Requirements: Mapeo Domain ↔ DB, Clean Architecture_

- [ ] 6.4 Instance repository adapter - Implementación
  - Implementar `internal/infrastructure/persistence/postgres/instance_repository.go`:
    - Type `InstanceRepository struct` con db *sql.DB
    - Implementar `Save(ctx, instance) error`:
      - Usar UPSERT (INSERT ... ON CONFLICT DO UPDATE)
      - Mapear domain → DB model
      - Ejecutar query con context
    - Implementar `SaveWithVersion(ctx, instance, expectedVersion) error`:
      - UPDATE con WHERE version = expectedVersion
      - Verificar RowsAffected == 1
      - Si 0 rows, retornar `ErrVersionConflict`
    - Implementar `FindByID(ctx, id) (*Instance, error)`
    - Implementar `FindByWorkflow(ctx, workflowID) ([]*Instance, error)`
    - Implementar `FindByState(ctx, state) ([]*Instance, error)`
    - Implementar `Query(ctx, spec) ([]*Instance, int, error)` con paginación
    - Guardar transitions en `saveTransitions(ctx, instance) error`
  - _Requirements: Repository Adapter, Optimistic Locking, Paginación_

- [ ] 6.5 Workflow repository adapter
  - Implementar `internal/infrastructure/persistence/postgres/workflow_repository.go`:
    - Type `WorkflowRepository struct`
    - Implementar métodos de `workflow.Repository` interface
    - Serializar workflow config como JSONB
    - Soft delete (usar deleted_at)
  - _Requirements: Repository Adapter, Soft Delete_

- [ ]* 6.6 Tests de integración PostgreSQL
  - Crear `test/integration/postgres_test.go`:
    - Setup con testcontainers-go (PostgreSQL container)
    - Ejecutar migraciones en container
    - `TestInstanceRepository_Save_Success`
    - `TestInstanceRepository_SaveWithVersion_OptimisticLock`
    - `TestInstanceRepository_FindByID_NotFound`
    - `TestInstanceRepository_Query_WithFilters`
    - Teardown de container
  - Usar build tag `//go:build integration`
  - _Requirements: Integration Testing, Testcontainers_

### Fase 7: Infrastructure - Redis Cache

- [ ] 7.1 Redis connection
  - Implementar `internal/infrastructure/persistence/redis/connection.go`:
    - Función `NewRedisClient(config RedisConfig) *redis.Client`
    - Configurar opciones (addr, password, DB, MaxRetries)
    - Health check con Ping
  - _Requirements: Redis Client Setup_

- [ ] 7.2 Instance cache adapter
  - Implementar `internal/infrastructure/persistence/redis/instance_cache.go`:
    - Type `InstanceCache struct` con client *redis.Client, ttl time.Duration
    - Método `Get(ctx, id) (*instance.Instance, error)`:
      - Key format: `instance:{id}`
      - Deserializar desde JSON
      - Retornar `ErrNotFound` si redis.Nil
    - Método `Set(ctx, inst) error`:
      - Serializar a JSON
      - SET con TTL
    - Método `Delete(ctx, id) error` (DEL key)
    - Método `InvalidateByWorkflow(ctx, workflowID) error` (SCAN + DEL)
  - _Requirements: Cache Adapter, TTL Management_

- [ ] 7.3 Serializer para cache
  - Implementar `internal/infrastructure/persistence/redis/serializer.go`:
    - Type `Serializer struct` con compress bool
    - Método `SerializeInstance(inst) ([]byte, error)`:
      - Marshal a JSON
      - Si compress y size > 1KB, usar gzip
    - Método `DeserializeInstance(data) (*instance.Instance, error)`:
      - Detectar si está comprimido
      - Deserializar JSON → domain object
  - _Requirements: Serialización, Compresión_

- [ ] 7.4 Distributed lock adapter
  - Implementar `internal/infrastructure/persistence/redis/distributed_lock.go`:
    - Type `DistributedLocker struct` implementando `ports.Locker`
    - Type `Lock struct` implementando `ports.Lock`
    - Método `Lock(ctx, key, ttl) (Lock, error)`:
      - Key format: `lock:{key}`
      - Redis SET NX con TTL
      - UUID como valor del lock
      - Retornar `ErrLockAlreadyHeld` si no se puede adquirir
    - Método `Unlock(ctx) error`:
      - Lua script para atomic check-and-delete
      - Solo el owner puede liberar el lock
    - Método `Refresh(ctx, ttl) error` (extender TTL)
  - _Requirements: Distributed Locking, Atomic Operations_

- [ ]* 7.5 Tests de integración Redis
  - Crear `test/integration/redis_test.go`:
    - Setup con testcontainers-go (Redis container)
    - `TestInstanceCache_GetSet_Success`
    - `TestInstanceCache_Get_NotFound`
    - `TestInstanceCache_TTL_Expiration`
    - `TestDistributedLock_AcquireRelease`
    - `TestDistributedLock_AlreadyHeld_Error`
    - `TestDistributedLock_OnlyOwnerCanRelease`
  - Usar build tag `//go:build integration`
  - _Requirements: Integration Testing, Cache Testing_

### Fase 8: Infrastructure - Hybrid Repository

- [ ] 8.1 Hybrid instance repository
  - Implementar `internal/infrastructure/persistence/hybrid/instance_repository.go`:
    - Type `HybridInstanceRepository struct` con:
      - cache *redis.InstanceCache
      - durable *postgres.InstanceRepository
      - config HybridConfig
    - Type `HybridConfig struct`:
      - CacheTTL, WriteThrough, ReadThrough, AsyncWrite bool
    - Implementar `FindByID(ctx, id) (*Instance, error)`:
      - 1. Try cache.Get()
      - 2. If hit, return
      - 3. If miss, durable.FindByID()
      - 4. If ReadThrough, cache.Set()
      - 5. Return instance
    - Implementar `Save(ctx, inst) error`:
      - 1. cache.Set() (rápido)
      - 2. If AsyncWrite, go durable.Save() en goroutine
      - 3. Else durable.Save() sync
      - 4. Si durable falla, cache.Delete() para rollback
    - Implementar `SaveWithVersion()` similar
    - Delegar queries complejas a durable
  - _Requirements: Hybrid Strategy, Write-Through, Read-Through_

- [ ] 8.2 Cache invalidation strategy
  - Agregar métodos de invalidación:
    - `InvalidateInstance(ctx, id) error`
    - `InvalidateByWorkflow(ctx, workflowID) error`
  - Hook en Save para invalidar cache de queries relacionadas
  - _Requirements: Cache Invalidation_

- [ ]* 8.3 Tests de integración Hybrid
  - Crear `test/integration/hybrid_test.go`:
    - Setup con PostgreSQL + Redis containers
    - `TestHybridRepository_FindByID_CacheHit`
    - `TestHybridRepository_FindByID_CacheMiss_PopulatesCache`
    - `TestHybridRepository_Save_WritesThrough`
    - `TestHybridRepository_SaveWithVersion_CacheInvalidation`
    - Medir hit rate esperado >90%
  - _Requirements: Integration Testing, Cache Performance_

### Fase 9: Application Layer - Use Cases Base

- [ ] 9.1 DTOs para use cases
  - Crear `internal/application/instance/dto.go`:
    - `CreateInstanceCommand struct` (WorkflowID, ActorID, ActorRole, Data)
    - `CreateInstanceResult struct` (InstanceID, InitialState, CreatedAt)
    - `TriggerEventCommand struct` (InstanceID, EventName, ActorID, Data)
    - `TriggerEventResult struct` (InstanceID, PreviousState, CurrentState, Version)
    - `QueryInstancesQuery struct` (WorkflowID, States, Actors, FromDate, ToDate, Status, Limit, Offset)
    - `QueryInstancesResult struct` (Instances, Total, Limit, Offset)
  - Crear `internal/application/workflow/dto.go` similar
  - _Requirements: DTOs, Input Validation_

- [ ] 9.2 CreateInstanceUseCase
  - Implementar `internal/application/instance/create_instance.go`:
    - Type `CreateInstanceUseCase struct` con:
      - instanceRepo, workflowRepo, eventBus, logger (inyección de dependencias)
    - Constructor `NewCreateInstanceUseCase(repo, workflowRepo, eventBus, logger)`
    - Método `Execute(ctx, cmd CreateInstanceCommand) (*CreateInstanceResult, error)`:
      - 1. Validar command
      - 2. Cargar workflow desde workflowRepo
      - 3. Crear instance con workflow.InitialState()
      - 4. Guardar en instanceRepo
      - 5. Publicar domain events vía eventBus
      - 6. Log success
      - 7. Retornar resultado
  - _Requirements: Use Case Pattern, Orchestration_

- [ ] 9.3 TriggerEventUseCase (caso crítico)
  - Implementar `internal/application/instance/trigger_event.go`:
    - Type `TriggerEventUseCase struct` con:
      - instanceRepo, workflowRepo, eventBus, locker, logger
    - Método `Execute(ctx, cmd TriggerEventCommand) (*TriggerEventResult, error)`:
      - 1. Validar command
      - 2. **Adquirir distributed lock** (key: instanceID, ttl: 30s)
      - 3. defer lock.Unlock()
      - 4. Cargar instance desde repo
      - 5. Cargar workflow desde repo
      - 6. Guardar previousState
      - 7. Ejecutar **inst.Transition()** (domain logic)
      - 8. Persistir con instanceRepo.Save()
      - 9. Obtener domain events con inst.DomainEvents()
      - 10. Publicar cada evento vía eventBus
      - 11. Log success con metrics
      - 12. Retornar resultado
    - Manejo de errores específico por tipo
  - _Requirements: Use Case Pattern, Locking, Domain Events_

- [ ] 9.4 QueryInstancesUseCase
  - Implementar `internal/application/instance/query_instances.go`:
    - Type `QueryInstancesUseCase struct` con instanceRepo, logger
    - Método `Execute(ctx, query QueryInstancesQuery) (*QueryInstancesResult, error)`:
      - 1. Validar query (limit <= 100)
      - 2. Construir Specification desde query
      - 3. Ejecutar instanceRepo.Query(spec) con paginación
      - 4. Convertir domain → DTO
      - 5. Retornar resultado con total count
  - _Requirements: Query Pattern, Paginación_

- [ ]* 9.5 Tests unitarios de use cases (con mocks)
  - Crear `internal/application/instance/trigger_event_test.go`:
    - Setup con mocks (mockery):
      - `mockInstanceRepo := mocks.NewInstanceRepository(t)`
      - `mockWorkflowRepo := mocks.NewWorkflowRepository(t)`
      - `mockEventBus := mocks.NewEventDispatcher(t)`
      - `mockLocker := mocks.NewLocker(t)`
    - `TestTriggerEventUseCase_Execute_Success`:
      - Setup expectations con `.On()` y `.Return()`
      - Ejecutar use case
      - Verificar resultado
      - Assert `.AssertExpectations(t)`
    - `TestTriggerEventUseCase_Execute_InstanceNotFound`
    - `TestTriggerEventUseCase_Execute_InvalidTransition`
    - `TestTriggerEventUseCase_Execute_LockFailed`
  - Similar para CreateInstanceUseCase y QueryInstancesUseCase
  - Coverage objetivo: >80%
  - _Requirements: Unit Testing con Mocks, Testify_

### Fase 10: Infrastructure - Event Dispatching (Implementado)

> **NOTA**: RabbitMQ no fue implementado. El sistema de eventos usa MultiDispatcher con WebhookDispatcher y LogDispatcher.

- [x] 10.1 Event dispatcher infrastructure
  - Implementado mediante `MultiDispatcher` que agrega múltiples dispatchers
  - `WebhookDispatcher` para envío de eventos via HTTP webhooks
  - `LogDispatcher` para registro de eventos en logs

- [x] 10.2 Event dispatcher adapter
  - `MultiDispatcher` implementando `event.Dispatcher`
  - Método `Dispatch(ctx, evt DomainEvent) error` delega a todos los dispatchers registrados

- [ ] 10.3 Event subscriber adapter (no implementado - futuro)
  - Pendiente si se requiere suscripción a eventos externos

- [ ]* 10.4 Tests de integración Event Dispatcher
  - Tests para WebhookDispatcher y LogDispatcher
  - _Requirements: Integration Testing_

### Fase 11: Infrastructure - Webhooks

- [ ] 11.1 Webhook client
  - Implementar `internal/infrastructure/messaging/webhook/client.go`:
    - Type `WebhookClient struct` con httpClient *http.Client
    - Método `Send(ctx, webhook WebhookConfig, event DomainEvent) error`:
      - Serializar event a JSON
      - Generar firma HMAC-SHA256 con secret
      - Crear request POST con headers:
        - `Content-Type: application/json`
        - `X-FlowEngine-Signature: sha256={signature}`
        - `X-FlowEngine-Event: {event.Type()}`
        - Custom headers del webhook
      - Ejecutar request con timeout
      - Retry con exponential backoff (max 3 intentos)
    - Método `SendAsync(webhook, event)` en goroutine con worker pool
  - _Requirements: Webhooks, HMAC Signatures, Retry Logic_

- [ ] 11.2 Webhook dispatcher
  - Implementar `internal/infrastructure/messaging/webhook/dispatcher.go`:
    - Type `WebhookDispatcher struct` con client, workers int, queue chan
    - Método `Start()` (iniciar worker pool)
    - Método `Dispatch(webhook, event)` (enviar a queue)
    - Worker goroutines procesando queue
    - Dead letter queue para errores persistentes
  - _Requirements: Worker Pool, Async Processing_

- [ ]* 11.3 Tests de webhooks
  - Crear `test/integration/webhook_test.go`:
    - Setup con httptest server mock
    - `TestWebhookClient_Send_Success`
    - `TestWebhookClient_Send_Retry_On_Failure`
    - `TestWebhookClient_Send_HMAC_Signature`
    - Verificar retry count y backoff
  - _Requirements: Integration Testing, HTTP Mocking_

### Fase 12: Infrastructure - Workflow Loader (YAML/JSON)

- [ ] 12.1 Workflow configuration structs
  - Crear `internal/infrastructure/config/loader/types.go`:
    - Type `WorkflowConfig struct` matching YAML structure:
      - Version, Workflow (WorkflowDefinition)
    - Type `WorkflowDefinition struct`:
      - ID, Name, Description, InitialState
      - States []StateConfig
      - Events []EventConfig
      - Webhooks []WebhookConfig
      - SLA *SLAConfig
    - Type `StateConfig struct` (id, name, description, timeout, on_timeout, final, allowed_roles)
    - Type `EventConfig struct` (name, from []string, to string, validators, actions)
  - Agregar tags YAML: `` `yaml:"field_name"` ``
  - _Requirements: Configuration Schema_

- [ ] 12.2 YAML parser
  - Implementar `internal/infrastructure/config/loader/yaml_loader.go`:
    - Type `YAMLWorkflowLoader struct` con validator
    - Método `LoadFromFile(path string) (*workflow.Workflow, error)`:
      - 1. os.ReadFile(path)
      - 2. yaml.Unmarshal(data, &config)
      - 3. Validar config con JSON schema
      - 4. buildWorkflow(config) → domain.Workflow
    - Método `buildWorkflow(config) (*workflow.Workflow, error)`:
      - Convertir StateConfig → domain.State
      - Convertir EventConfig → domain.Event
      - Construir workflow con NewWorkflow()
      - AddState() y AddEvent() para cada elemento
  - _Requirements: YAML Parsing, Configuration Validation_

- [ ] 12.3 Workflow validator
  - Implementar `internal/infrastructure/config/loader/validator.go`:
    - Type `WorkflowValidator struct`
    - Método `Validate(config *WorkflowConfig) error`:
      - Verificar initial_state existe en states
      - Verificar events referencian estados válidos
      - Validar timeouts son duraciones válidas
      - Verificar no hay ciclos infinitos obligatorios
      - Validar nombres únicos
  - _Requirements: Schema Validation_

- [ ] 12.4 Workflow templates
  - Crear `internal/infrastructure/config/templates/radicacion.yaml`:
    - Definir 6 estados completos (ver design.md sección 3.3.6)
    - Eventos con validadores y actions
    - Timers y escalamientos
    - Webhooks configurados
  - Crear `simple_approval.yaml` como ejemplo simple
  - _Requirements: Workflow Templates_

- [ ]* 12.5 Tests de workflow loader
  - Crear `internal/infrastructure/config/loader/yaml_loader_test.go`:
    - `TestYAMLLoader_LoadValid_Success`
    - `TestYAMLLoader_LoadInvalid_Error`
    - `TestYAMLLoader_BuildWorkflow_CompleteRadicacion`
    - Usar fixtures de prueba en `test/fixtures/workflows/`
  - _Requirements: Unit Testing, Fixtures_

### Fase 13: Infrastructure - HTTP REST API

- [ ] 13.1 Gin server setup
  - Implementar `internal/infrastructure/http/rest/server.go`:
    - Type `Server struct` con engine *workflow.Engine, router *gin.Engine, config ServerConfig
    - Type `ServerConfig struct` (Port, EnableCORS, AuthEnabled, RateLimit, MetricsEnabled)
    - Constructor `NewServer(engine, config) *Server`
    - Método `setupMiddlewares()`:
      - gin.Logger()
      - gin.Recovery()
      - CORS middleware (si enabled)
      - Auth middleware (si enabled)
      - Rate limit middleware
    - Método `setupRoutes()` (delegará a handlers)
    - Método `Run() error` (start HTTP server)
    - Método `Shutdown(ctx) error` (graceful shutdown)
  - _Requirements: HTTP Server, Graceful Shutdown_

- [ ] 13.2 Middlewares
  - Implementar `internal/infrastructure/http/rest/middleware/logging.go`:
    - Middleware que logea request/response con structured logging
    - Log: method, path, status, duration, user_id
  - Implementar `internal/infrastructure/http/rest/middleware/cors.go`:
    - CORS headers configurables
  - Implementar `internal/infrastructure/http/rest/middleware/recovery.go`:
    - Panic recovery con stack trace
    - Retornar 500 Internal Server Error
  - Implementar `internal/infrastructure/http/rest/middleware/auth.go`:
    - JWT token validation
    - Extraer user_id y roles del token
    - Set en context
  - Implementar `internal/infrastructure/http/rest/middleware/metrics.go`:
    - Incrementar counters de requests
    - Observar latency
  - _Requirements: Middlewares, Logging, Security_

- [ ] 13.3 Router y rutas
  - Implementar `internal/infrastructure/http/rest/router.go`:
    - Función `SetupRoutes(router *gin.Engine, handlers Handlers)`:
      - API v1 group: `/api/v1`
      - Workflows: GET, POST, PUT, DELETE `/workflows`, `/workflows/:id`
      - Instances: POST `/instances`, GET `/instances`, GET `/instances/:id`
      - Events: POST `/instances/:id/events`
      - History: GET `/instances/:id/history`
      - Lifecycle: POST `/instances/:id/pause|resume`, DELETE `/instances/:id`
      - Queries: POST `/queries/instances`, GET `/queries/statistics`
      - Webhooks: GET, POST, DELETE `/webhooks`
      - Health: GET `/health`
      - Metrics: GET `/metrics` (Prometheus format)
  - _Requirements: REST API Routes_

- [ ] 13.4 Instance handler
  - Implementar `internal/infrastructure/http/rest/handlers/instance_handler.go`:
    - Type `InstanceHandler struct` con use cases inyectados
    - `CreateInstance(c *gin.Context)`:
      - Bind JSON request → CreateInstanceRequest
      - Validar request
      - Ejecutar createUseCase.Execute()
      - Retornar 201 Created con InstanceResponse
    - `TriggerEvent(c *gin.Context)`:
      - Parse instanceID de path param
      - Bind JSON request → TriggerEventRequest
      - Ejecutar triggerUseCase.Execute()
      - Retornar 200 OK con TransitionResponse
    - `GetInstance(c *gin.Context)`
    - `GetHistory(c *gin.Context)`
    - `PauseInstance(c *gin.Context)`
    - `ResumeInstance(c *gin.Context)`
    - `CancelInstance(c *gin.Context)`
    - Método privado `handleError(c, err)`:
      - Mapear domain errors → HTTP status codes
      - ErrNotFound → 404
      - ErrInvalidTransition → 409
      - ErrVersionConflict → 409
      - Default → 500
  - _Requirements: HTTP Handlers, Error Handling_

- [ ] 13.5 Workflow handler
  - Implementar `internal/infrastructure/http/rest/handlers/workflow_handler.go`:
    - Similar a InstanceHandler
    - `CreateWorkflow(c)` (upload YAML/JSON)
    - `GetWorkflow(c)`
    - `ListWorkflows(c)`
    - `UpdateWorkflow(c)`
    - `DeleteWorkflow(c)`
    - `VisualizeWorkflow(c)` (retornar mermaid diagram)
  - _Requirements: HTTP Handlers, File Upload_

- [ ] 13.6 Query handler
  - Implementar `internal/infrastructure/http/rest/handlers/query_handler.go`:
    - `QueryInstances(c)`:
      - Bind QueryRequest
      - Ejecutar queryUseCase
      - Retornar paginado con total
    - `GetStatistics(c)` (estados, counts, duraciones)
    - `GetActorWorkload(c)` (tareas por actor)
  - _Requirements: Query Handlers, Statistics_

- [ ] 13.7 Health y metrics handlers
  - Implementar `internal/infrastructure/http/rest/handlers/health_handler.go`:
    - `HealthCheck(c)`:
      - Verificar DB connection (ping)
      - Verificar Redis connection
      - Retornar status: "healthy" / "unhealthy"
      - Incluir detalles de dependencias
  - Implementar `internal/infrastructure/http/rest/handlers/metrics_handler.go`:
    - `PrometheusMetrics(c)`:
      - Usar promhttp.Handler()
      - Exponer métricas en formato Prometheus
  - _Requirements: Health Checks, Observability_

- [ ] 13.8 Request/Response DTOs
  - Crear `internal/infrastructure/http/rest/presenter.go`:
    - Funciones para convertir domain → response DTOs
    - `toInstanceResponse(inst *instance.Instance) InstanceResponse`
    - `toWorkflowResponse(wf *workflow.Workflow) WorkflowResponse`
    - `toTransitionResponse(result) TransitionResponse`
  - _Requirements: DTOs, Presentation Layer_

- [ ]* 13.9 Tests de HTTP handlers
  - Crear `internal/infrastructure/http/rest/handlers/instance_handler_test.go`:
    - Setup con gin test mode y httptest
    - Mock de use cases
    - `TestInstanceHandler_CreateInstance_Success`
    - `TestInstanceHandler_TriggerEvent_Success`
    - `TestInstanceHandler_TriggerEvent_InvalidTransition_Returns409`
    - `TestInstanceHandler_GetInstance_NotFound_Returns404`
    - Verificar status codes, response bodies, headers
  - _Requirements: HTTP Testing, Mocking_

### Fase 14: Infrastructure - Dependency Injection

- [ ] 14.1 DI Container
  - Implementar `internal/infrastructure/di/container.go`:
    - Type `Container struct` con todos los componentes:
      - db *sql.DB
      - redisClient *redis.Client
      - repositories (instance, workflow)
      - use cases (create, trigger, query)
      - handlers (instance, workflow, query)
      - eventBus, locker, logger
    - Constructor `NewContainer(config *Config) (*Container, error)`:
      - 1. initInfrastructure() → DB, Redis connections
      - 2. initRepositories() → PostgresRepo, RedisCache, HybridRepo
      - 3. initEventBus() → MultiDispatcher (WebhookDispatcher, LogDispatcher)
      - 4. initUseCases() → inyectar dependencias
      - 5. initHandlers() → inyectar use cases
    - Método `Close() error` (cleanup de connections)
    - Getters para exponer handlers
  - _Requirements: Dependency Injection, Factory Pattern_

- [ ] 14.2 Configuration loader
  - Implementar `internal/infrastructure/di/config.go`:
    - Type `Config struct` (ver design.md sección 9.2)
    - Función `LoadConfig() (*Config, error)`:
      - Usar viper para cargar desde:
        - config.yaml
        - Variables de entorno (precedencia)
      - Validar configuración completa
  - _Requirements: Configuration Management_

- [ ]* 14.3 Tests del container
  - Crear `internal/infrastructure/di/container_test.go`:
    - `TestNewContainer_ValidConfig_Success`
    - `TestContainer_WiringIsCorrect` (verificar dependencies no nil)
    - Usar config de prueba
  - _Requirements: Integration Testing_

### Fase 15: Infrastructure - Observability

- [ ] 15.1 Structured logger
  - Implementar `internal/infrastructure/observability/logging/logger.go`:
    - Implementar `ports.Logger` interface usando zerolog
    - Type `ZerologLogger struct` con logger zerolog.Logger
    - Métodos: Debug, Info, Warn, Error con key-value pairs
    - Método `With(keysAndValues)` para contexto
    - Configurar output (console/json), level, timestamp
  - _Requirements: Structured Logging, Ports Implementation_

- [ ] 15.2 Prometheus metrics
  - Implementar `internal/infrastructure/observability/metrics/prometheus.go`:
    - Definir métricas globales:
      - `transitionDuration` (Histogram por workflow_id, event)
      - `lockWaitDuration` (Histogram)
      - `cacheHitRate` (Counter por result=hit/miss)
      - `instancesTotal` (Gauge por status)
      - `httpRequestDuration` (Histogram por method, path, status)
      - `httpRequestsTotal` (Counter)
      - `dbConnectionsOpen` (Gauge)
    - Función `init()` para registrar métricas
    - Helpers para instrumentar código
  - _Requirements: Prometheus Metrics, Observability_

- [ ] 15.3 OpenTelemetry tracing
  - Implementar `internal/infrastructure/observability/tracing/opentelemetry.go`:
    - Función `InitTracer(serviceName, endpoint) (*sdktrace.TracerProvider, error)`
    - Configurar Jaeger exporter
    - Configurar sampler (AlwaysSample en dev, probabilistic en prod)
    - Helpers para crear spans
  - Instrumentar use cases con spans
  - _Requirements: Distributed Tracing_

- [ ]* 15.4 Tests de observability
  - Verificar logs se generan correctamente
  - Verificar métricas se incrementan
  - Mock de tracer
  - _Requirements: Observability Testing_

### Fase 16: Application - Subprocesos

- [ ] 16.1 Subprocess domain logic
  - Extender `internal/domain/instance/instance.go`:
    - Agregar método `SpawnSubprocess(workflowID, data) (*Instance, error)`:
      - Crear nueva instancia con parentID = self.ID
      - Copiar context/variables relevantes
      - Retornar subproceso
    - Agregar método `WaitForSubprocess(subprocessID) error`
  - Extender repository para queries de subprocesos
  - _Requirements: Subprocess Support, Parent-Child Relationship_

- [ ] 16.2 SpawnSubprocessUseCase
  - Implementar `internal/application/subprocess/spawn_subprocess.go`:
    - Type `SpawnSubprocessCommand` (ParentInstanceID, WorkflowID, Data, WaitForCompletion, Timeout)
    - Ejecutar:
      - 1. Cargar parent instance
      - 2. Crear subprocess instance con parent_id
      - 3. Guardar subprocess
      - 4. Si WaitForCompletion, polling o wait channel
      - 5. Merge resultados
  - _Requirements: Subprocess Use Case_

- [ ]* 16.3 Tests de subprocesos
  - `TestInstance_SpawnSubprocess_Success`
  - `TestSpawnSubprocessUseCase_WithWait`
  - `TestSpawnSubprocessUseCase_Timeout`
  - _Requirements: Subprocess Testing_

### Fase 17: Application - Actores y Roles

- [ ] 17.1 Actor domain model
  - Crear `internal/domain/actor/actor.go`:
    - Type `Actor struct` (id, name, roles []Role)
    - Type `Role` enum (Radicador, Asignador, Gestionador, Revisor, Aprobador)
    - Type `Permission` value object
  - Crear `internal/domain/actor/service.go`:
    - Domain service `ValidatePermission(actor, requiredRole) error`
  - _Requirements: DDD Domain Services, RBAC_

- [ ] 17.2 Actor management use cases
  - Implementar `internal/application/actor/assign_actor.go`
  - Implementar `internal/application/actor/reassign_actor.go`
  - Validar permisos antes de transiciones
  - _Requirements: Actor Management_

- [ ]* 17.3 Tests de actores
  - Test de validación de permisos
  - Test de asignación/reasignación
  - _Requirements: Actor Testing_

### Fase 18: Infrastructure - Timers y Scheduler

- [ ] 18.1 Timer domain model
  - Crear `internal/domain/timer/timer.go`:
    - Type `Timer struct` (id, instanceID, state, eventOnTimeout, expiresAt)
    - Constructor `NewTimer(instanceID, state, timeout, eventOnTimeout)`
  - Crear `internal/domain/timer/scheduler.go`:
    - Interface `Scheduler` port
  - _Requirements: Timer Domain Model_

- [ ] 18.2 Timer repository
  - Implementar PostgreSQL repository para timers
  - Queries: FindExpired, FindByInstance
  - _Requirements: Timer Persistence_

- [ ] 18.3 Timer scheduler adapter
  - Implementar `internal/infrastructure/scheduler/timer_scheduler.go`:
    - Background worker que:
      - 1. Cada X segundos query timers expirados
      - 2. Para cada timer expirado, trigger evento timeout
      - 3. Marcar timer como fired
    - Worker pool con goroutines
  - _Requirements: Background Workers, Scheduling_

- [ ]* 18.4 Tests de timers
  - Test de creación y expiración
  - Test de scheduler dispara eventos
  - _Requirements: Timer Testing_

### Fase 19: Domain Layer - Escalamientos Manuales (R18)

- [ ] 19.1 Escalation aggregate
  - Crear `internal/domain/escalation/escalation.go`:
    - Type `Escalation struct` con:
      - id, instanceID, departmentID
      - reason, escalatedBy, escalatedAt
      - status (EscalationStatus enum)
      - response, respondedBy, respondedAt
      - closedBy, closedAt
      - domainEvents []event.DomainEvent
    - Constructor `NewEscalation(instanceID, departmentID, reason, escalatedBy) (*Escalation, error)`
    - Método `Reply(response, respondedBy string) error`:
      - Validar status == pending
      - Cambiar status a responded
      - Guardar response, respondedBy, respondedAt
      - Generar evento EscalationReplied
    - Método `Close(closedBy string) error`:
      - Validar status == responded
      - Cambiar status a closed
      - Guardar closedBy, closedAt
      - Generar evento EscalationClosed
    - Método `Cancel() error`:
      - Validar status == pending
      - Cambiar status a canceled
      - Generar evento EscalationCanceled
    - Métodos `Status()`, `CanCancel() bool`, `DomainEvents()`
  - _Requirements: R18, DDD Aggregates, Domain Events_

- [ ] 19.2 Escalation value objects y errors
  - Crear `internal/domain/escalation/status.go`:
    - Type `EscalationStatus` enum: Pending, Responded, Closed, Canceled
    - Métodos `IsActive() bool`, `IsFinal() bool`
  - Crear `internal/domain/escalation/department_id.go`:
    - Type `DepartmentID struct` con validación
    - Constructor `NewDepartmentID(value string) (DepartmentID, error)`
  - Crear `internal/domain/escalation/errors.go`:
    - `ErrEscalationNotFound`, `ErrInvalidEscalationStatus`
    - `ErrCannotReply`, `ErrCannotClose`, `ErrCannotCancel`
  - _Requirements: R18, DDD Value Objects_

- [ ] 19.3 Escalation repository port
  - Crear `internal/domain/escalation/repository.go`:
    - Interface `Repository` con métodos:
      - `Save(ctx, *Escalation) error`
      - `FindByID(ctx, ID) (*Escalation, error)`
      - `FindByInstance(ctx, instanceID) ([]*Escalation, error)`
      - `FindByDepartment(ctx, departmentID string, status EscalationStatus) ([]*Escalation, error)`
      - `FindPending(ctx) ([]*Escalation, error)`
  - _Requirements: R18, Repository Pattern_

- [ ] 19.4 Escalation repository adapter (PostgreSQL)
  - Implementar `internal/infrastructure/persistence/postgres/escalation_repository.go`:
    - Type `EscalationRepository struct` con db *sql.DB
    - Implementar todos los métodos de la interface
    - Mapeo domain ↔ DB model
    - Queries optimizados con índices
  - _Requirements: R18, Repository Adapter_

- [ ] 19.5 Escalation use cases
  - Implementar `internal/application/escalation/escalate_instance.go`:
    - Type `EscalateInstanceCommand` (InstanceID, DepartmentID, Reason, EscalatedBy)
    - Ejecutar:
      - 1. Cargar instance, validar existe y está activa
      - 2. Crear escalation con NewEscalation()
      - 3. Guardar en escalationRepo
      - 4. Publicar evento DocumentEscalated
      - 5. Retornar EscalationID
  - Implementar `internal/application/escalation/reply_escalation.go`:
    - Type `ReplyEscalationCommand` (EscalationID, Response, RespondedBy)
    - Ejecutar:
      - 1. Cargar escalation
      - 2. Llamar escalation.Reply()
      - 3. Guardar cambios
      - 4. Publicar eventos
  - Implementar `internal/application/escalation/close_escalation.go`:
    - Similar a Reply
  - Implementar `internal/application/escalation/query_escalations.go`:
    - Queries con filtros: por instancia, departamento, status
  - _Requirements: R18, Use Case Pattern_

- [ ] 19.6 Escalation HTTP handlers
  - Implementar `internal/infrastructure/http/rest/handlers/escalation_handler.go`:
    - `EscalateInstance(c)` → POST /instances/:id/escalations
    - `ReplyEscalation(c)` → POST /escalations/:id/reply
    - `CloseEscalation(c)` → POST /escalations/:id/close
    - `CancelEscalation(c)` → DELETE /escalations/:id
    - `GetEscalation(c)` → GET /escalations/:id
    - `GetInstanceEscalations(c)` → GET /instances/:id/escalations
    - `GetDepartmentEscalations(c)` → GET /departments/:id/escalations
    - Validar permisos según rol
    - Mapear errors → HTTP status codes
  - _Requirements: R18, HTTP Handlers_

- [ ]* 19.7 Tests de escalamientos
  - Test `internal/domain/escalation/escalation_test.go`:
    - `TestNewEscalation_Success`
    - `TestEscalation_Reply_Success`
    - `TestEscalation_Reply_InvalidStatus_Error`
    - `TestEscalation_Close_Success`
    - `TestEscalation_Cancel_OnlyWhenPending`
  - Test de use cases con mocks
  - Test de integration con PostgreSQL
  - Coverage objetivo: >85%
  - _Requirements: R18, Unit Testing, Integration Testing_

### Fase 20: Domain Layer - Reclasificación de Instancias (R19)

- [ ] 20.1 Reclassification domain logic
  - Extender `internal/domain/instance/instance.go` (ya implementado en Fase 4.3):
    - Método `Reclassify(newType, reason, actor string) error`
  - Crear `internal/domain/instance/reclassification.go`:
    - Type `ReclassificationType struct` para validar tipos permitidos
    - Función `ValidateReclassification(fromType, toType string) error`
  - Crear evento `internal/domain/event/document_reclassified.go`:
    - Type `DocumentReclassified struct` implementando DomainEvent
    - Campos: instanceID, fromType, toType, reason, actor, occurredAt
  - _Requirements: R19, Domain Logic_

- [ ] 20.2 Reclassification use case
  - Implementar `internal/application/instance/reclassify_instance.go`:
    - Type `ReclassifyInstanceCommand` (InstanceID, NewType, Reason, Actor)
    - Type `ReclassifyInstanceResult` (InstanceID, OldType, NewType, Version)
    - Ejecutar:
      - 1. Validar command (campos requeridos)
      - 2. Adquirir distributed lock sobre instancia
      - 3. Cargar instance
      - 4. Validar permisos del actor (requires senior role via guard)
      - 5. Validar tipos old → new son diferentes
      - 6. Llamar instance.Reclassify()
      - 7. Guardar con instanceRepo.Save()
      - 8. Publicar evento DocumentReclassified
      - 9. Liberar lock
      - 10. Retornar resultado
  - _Requirements: R19, Use Case Pattern_

- [ ] 20.3 Reclassification HTTP handler
  - Implementar endpoint en `internal/infrastructure/http/rest/handlers/instance_handler.go`:
    - `ReclassifyInstance(c)` → POST /instances/:id/reclassify
    - Request body: `{new_type, reason}`
    - Validar permisos (solo senior users)
    - Ejecutar use case
    - Retornar 200 OK con resultado
    - Manejo de errores: 403 si no tiene permisos, 409 si lock/conflict
  - Agregar ruta en router
  - _Requirements: R19, HTTP Handlers, Authorization_

- [ ] 20.4 Reclassification workflow configuration
  - Extender `internal/infrastructure/config/loader/types.go`:
    - Agregar campo `Reclassification *ReclassificationConfig` a WorkflowDefinition
    - Type `ReclassificationConfig struct` con:
      - Enabled bool
      - AllowedTypes []string
      - RequiresSenior bool
  - Actualizar YAML loader para parsear configuración
  - Validar configuración al cargar workflow
  - _Requirements: R19, Configuration_

- [ ]* 20.5 Tests de reclasificación
  - Test `TestInstance_Reclassify_Success`
  - Test `TestInstance_Reclassify_SameType_Error`
  - Test `TestReclassifyUseCase_Execute_Success`
  - Test `TestReclassifyUseCase_Execute_NoPermissions_Error`
  - Test del handler HTTP con validación de permisos
  - Coverage objetivo: >85%
  - _Requirements: R19, Unit Testing_

### Fase 21: Domain Layer - Guards de Transición Avanzados (R20)

- [ ] 21.1 Guard interface y registry
  - Crear `internal/domain/workflow/guard.go`:
    - Interface `Guard` con método:
      - `Validate(ctx context.Context, instance *instance.Instance, actor *actor.Actor) error`
    - Interface `GuardRegistry` con métodos:
      - `Register(name string, guard Guard) error`
      - `Get(name string) (Guard, error)`
      - `Has(name string) bool`
    - Type `GuardRegistry struct` (singleton global)
    - Función `RegisterGuard(name, guard)` y `GetGuard(name)`
  - _Requirements: R20, Strategy Pattern_

- [ ] 21.2 Guards pre-definidos - Roles
  - Implementar `internal/domain/workflow/guards/role_guards.go`:
    - `HasRoleGuard` struct:
      - Config: role string
      - Validate: verificar actor.HasRole(role)
    - `HasAnyRoleGuard` struct:
      - Config: roles []string
      - Validate: verificar actor.HasAnyRole(roles)
    - Registrar en init(): "has_role", "has_any_role"
  - _Requirements: R20, Guard Implementation_

- [ ] 21.3 Guards pre-definidos - Asignación
  - Implementar `internal/domain/workflow/guards/assignment_guards.go`:
    - `IsAssignedToActorGuard`: Validar instance.CurrentActor == actor.ID
    - `IsNotAssignedGuard`: Validar instance.CurrentActor == nil
    - Registrar: "is_assigned_to_actor", "is_not_assigned"
  - _Requirements: R20, Guard Implementation_

- [ ] 21.4 Guards pre-definidos - Campos
  - Implementar `internal/domain/workflow/guards/field_guards.go`:
    - `FieldEqualsGuard`: data[field] == value
    - `FieldNotEmptyGuard`: data[field] != nil && != ""
    - `FieldExistsGuard`: data[field] existe
    - `FieldMatchesGuard`: regex.Match(data[field])
    - Registrar: "field_equals", "field_not_empty", "field_exists", "field_matches"
  - _Requirements: R20, Guard Implementation_

- [ ] 21.5 Guards pre-definidos - Tiempo
  - Implementar `internal/domain/workflow/guards/time_guards.go`:
    - `InstanceAgeLessThanGuard`: now - instance.CreatedAt < duration
    - `InstanceAgeMoreThanGuard`: now - instance.CreatedAt > duration
    - `BeforeTimeGuard`: now < time
    - `AfterTimeGuard`: now > time
    - `OnWeekdayGuard`: now.Weekday() in [Mon..Fri]
    - Registrar: "instance_age_less_than", "instance_age_more_than", "before_time", "after_time", "on_weekday"
  - _Requirements: R20, Guard Implementation_

- [ ] 21.6 Guards pre-definidos - Estado y Datos
  - Implementar `internal/domain/workflow/guards/state_guards.go`:
    - `SubStateEqualsGuard`: currentSubState == value (R17)
    - `ParentStateEqualsGuard`: parent.CurrentState == value
  - Implementar `internal/domain/workflow/guards/data_guards.go`:
    - `DataSizeLessThanGuard`: len(data) < maxSize
    - `HasAttachmentsGuard`: data["attachments"] != nil && len > 0
    - Registrar todos
  - _Requirements: R20, Guard Implementation_

- [ ] 21.7 Guard evaluation con lógica OR/AND
  - Extender `internal/domain/workflow/workflow.go`:
    - Método `CanTransition(from State, event Event, instance, actor) (bool, error)`:
      - Evaluar lista de guards del evento
      - AND: Todos deben pasar (short-circuit en primer fallo)
      - OR: Detectar sintaxis `or:` en YAML, al menos uno debe pasar
      - Retornar error detallado indicando qué guard falló
  - Implementar `internal/domain/workflow/guard_evaluator.go`:
    - Type `GuardEvaluator struct`
    - Método `EvaluateAll(guards []GuardConfig, instance, actor) error`
    - Soporte para anidación hasta 3 niveles
  - _Requirements: R20, Complex Logic Evaluation_

- [ ] 21.8 Guard configuration en YAML
  - Extender `internal/infrastructure/config/loader/types.go`:
    - Type `GuardConfig struct` con:
      - Name string
      - Config map[string]interface{}
      - OR []GuardConfig (para lógica OR)
    - Agregar campo `Guards []GuardConfig` a EventConfig
  - Actualizar YAML loader para parsear guards
  - Validar que todos los guards existan en registry al cargar
  - _Requirements: R20, Configuration_

- [ ]* 21.9 Tests de guards
  - Test de cada guard pre-definido (15+ tests)
  - Test de GuardEvaluator con lógica AND
  - Test de GuardEvaluator con lógica OR
  - Test de anidación de guards
  - Test de workflow.CanTransition con guards
  - Test de configuración YAML con guards
  - Coverage objetivo: >90%
  - _Requirements: R20, Unit Testing_

### Fase 22: Application - Plantillas de Workflows (R21)

- [ ] 22.1 Template domain logic
  - Extender `internal/domain/workflow/workflow.go`:
    - Agregar campo `isTemplate bool`
    - Agregar campo `templateID *ID` (referencia al template origen)
    - Método `MarkAsTemplate() error`
    - Método `IsTemplate() bool`
    - Validar: No se pueden crear instancias directamente de templates
  - _Requirements: R21, Domain Logic_

- [ ] 22.2 Workflow cloning logic
  - Implementar `internal/domain/workflow/cloner.go`:
    - Type `WorkflowCloner struct`
    - Método `Clone(template *Workflow, overrides map[string]interface{}) (*Workflow, error)`:
      - 1. Validar template.IsTemplate() == true
      - 2. Deep copy de toda la configuración
      - 3. Aplicar overrides con dot notation (ej: "states.filed.timeout": "48h")
      - 4. Establecer templateID = template.ID
      - 5. Validar workflow resultante
      - 6. Retornar nuevo workflow
    - Función helper `applyOverride(config, path string, value interface{}) error`
      - Parsear dot notation: "states.filed.timeout" → ["states", "filed", "timeout"]
      - Navegar estructura anidada
      - Aplicar valor en la ruta correcta
  - _Requirements: R21, Deep Copy, Dot Notation_

- [ ] 22.3 Template use cases
  - Implementar `internal/application/workflow/clone_from_template.go`:
    - Type `CloneWorkflowCommand` (TemplateID, NewID, Name, Description, Overrides map)
    - Type `CloneWorkflowResult` (WorkflowID, TemplateID, AppliedOverrides)
    - Ejecutar:
      - 1. Cargar template desde workflowRepo
      - 2. Validar que isTemplate == true
      - 3. Usar WorkflowCloner.Clone()
      - 4. Aplicar overrides
      - 5. Generar nuevo ID si NewID == nil
      - 6. Actualizar name, description
      - 7. Validar workflow completo
      - 8. Guardar en workflowRepo
      - 9. Generar evento WorkflowClonedFromTemplate
      - 10. Retornar resultado
  - Implementar `internal/application/workflow/mark_as_template.go`:
    - Cargar workflow, llamar MarkAsTemplate(), guardar
  - _Requirements: R21, Use Case Pattern_

- [ ] 22.4 Template HTTP handlers
  - Implementar endpoints en `internal/infrastructure/http/rest/handlers/workflow_handler.go`:
    - `CloneFromTemplate(c)` → POST /workflows/from-template
      - Request: `{template_id, name, description, overrides: {...}}`
      - Ejecutar use case
      - Retornar 201 Created con WorkflowResponse
    - `MarkAsTemplate(c)` → POST /workflows/:id/mark-template
      - Solo admin
      - Retornar 200 OK
    - Extender `ListWorkflows(c)` → GET /workflows?is_template=true
      - Filtrar por isTemplate
    - `GetTemplateDerivedWorkflows(c)` → GET /workflows?template_id=xxx
      - Listar workflows derivados
  - Agregar rutas en router
  - _Requirements: R21, HTTP Handlers_

- [ ] 22.5 Template repository support
  - Extender `internal/infrastructure/persistence/postgres/workflow_repository.go`:
    - Actualizar queries para incluir is_template, template_id
    - Método `FindTemplates(ctx) ([]*Workflow, error)`
    - Método `FindByTemplateID(ctx, templateID) ([]*Workflow, error)`
    - Validar: No permitir crear instancias de templates
  - _Requirements: R21, Repository Pattern_

- [ ]* 22.6 Tests de plantillas
  - Test `TestWorkflow_MarkAsTemplate_Success`
  - Test `TestWorkflowCloner_Clone_ApplyOverrides`
  - Test `TestWorkflowCloner_DotNotation_NestedPaths`
  - Test `TestCloneFromTemplateUseCase_Success`
  - Test `TestWorkflowRepo_PreventInstancesFromTemplate`
  - Test de handlers HTTP
  - Coverage objetivo: >85%
  - _Requirements: R21, Unit Testing_

### Fase 23: Application - Import/Export de Workflows (R22)

- [ ] 23.1 Export workflow logic
  - Implementar `internal/application/workflow/export_workflow.go`:
    - Type `ExportWorkflowCommand` (WorkflowID, Format string)
    - Type `ExportWorkflowResult` (Content []byte, Format, Filename, Checksum string)
    - Ejecutar:
      - 1. Cargar workflow desde repo
      - 2. Serializar a YAML o JSON según format
      - 3. Agregar metadata:
         - schema_version: "2.0"
         - exported_at: timestamp
         - workflow: configuración completa
      - 4. Calcular checksum SHA256 del contenido
      - 5. Incluir checksum en metadata
      - 6. Retornar contenido serializado
  - _Requirements: R22, Serialization_

- [ ] 23.2 Import workflow logic
  - Implementar `internal/application/workflow/import_workflow.go`:
    - Type `ImportWorkflowCommand` (Content []byte, Format, Mode, GenerateNewID bool)
    - Type `ImportWorkflowResult` (WorkflowID, Warnings []string)
    - Modo: "create" | "update" | "force"
    - Ejecutar:
      - 1. Parsear YAML/JSON
      - 2. Validar schema_version y compatibilidad con versión actual
      - 3. Verificar checksum si presente
      - 4. Validar configuración del workflow
      - 5. Verificar si workflow con ese ID ya existe:
         - create: error si existe
         - update: actualizar existente
         - force: sobrescribir sin validar
      - 6. Si GenerateNewID, crear nuevo UUID
      - 7. Persistir workflow
      - 8. Generar warnings para configs específicas (webhooks, etc)
      - 9. Retornar resultado con warnings
  - _Requirements: R22, Deserialization, Validation_

- [ ] 23.3 Bulk import/export logic
  - Implementar `internal/application/workflow/export_bulk.go`:
    - Type `ExportBulkCommand` (WorkflowIDs []string, Format string)
    - Type `ExportBulkResult` (ZipContent []byte, Manifest map)
    - Ejecutar:
      - 1. Exportar cada workflow individualmente
      - 2. Crear ZIP archive con:
         - workflow_<id>.yaml (o .json)
         - manifest.json con metadata de todos
      - 3. Manifest incluye: lista de workflows, versiones, dependencies
      - 4. Retornar ZIP como []byte
  - Implementar `internal/application/workflow/import_bulk.go`:
    - Type `ImportBulkCommand` (ZipContent []byte, Mode string)
    - Ejecutar:
      - 1. Descomprimir ZIP
      - 2. Leer manifest.json
      - 3. Ordenar imports por dependencies
      - 4. Importar cada workflow secuencialmente
      - 5. Coleccionar errores y warnings
      - 6. Retornar reporte de imports (success, failures)
  - _Requirements: R22, Bulk Operations, ZIP_

- [ ] 23.4 Version compatibility matrix
  - Implementar `internal/application/workflow/version_compatibility.go`:
    - Type `VersionCompatibility struct`
    - Método `IsCompatible(fromVersion, toVersion string) (bool, error)`:
      - Matriz de compatibilidad:
        - 1.0 compatible con 1.x
        - 2.0 requiere migración desde 1.x
        - 2.1 compatible con 2.0
      - Retornar error con mensaje si no compatible
    - Método `Migrate(workflow, fromVersion, toVersion) (*Workflow, error)`:
      - Aplicar transformaciones necesarias
      - Actualizar schema_version
  - _Requirements: R22, Version Management_

- [ ] 23.5 Import/Export HTTP handlers
  - Implementar endpoints en `internal/infrastructure/http/rest/handlers/workflow_handler.go`:
    - `ExportWorkflow(c)` → GET /workflows/:id/export?format=yaml|json
      - Ejecutar use case
      - Set headers: Content-Type, Content-Disposition
      - Retornar archivo para descarga
    - `ImportWorkflow(c)` → POST /workflows/import?mode=create|update|force
      - Request: multipart/form-data con archivo
      - Ejecutar use case
      - Retornar 201 Created con warnings
    - `ExportBulk(c)` → POST /workflows/export-bulk
      - Request: `{workflow_ids: [], format: "yaml"}`
      - Retornar ZIP file
    - `ImportBulk(c)` → POST /workflows/import-bulk?mode=create
      - Request: ZIP file
      - Retornar reporte de imports
  - _Requirements: R22, HTTP Handlers, File Upload_

- [ ]* 23.6 Tests de import/export
  - Test `TestExportWorkflow_YAML_Success`
  - Test `TestExportWorkflow_JSON_Success`
  - Test `TestExportWorkflow_IncludesChecksum`
  - Test `TestImportWorkflow_ValidFile_Success`
  - Test `TestImportWorkflow_InvalidChecksum_Error`
  - Test `TestImportWorkflow_IncompatibleVersion_Error`
  - Test `TestVersionCompatibility_Matrix`
  - Test `TestExportBulk_CreatesZIP`
  - Test `TestImportBulk_ParsesManifest`
  - Test de handlers HTTP con file upload
  - Coverage objetivo: >85%
  - _Requirements: R22, Unit Testing_

### Fase 24: Command Line Interfaces

- [ ] 24.1 API Server entrypoint
  - Implementar `cmd/api/main.go`:
    - 1. Cargar config con LoadConfig()
    - 2. Crear DI container con NewContainer(config)
    - 3. Crear HTTP server con NewServer(container, config)
    - 4. Setup signal handling (SIGTERM, SIGINT)
    - 5. Start server
    - 6. Wait for signal
    - 7. Graceful shutdown con timeout
    - 8. container.Close()
  - _Requirements: Main Entrypoint, Graceful Shutdown_

- [ ] 24.2 Worker entrypoint (timers)
  - Implementar `cmd/worker/main.go`:
    - Similar a API server pero sin HTTP
    - Start timer scheduler
    - Start event subscriber
    - Process background tasks
  - _Requirements: Background Worker_

- [ ] 24.3 CLI tool entrypoint
  - Implementar `cmd/cli/main.go`:
    - Commands:
      - `migrate up|down` (ejecutar migraciones)
      - `seed` (cargar workflows de ejemplo)
      - `validate <workflow.yaml>` (validar workflow)
    - Usar cobra/cli library
  - _Requirements: CLI Tool_

### Fase 25: Deployment y Configuración

- [ ] 25.1 Dockerfile multi-stage
  - Crear `Dockerfile`:
    - Stage 1: Builder (go build)
    - Stage 2: Runtime (alpine + binary)
    - COPY configs
    - EXPOSE 8080 9090
    - HEALTHCHECK endpoint /health
    - CMD ["./flowengine"]
  - _Requirements: Docker, Multi-stage Build_

- [ ] 25.2 Docker Compose
  - Crear `docker-compose.yml`:
    - Services: flowengine, postgres, redis
    - Networks
    - Volumes para persistencia
    - Environment variables
    - Health checks
    - Adicional: prometheus, grafana para observability
  - _Requirements: Docker Compose, Local Development_

- [ ] 25.3 Kubernetes manifests
  - Crear `deployments/k8s/deployment.yaml`:
    - Deployment con 3 replicas
    - Resource limits (memory, CPU)
    - Liveness y readiness probes
  - Crear `deployments/k8s/service.yaml` (LoadBalancer)
  - Crear `deployments/k8s/configmap.yaml` y `secret.yaml`
  - _Requirements: Kubernetes, Production Deployment_

- [ ] 25.4 Variables de entorno
  - Crear `config/.env.example` con todas las variables
  - Documentar cada variable en README
  - _Requirements: Configuration Management_

- [ ] 25.5 Prometheus config
  - Crear `deployments/prometheus.yml`:
    - Scrape config para flowengine:9090
    - Retention, storage
  - _Requirements: Monitoring Setup_

### Fase 26: Ejemplo Completo - Flujo de Radicación

- [ ] 26.1 Workflow radicación completo
  - Verificar `internal/infrastructure/config/templates/radicacion.yaml` tiene:
    - 6 estados completos (radicar, asignar, gestionar, revisar, aprobar, enviar)
    - Todos los eventos con from/to correctos
    - Timeouts configurados
    - Validators y actions
    - Webhooks
    - Guards avanzados (R20)
    - Configuración de metadata (R23)
  - _Requirements: Workflow Configuration, R20, R23_

- [ ] 26.2 Custom actions para radicación
  - Implementar actions:
    - `GenerateIDAction` (generar numero radicado formato RAD-YYYY-NNNNNN)
    - `NotifyAction` (enviar email/notificación)
    - `SignDocumentAction` (firmar digitalmente)
    - `EmitEventAction` (publicar evento externo)
  - Registrar actions en action registry
  - _Requirements: Custom Actions_

- [ ] 26.3 Custom validators
  - Implementar validators:
    - `RequiredFieldsValidator` (verificar campos requeridos)
    - `CustomValidator` (lógica de negocio específica)
  - Registrar validators en validator registry
  - _Requirements: Custom Validators_

- [ ] 26.4 Seed script
  - Crear `scripts/seed.sh`:
    - Cargar workflow radicación a DB
    - Crear usuarios de ejemplo
    - Crear instancia de prueba
  - _Requirements: Database Seeding_

- [ ]* 26.5 E2E test flujo completo
  - Crear `test/e2e/radicacion_test.go`:
    - `TestRadicacionWorkflow_CompleteFlow_E2E`:
      - 1. Start containers (PostgreSQL, Redis)
      - 2. Start API server
      - 3. Cargar workflow radicación
      - 4. POST /instances (crear instancia)
      - 5. POST /instances/:id/events para cada transición:
         - generar_radicado
         - asignar_gestor
         - enviar_revision
         - aprobar_revision
         - aprobar_documento
      - 6. Verificar estado final = "enviar", status = "completed"
      - 7. GET /instances/:id/history (verificar 5 transiciones)
      - 8. Verificar eventos dispatched via WebhookDispatcher
      - 9. Verificar webhooks enviados
    - Casos alternos:
      - Test con rechazo (rechazar_revision)
      - Test con escalamiento (crear subproceso)
      - Test con timeout
  - Build tag `//go:build e2e`
  - _Requirements: E2E Testing, Complete Workflow_

### Fase 27: Performance Testing

- [ ] 27.1 k6 load test
  - Crear `test/performance/load_test.js`:
    - Scenarios:
      - Ramp up to 50 users (1 min)
      - Sustained 50 users (3 min)
      - Spike to 100 users (1 min)
      - Sustained 100 users (2 min)
      - Ramp down (1 min)
    - Thresholds:
      - p95 < 500ms
      - p99 < 1s
      - Error rate < 1%
    - Operations:
      - Create instance
      - Trigger event (multiple)
      - Query instances
  - _Requirements: Load Testing, Performance Benchmarks_

- [ ] 27.2 Benchmarks Go
  - Crear benchmarks:
    - `BenchmarkInstance_Transition`
    - `BenchmarkHybridRepository_FindByID`
    - `BenchmarkTriggerEventUseCase_Execute`
  - Ejecutar con `go test -bench=. -benchmem`
  - _Requirements: Benchmarking_

- [ ]* 27.3 Performance tuning
  - Ejecutar load tests
  - Identificar bottlenecks con profiling (pprof)
  - Optimizar queries, índices, cache
  - Objetivo: 10K+ transitions/second aggregate
  - _Requirements: Performance Optimization_

### Fase 28: Documentación

- [ ] 28.1 README principal
  - Crear `README.md` completo:
    - Descripción del proyecto
    - Features principales (incluir R17-R23)
    - Quick start (docker-compose up)
    - API examples (cURL)
    - Architecture diagram (mermaid)
    - Links a docs/
  - _Requirements: Documentation_

- [ ] 28.2 API documentation (OpenAPI)
  - Crear `internal/infrastructure/http/openapi/spec.yaml`:
    - OpenAPI 3.0 spec completo
    - Todos los endpoints (incluir nuevos de R18-R22)
    - Request/response schemas
    - Examples
    - Error codes
  - Integrar Swagger UI en `/docs`
  - _Requirements: API Documentation, OpenAPI_

- [ ] 28.3 Architecture Decision Records
  - Crear `docs/architecture/adr/`:
    - `001-hexagonal-architecture.md`
    - `002-redis-postgres-hybrid.md`
    - `003-optimistic-locking.md`
    - `004-event-driven-webhooks.md`
    - `005-advanced-guards.md` (R20)
    - `006-workflow-templates.md` (R21)
    - `007-metadata-validation.md` (R23)
  - Template ADR: Context, Decision, Consequences
  - _Requirements: ADR, Architecture Documentation_

- [ ] 28.4 Workflow examples documentation
  - Crear `docs/workflows/`:
    - `radicacion.md` (explicación paso a paso)
    - `custom-workflows.md` (guía para crear workflows)
    - `yaml-reference.md` (referencia completa de sintaxis con R17, R20, R23)
    - `guards-reference.md` (guía completa de guards - R20)
    - `templates-guide.md` (uso de plantillas - R21)
  - _Requirements: User Documentation_

- [ ] 28.5 Deployment guide
  - Crear `docs/deployment/`:
    - `local-development.md` (docker-compose)
    - `kubernetes.md` (k8s deployment)
    - `production-checklist.md`
    - `monitoring.md` (Prometheus + Grafana)
  - _Requirements: Operations Documentation_

### Fase 29: CI/CD

- [ ] 29.1 GitHub Actions - Tests
  - Crear `.github/workflows/test.yml`:
    - Trigger: push, pull_request
    - Jobs:
      - unit-tests (Go 1.24)
      - integration-tests (con services: postgres, redis)
      - lint (golangci-lint)
    - Upload coverage to codecov
  - _Requirements: CI, Automated Testing_

- [ ] 29.2 GitHub Actions - Build & Push
  - Crear `.github/workflows/build.yml`:
    - Trigger: push to main, tags
    - Jobs:
      - build-binary
      - docker-build-push (Docker Hub / GitHub Container Registry)
      - Tag con version desde git tag
  - _Requirements: CI, Docker Registry_

- [ ] 29.3 GitHub Actions - Security
  - Crear `.github/workflows/security.yml`:
    - Dependabot alerts
    - Trivy security scan (Docker image)
    - gosec (Go security scanner)
    - SAST checks
  - _Requirements: Security Scanning_

- [ ] 29.4 Pre-commit hooks (opcional)
  - Crear `.pre-commit-config.yaml`:
    - go fmt
    - go vet
    - golangci-lint
  - Setup con pre-commit framework
  - _Requirements: Code Quality_

### Fase 30: Final Review y Polish

- [ ]* 30.1 Code review completo
  - Revisar SOLID principles en cada layer
  - Verificar separation of concerns
  - Eliminar código duplicado
  - Verificar error handling consistente
  - Revisar implementaciones de R17-R23
  - _Requirements: Code Quality, Clean Code_

- [ ]* 30.2 Test coverage verification
  - Ejecutar `make test-coverage`
  - Verificar >80% coverage total
  - Verificar >90% coverage en domain layer
  - Verificar coverage en nuevos componentes (Escalation, Guards, Templates, Import/Export)
  - Identificar y testear edge cases faltantes
  - _Requirements: Test Coverage_

- [ ]* 30.3 Performance verification
  - Ejecutar k6 load tests
  - Verificar thresholds se cumplen:
    - p95 < 500ms
    - p99 < 1s
    - Error rate < 1%
  - Verificar cache hit rate >90%
  - Testear impacto de guards en performance
  - _Requirements: Performance Benchmarks_

- [ ]* 30.4 Security review
  - Verificar no secrets hardcodeados
  - Verificar input validation en todos los endpoints (incluir nuevos)
  - Verificar SQL injection protection (prepared statements)
  - Verificar HMAC signatures en webhooks
  - Revisar guards no introducen vulnerabilidades
  - Revisar dependency vulnerabilities
  - _Requirements: Security_

- [ ]* 30.5 Documentation review
  - Verificar README está completo y actualizado con R17-R23
  - Verificar API docs (OpenAPI) están sincronizados
  - Verificar ejemplos funcionan (incluir nuevos features)
  - Verificar documentación de guards, templates, import/export
  - Spell check
  - _Requirements: Documentation Quality_

- [ ] 30.6 Release preparation
  - Crear CHANGELOG.md con v2.0.0 (incluir R17-R23)
  - Tag release en git: `git tag v2.0.0`
  - Generar release notes detallando nuevas características
  - Publicar Docker image
  - Anuncio/comunicación
  - _Requirements: Release Management_

---

## Resumen de Fases

| Fase | Descripción | Duración Estimada |
|------|-------------|-------------------|
| 1 | Setup y Estructura | 1-2 días |
| 2-5 | Domain Layer Completo | 3-4 días |
| 6-8 | Infrastructure - Persistence | 3-4 días |
| 9 | Application - Use Cases | 2 días |
| 10-11 | Infrastructure - Messaging | 2 días |
| 12 | Infrastructure - Config Loader | 2 días |
| 13 | Infrastructure - HTTP API | 3-4 días |
| 14 | Infrastructure - DI | 1 día |
| 15 | Infrastructure - Observability | 2 días |
| 16-18 | Features Avanzados (Subprocesos, Actores, Timers) | 3-4 días |
| **19** | **Escalamientos Manuales (R18)** | **3-4 días** |
| **20** | **Reclasificación de Instancias (R19)** | **2 días** |
| **21** | **Guards de Transición Avanzados (R20)** | **3-4 días** |
| **22** | **Plantillas de Workflows (R21)** | **2-3 días** |
| **23** | **Import/Export de Workflows (R22)** | **3 días** |
| 24 | CLI Entrypoints | 1 día |
| 25 | Deployment | 2 días |
| 26 | Ejemplo Radicación | 2-3 días |
| 27 | Performance Testing | 2 días |
| 28 | Documentación | 3-4 días |
| 29 | CI/CD | 1-2 días |
| 30 | Review y Polish | 3-4 días |

**Total Estimado: 11-14 semanas** (incremento de 3-4 semanas debido a R17-R23)

### Desglose de Nuevos Requerimientos

**R17 (Subestados)**: Integrado en Fases 4, 6 (migraciones), sin fase dedicada (+1 día distribuido)
**R18 (Escalamientos)**: Fase 19 dedicada (3-4 días) - Aggregate completo, use cases, handlers, tests
**R19 (Reclasificación)**: Fase 20 dedicada (2 días) - Use case, validaciones, configuración
**R20 (Guards Avanzados)**: Fase 21 dedicada (3-4 días) - 15+ guards, registry, evaluación, tests
**R21 (Plantillas)**: Fase 22 dedicada (2-3 días) - Clonación, overrides, dot notation
**R22 (Import/Export)**: Fase 23 dedicada (3 días) - Serialización, versiones, bulk operations
**R23 (Metadata)**: Integrado en Fases 4, 6 (migraciones), sin fase dedicada (+0.5 días distribuido)

---

## Notas

- Tareas marcadas con `*` pueden ejecutarse en paralelo con otras tareas
- Cada fase debe completar sus tests antes de avanzar
- Commits frecuentes con mensajes descriptivos
- Pull requests para features completas
- Code reviews obligatorios
- Mantener coverage >80% en todo momento

---

**Última actualización**: 2025-11-10
**Versión del plan**: 2.0

## Cambios en v2.0

Esta versión del plan incorpora los siguientes nuevos requerimientos:

- **R17**: Sistema de Subestados Jerárquicos (integrado en Fases 4, 6)
- **R18**: Escalamientos Manuales a Departamentos Externos (Fase 19 nueva)
- **R19**: Reclasificación de Instancias sin Cambio de Estado (Fase 20 nueva)
- **R20**: Guards de Transición Avanzados con Lógica Compleja (Fase 21 nueva)
- **R21**: Plantillas de Workflows Reutilizables (Fase 22 nueva)
- **R22**: Import/Export de Workflows con Versionado (Fase 23 nueva)
- **R23**: Metadata Extendida en Transiciones con Validación (integrado en Fases 4, 6)

**Incremento de esfuerzo**: +3-4 semanas (de 8-10 a 11-14 semanas)
**Nuevas tablas**: 1 (workflow_escalations)
**Nuevos campos**: 9 distribuidos en 3 tablas existentes
**Nuevos endpoints**: ~20+
**Nuevos aggregates/value objects**: 3+
**Nuevos use cases**: 10+
