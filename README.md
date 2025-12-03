# FlowEngine

FlowEngine es un motor de workflows genérico, escalable y cloud-native basado en máquinas de estados finitos (FSM). Diseñado con arquitectura hexagonal (Clean Architecture) para proporcionar alta cohesión y bajo acoplamiento.

## Características Principales

- **Workflows Configurables**: Define workflows mediante YAML/JSON con estados, eventos y transiciones
- **Arquitectura Hexagonal**: Separación clara entre dominio, aplicación e infraestructura
- **Persistencia Híbrida**: Combina Redis (cache) y PostgreSQL (persistencia) para máximo rendimiento
- **Concurrencia Segura**: Optimistic locking para manejo de conflictos
- **Escalabilidad**: Soporta múltiples instancias de workflow en paralelo
- **Sistema de Eventos**: Integración con RabbitMQ para eventos externos
- **Subprocesos Jerárquicos**: Soporte para workflows anidados
- **Actores y Roles**: Sistema completo de autorización y asignación de tareas
- **Timers y Escalamientos**: Soporte para timeouts y escalamiento automático

## Stack Tecnológico

- **Go 1.21+**: Lenguaje principal
- **Gin**: Framework HTTP
- **PostgreSQL 15+**: Base de datos relacional
- **Redis 7+**: Cache y bloqueo distribuido
- **RabbitMQ**: Sistema de mensajería
- **looplab/fsm**: Librería de máquinas de estados
- **Docker & Kubernetes**: Contenedorización y orquestación

## Estructura del Proyecto

```
FlowEngine/
├── cmd/                    # Puntos de entrada (API, worker, CLI)
├── internal/
│   ├── domain/            # Lógica de negocio pura
│   │   ├── workflow/      # Aggregate Workflow
│   │   ├── instance/      # Aggregate Instance
│   │   ├── actor/         # Sistema de actores
│   │   ├── event/         # Domain events
│   │   └── shared/        # Value objects compartidos
│   ├── application/       # Casos de uso
│   └── infrastructure/    # Adaptadores externos
│       ├── persistence/   # PostgreSQL, Redis
│       ├── messaging/     # RabbitMQ
│       └── http/          # REST API
├── pkg/                   # Código público reutilizable
├── config/                # Configuraciones
├── migrations/            # Migraciones de BD
└── test/                  # Tests de integración

```

## Inicio Rápido

### Prerrequisitos Mínimos

- **Go 1.21 o superior** ✅
- (Opcional) PostgreSQL 15+, Redis 7+, RabbitMQ 3.12+ para producción

### Instalación y Ejecución

```bash
# Clonar repositorio
git clone https://github.com/LaFabric-LinkTIC/FlowEngine.git
cd FlowEngine

# Instalar dependencias
go mod download

# Ejecutar tests
make test

# 🚀 Iniciar API server (con repositorios in-memory)
make run
```

**El servidor estará disponible en `http://localhost:8080`**

### Probar el API

#### Opción 1: Con Postman (Recomendado) ⭐

```bash
# Importar la colección en Postman
# Archivos: postman/FlowEngine_API.postman_collection.json
#          postman/FlowEngine_Environment.postman_environment.json

# Ver guía completa
cat postman/README.md
```

#### Opción 2: Con cURL

```bash
# Health check
curl http://localhost:8080/health

# Crear un workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/create_workflow.json

# Ver documentación completa
cat docs/api_quickstart.md
```

## Comandos Disponibles

```bash
make build              # Compilar binario
make test               # Ejecutar tests unitarios
make test-integration   # Tests de integración
make test-coverage      # Reporte de cobertura
make lint               # Linter (golangci-lint)
make run                # Ejecutar API server
make docker-build       # Build imagen Docker
make docker-up          # Levantar stack completo
make migrate-up         # Aplicar migraciones
make migrate-down       # Revertir migraciones
```

## Documentación

- [Requisitos](requirements.md)
- [Diseño Arquitectónico](design.md)
- [Plan de Implementación](task.md)
- **[API REST - Quick Start](docs/api_quickstart.md)** ⭐ ¡NUEVO!
- [REST API - Estado Completo](REST_API_COMPLETE.md)

## Arquitectura

FlowEngine sigue los principios de **Clean Architecture** y **Domain-Driven Design (DDD)**:

- **Domain Layer**: Lógica de negocio pura, sin dependencias externas
- **Application Layer**: Casos de uso y orquestación
- **Infrastructure Layer**: Adaptadores para BD, mensajería, HTTP, etc.
- **Ports & Adapters**: Interfaces para inversión de dependencias

## Contribución

Este proyecto está en desarrollo activo. Para contribuir:

1. Fork el repositorio
2. Crea una rama feature (`git checkout -b feature/nueva-funcionalidad`)
3. Commit tus cambios (`git commit -am 'Agregar nueva funcionalidad'`)
4. Push a la rama (`git push origin feature/nueva-funcionalidad`)
5. Crea un Pull Request

## Licencia

_Por definir_

## Contacto

LaFabric-LinkTIC - [GitHub](https://github.com/LaFabric-LinkTIC)
