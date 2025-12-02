# FlowEngine - Architecture Documentation

## System Overview

FlowEngine es un motor de workflows empresarial diseñado con arquitectura hexagonal (Clean Architecture) para gestión documental con soporte específico para flujos MinTrabajo.

## High-Level Architecture

```mermaid
graph TB
    subgraph "External Clients"
        Web[Web Application]
        Mobile[Mobile App]
        External[External Systems]
    end

    subgraph "API Gateway Layer"
        GIN[Gin HTTP Server]
        MW[Middleware Stack]
    end

    subgraph "Application Layer"
        WUC[Workflow Use Cases]
        IUC[Instance Use Cases]
    end

    subgraph "Domain Layer"
        WF[Workflow Aggregate]
        INST[Instance Aggregate]
        EVT[Event System]
        TMR[Timer Entity]
    end

    subgraph "Infrastructure Layer"
        PG[(PostgreSQL)]
        RD[(Redis Cache)]
        SCH[Scheduler Worker]
    end

    Web --> GIN
    Mobile --> GIN
    External --> GIN

    GIN --> MW
    MW --> WUC
    MW --> IUC

    WUC --> WF
    IUC --> INST
    WUC --> EVT
    IUC --> EVT

    WF --> PG
    INST --> PG
    WF --> RD
    INST --> RD
    TMR --> SCH
    SCH --> IUC
```

## Component Architecture

```mermaid
graph LR
    subgraph "HTTP Layer"
        R[Router] --> WH[Workflow Handler]
        R --> IH[Instance Handler]
    end

    subgraph "Middleware"
        AUTH[JWT Auth]
        CORS[CORS]
        LOG[Logger]
        RID[Request ID]
    end

    subgraph "Use Cases"
        CW[CreateWorkflow]
        CWY[CreateFromYAML]
        GW[GetWorkflow]
        CI[CreateInstance]
        TI[TransitionInstance]
        GI[GetInstance]
        CLI[CloneInstance]
    end

    subgraph "Domain"
        WFA[Workflow]
        INSTA[Instance]
        EVTA[Events]
    end

    subgraph "Repositories"
        WR[WorkflowRepo]
        IR[InstanceRepo]
        TR[TimerRepo]
    end

    WH --> CW
    WH --> CWY
    WH --> GW
    IH --> CI
    IH --> TI
    IH --> GI
    IH --> CLI

    CW --> WFA
    CI --> INSTA
    TI --> INSTA

    WFA --> WR
    INSTA --> IR
```

## Domain Model

```mermaid
classDiagram
    class Workflow {
        +ID id
        +String name
        +String description
        +Version version
        +Map~string,State~ states
        +Map~string,Event~ events
        +Timestamp createdAt
        +Timestamp updatedAt
        +AddState(state)
        +AddEvent(event)
        +CanTransition(from, event)
        +Validate()
    }

    class State {
        +String ID
        +String Name
        +String Description
        +Duration Timeout
        +String OnTimeout
        +Bool IsFinal
        +WithTimeout(duration)
        +AsFinal()
    }

    class Event {
        +String Name
        +State[] Sources
        +State Destination
        +Validator[] Validators
        +CanTransitionFrom(state)
    }

    class Instance {
        +ID id
        +ID workflowID
        +ID parentID
        +String currentState
        +Status status
        +Version version
        +Data data
        +Variables variables
        +Transition[] history
        +Transition(toState, event, actor, metadata)
        +Complete()
        +Cancel()
        +Pause()
        +Resume()
    }

    class Transition {
        +ID id
        +String fromState
        +String toState
        +String event
        +ID actorID
        +TransitionMetadata metadata
        +Data dataSnapshot
        +Timestamp timestamp
    }

    class Status {
        <<enumeration>>
        RUNNING
        PAUSED
        COMPLETED
        CANCELED
        FAILED
    }

    Workflow "1" *-- "*" State
    Workflow "1" *-- "*" Event
    Event "*" --> "*" State : sources
    Event "*" --> "1" State : destination
    Instance "*" --> "1" Workflow
    Instance "1" *-- "*" Transition
    Instance --> Status
```

## State Machine Flow

```mermaid
stateDiagram-v2
    [*] --> RUNNING: CreateInstance

    RUNNING --> RUNNING: Transition
    RUNNING --> PAUSED: Pause
    RUNNING --> COMPLETED: Complete (Final State)
    RUNNING --> CANCELED: Cancel
    RUNNING --> FAILED: Error

    PAUSED --> RUNNING: Resume
    PAUSED --> CANCELED: Cancel

    COMPLETED --> [*]
    CANCELED --> [*]
    FAILED --> [*]
```

## Request Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant R as Router
    participant M as Middleware
    participant H as Handler
    participant UC as UseCase
    participant D as Domain
    participant DB as Repository
    participant E as EventBus

    C->>R: POST /api/v1/instances/:id/transitions
    R->>M: Process Request
    M->>M: Validate JWT
    M->>M: Add Request ID
    M->>M: Log Request
    M->>H: Forward to Handler

    H->>H: Parse & Validate JSON:API
    H->>UC: Execute Command

    UC->>DB: FindByID(instanceID)
    DB-->>UC: Instance
    UC->>DB: FindByID(workflowID)
    DB-->>UC: Workflow

    UC->>D: Validate Transition
    D-->>UC: Valid

    UC->>D: Instance.Transition()
    D->>D: Create Transition Record
    D->>D: Update State
    D->>D: Record Domain Event
    D-->>UC: Updated Instance

    UC->>DB: Save(instance)
    DB-->>UC: Success

    UC->>E: DispatchBatch(events)
    E-->>UC: Dispatched

    UC-->>H: TransitionResult
    H->>H: Format JSON:API Response
    H-->>C: 200 OK + Response
```

## Data Flow Architecture

```mermaid
flowchart TB
    subgraph "Input"
        REQ[HTTP Request]
        YAML[YAML File]
    end

    subgraph "Validation"
        JV[JSON:API Validation]
        YP[YAML Parser]
        DV[Domain Validation]
    end

    subgraph "Processing"
        CMD[Command]
        AGG[Aggregate]
        EVT[Domain Events]
    end

    subgraph "Persistence"
        PG[(PostgreSQL)]
        RD[(Redis)]
    end

    subgraph "Output"
        RES[JSON:API Response]
        EVTD[Event Dispatch]
    end

    REQ --> JV
    YAML --> YP
    JV --> CMD
    YP --> CMD
    CMD --> DV
    DV --> AGG
    AGG --> EVT
    AGG --> PG
    AGG --> RD
    EVT --> EVTD
    AGG --> RES
```

## Directory Structure

```
FlowEngine/
├── cmd/
│   ├── api/              # REST API Server
│   │   └── main.go
│   ├── emulator/         # Workflow Testing Tool
│   ├── demo/             # Demo Utilities
│   └── quick-test/       # Quick Testing
│
├── internal/
│   ├── domain/           # Business Logic (Core)
│   │   ├── workflow/     # Workflow Aggregate
│   │   ├── instance/     # Instance Aggregate
│   │   ├── event/        # Domain Events
│   │   ├── timer/        # Timer Entity
│   │   └── shared/       # Value Objects
│   │
│   ├── application/      # Use Cases
│   │   ├── workflow/     # Workflow Operations
│   │   └── instance/     # Instance Operations
│   │
│   └── infrastructure/   # External Adapters
│       ├── http/         # REST API
│       │   ├── handler/
│       │   ├── middleware/
│       │   └── router/
│       ├── persistence/
│       │   ├── postgres/
│       │   └── memory/
│       ├── cache/        # Redis
│       ├── parser/       # YAML Parser
│       ├── security/     # JWT
│       └── scheduler/    # Timer Worker
│
├── pkg/                  # Public Packages
│   ├── jsonapi/          # JSON:API Helpers
│   └── logger/           # Logging
│
├── config/
│   └── templates/        # YAML Workflow Templates
│
├── migrations/           # Database Migrations
├── scripts/              # Utility Scripts
└── docs/                 # Documentation
```

## Deployment Architecture

```mermaid
graph TB
    subgraph "Cloud Run"
        API[FlowEngine API]
    end

    subgraph "Cloud SQL"
        PG[(PostgreSQL 16)]
    end

    subgraph "Memorystore"
        RD[(Redis 7)]
    end

    subgraph "External"
        LB[Load Balancer]
        SM[Secret Manager]
    end

    LB --> API
    API --> PG
    API --> RD
    API --> SM

    style API fill:#4285F4
    style PG fill:#336791
    style RD fill:#DC382D
```

## Technology Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| HTTP Framework | Gin | High-performance REST API |
| Database | PostgreSQL 16 | Primary persistence |
| Cache | Redis 7 | Response caching & sessions |
| Authentication | JWT (HMAC-SHA256) | API security |
| Configuration | Environment Variables | 12-factor app |
| Logging | Structured JSON | Observability |
| API Format | JSON:API v1.0 | Standard response format |
| Workflow Definition | YAML | Human-readable configs |

## Key Design Patterns

1. **Hexagonal Architecture**: Clear separation between domain, application, and infrastructure
2. **Domain-Driven Design**: Aggregates, Value Objects, Domain Events
3. **Repository Pattern**: Data access abstraction
4. **Command/Query Separation**: Use cases follow CQRS principles
5. **Event Sourcing Ready**: Domain events for state changes
6. **Optimistic Locking**: Version-based concurrency control

## Security Considerations

- JWT Bearer token authentication
- CORS configuration
- Request ID tracking for audit
- No sensitive data in logs
- Secret Manager integration for production
- SQL injection prevention via parameterized queries

## Scalability Features

- Stateless API design (Cloud Run compatible)
- Connection pooling (PostgreSQL)
- Redis caching layer
- Horizontal scaling support
- Graceful shutdown handling
