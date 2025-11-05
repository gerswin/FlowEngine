# FlowEngine - Documento de Requerimientos Técnicos

## Introduction

FlowEngine es un sistema de orquestación de workflows basado en máquina de estados finitos (FSM) que permite gestionar procesos de negocio complejos con múltiples estados, transiciones, actores y reglas. El sistema proporciona persistencia híbrida con Redis y PostgreSQL para alto rendimiento, soporta ejecución paralela de múltiples instancias de workflows, subprocesos jerárquicos, configuración de workflows mediante YAML/JSON, API REST completa, sistema de eventos externos, gestión de actores con roles y permisos, timers con escalamientos automáticos, y una arquitectura hexagonal que asegura código limpio, testeable y mantenible. El sistema está diseñado para alcanzar más de 10,000 transiciones por segundo con latencias sub-segundo, optimistic locking para control de concurrencia, y observabilidad completa mediante logging estructurado, métricas Prometheus y tracing distribuido.

---

## Glossary

### Términos de Dominio

**Workflow**
Definición de un proceso de negocio que especifica los estados posibles, las transiciones permitidas entre estados, y las reglas de validación. Un workflow es una plantilla reutilizable que puede instanciarse múltiples veces.

**Instance (Instancia)**
Ejecución concreta de un workflow. Cada instancia mantiene su propio estado actual, historial de transiciones, datos asociados, y variables de contexto.

**State (Estado)**
Punto específico en el ciclo de vida de una instancia de workflow donde se encuentra en un momento dado. Un estado puede tener configuraciones como timeouts, roles permitidos, y acciones automáticas.

**Event (Evento)**
Acción o trigger que causa una transición de estado. Los eventos pueden ser invocados por actores externos o generados automáticamente por el sistema (ej: timeouts).

**Transition (Transición)**
Cambio de un estado a otro dentro de una instancia de workflow. Cada transición queda registrada en el historial con timestamp, actor que la ejecutó, y datos asociados.

**Actor (Actor)**
Usuario o sistema externo que interactúa con las instancias de workflows ejecutando transiciones. Cada actor tiene roles asignados que determinan qué acciones puede realizar.

**Role (Rol)**
Conjunto de permisos que determina qué transiciones puede ejecutar un actor en determinados estados del workflow.

**Domain Event (Evento de Dominio)**
Notificación generada cuando ocurre algo significativo en el sistema (ej: instancia creada, estado cambiado, instancia completada). Los eventos de dominio se publican a sistemas externos mediante message queues o webhooks.

**Subprocess (Subproceso)**
Instancia de workflow creada como hijo de otra instancia (padre). Los subprocesos heredan contexto del padre y pueden sincronizarse con él.

**Timer**
Mecanismo que dispara un evento automáticamente después de un período de tiempo. Los timers se usan para escalamientos, recordatorios, y SLAs.

**Optimistic Locking**
Estrategia de control de concurrencia que usa versionado para detectar y prevenir conflictos cuando múltiples procesos intentan modificar la misma instancia simultáneamente.

**Aggregate**
Cluster de objetos de dominio tratados como una unidad para propósitos de consistencia de datos. Workflow e Instance son aggregates raíz en el sistema.

**Repository**
Abstracción que encapsula la lógica de persistencia y recuperación de aggregates, permitiendo que el dominio permanezca independiente de la infraestructura.

**Use Case**
Operación de la capa de aplicación que orquesta la ejecución de lógica de dominio, coordinando repositorios, servicios externos, y eventos.

**Hybrid Repository**
Estrategia de persistencia que combina Redis (cache rápido) con PostgreSQL (almacenamiento durable) para optimizar rendimiento y confiabilidad.

**Webhook**
Mecanismo de notificación HTTP que envía eventos del sistema a URLs externas configuradas, permitiendo integraciones en tiempo real.

**Specification Pattern**
Patrón de diseño usado para encapsular criterios de búsqueda complejos de forma reutilizable y componible.

**Clone (Clonación)**
Mecanismo que permite distribuir una instancia de workflow entre múltiples usuarios para obtener respuestas o información parcial que será consolidada. Una clonación es implementada como un tipo especial de subproceso con estados simplificados y gestión de tiempos específica.

**Cloned Instance (Instancia Clonada)**
Subproceso creado a partir de una instancia principal (padre) que permite a un usuario específico trabajar en una parte del proceso y proporcionar una respuesta que será consolidada. Tiene estados simplificados: pending, accepted, responded, rejected.

**Consolidator (Consolidador)**
Actor designado responsable de recibir y consolidar todas las respuestas de las instancias clonadas para continuar con el proceso principal. Puede ser el mismo actor que inició la clonación o uno diferente.

**Clone Time Restriction (Restricción de Tiempo de Clonación)**
Regla de negocio que limita el tiempo asignado a las clonaciones para asegurar que el proceso principal tenga margen suficiente para consolidación y cierre.

---

## Requirements

### R1: Gestión de Workflows

#### R1.1: Crear Workflow desde Configuración YAML

**User Story**
Como administrador del sistema, quiero crear workflows desde archivos YAML, para definir procesos de negocio sin necesidad de escribir código.

**Acceptance Criteria**

1. **WHEN** el administrador carga un archivo YAML con la definición del workflow
   **THEN** el sistema SHALL validar la sintaxis del archivo
   **AND** el sistema SHALL verificar que exista un estado inicial definido
   **AND** el sistema SHALL verificar que todos los eventos referencien estados existentes
   **AND** el sistema SHALL almacenar el workflow en la base de datos
   **AND** el sistema SHALL asignar un ID único al workflow

2. **WHEN** el archivo YAML contiene errores de sintaxis
   **THEN** el sistema SHALL retornar un error descriptivo indicando la línea y tipo de error
   **AND** el sistema SHALL NOT crear el workflow

3. **WHEN** el archivo YAML define estados con nombres duplicados
   **THEN** el sistema SHALL retornar un error "Duplicate state ID"
   **AND** el sistema SHALL NOT crear el workflow

4. **IF** el archivo YAML incluye configuración de timers para estados
   **THEN** el sistema SHALL validar que los timeout sean duraciones válidas (ej: "2h", "30m")
   **AND** el sistema SHALL validar que el evento de timeout exista en el workflow

5. **IF** el archivo YAML define validators o actions personalizadas
   **THEN** el sistema SHALL verificar que estén registradas en el action registry
   **AND** el sistema SHALL retornar error si alguna acción no existe

#### R1.2: Listar y Consultar Workflows

**User Story**
Como usuario del sistema, quiero listar todos los workflows disponibles y consultar sus detalles, para conocer qué procesos puedo iniciar.

**Acceptance Criteria**

1. **WHEN** el usuario solicita la lista de workflows mediante GET /api/v1/workflows
   **THEN** el sistema SHALL retornar un array de workflows
   **AND** cada workflow SHALL incluir: id, name, description, version, created_at
   **AND** la respuesta SHALL estar en formato JSON

2. **WHEN** el usuario solicita detalles de un workflow específico mediante GET /api/v1/workflows/:id
   **THEN** el sistema SHALL retornar la configuración completa del workflow
   **AND** la respuesta SHALL incluir: todos los estados, todos los eventos, configuración de webhooks, y SLA

3. **WHEN** el usuario solicita un workflow que no existe
   **THEN** el sistema SHALL retornar HTTP 404 Not Found
   **AND** el mensaje de error SHALL indicar "Workflow not found"

4. **IF** el usuario solicita visualización del workflow
   **THEN** el sistema SHALL retornar un diagrama Mermaid representando estados y transiciones

#### R1.3: Actualizar y Eliminar Workflows

**User Story**
Como administrador del sistema, quiero actualizar o eliminar workflows existentes, para mantener las definiciones de procesos actualizadas.

**Acceptance Criteria**

1. **WHEN** el administrador actualiza un workflow mediante PUT /api/v1/workflows/:id
   **THEN** el sistema SHALL validar la nueva configuración con las mismas reglas de creación
   **AND** el sistema SHALL incrementar el campo version del workflow
   **AND** el sistema SHALL almacenar los cambios

2. **IF** existen instancias activas del workflow siendo actualizado
   **THEN** el sistema SHALL retornar una advertencia
   **AND** las instancias activas SHALL continuar usando la versión anterior del workflow
   **AND** las nuevas instancias SHALL usar la versión actualizada

3. **WHEN** el administrador elimina un workflow mediante DELETE /api/v1/workflows/:id
   **THEN** el sistema SHALL realizar soft delete (marcar deleted_at)
   **AND** el workflow SHALL NOT aparecer en listados
   **AND** las instancias existentes SHALL continuar funcionando

4. **IF** se intenta eliminar un workflow con instancias en estado Running
   **THEN** el sistema SHALL retornar HTTP 409 Conflict
   **AND** el mensaje SHALL indicar el número de instancias activas

---

### R2: Gestión de Instancias de Workflow

#### R2.1: Crear Instancia de Workflow

**User Story**
Como usuario autorizado, quiero crear una nueva instancia de un workflow, para iniciar un proceso de negocio específico.

**Acceptance Criteria**

1. **WHEN** el usuario crea una instancia mediante POST /api/v1/instances
   **THEN** el sistema SHALL validar que el workflow_id exista
   **AND** el sistema SHALL validar que el actor_id y actor_role sean válidos
   **AND** el sistema SHALL crear una nueva instancia con ID único (UUID)
   **AND** el sistema SHALL establecer current_state al estado inicial del workflow
   **AND** el sistema SHALL establecer status = "running"
   **AND** el sistema SHALL establecer version = 1
   **AND** el sistema SHALL almacenar data proporcionado en formato JSONB

2. **WHEN** la instancia es creada exitosamente
   **THEN** el sistema SHALL generar un evento de dominio "InstanceCreated"
   **AND** el sistema SHALL publicar el evento al message queue
   **AND** el sistema SHALL retornar HTTP 201 Created
   **AND** la respuesta SHALL incluir: instance_id, workflow_id, current_state, version, created_at

3. **WHEN** el workflow especificado no existe
   **THEN** el sistema SHALL retornar HTTP 404 Not Found
   **AND** el mensaje SHALL ser "Workflow not found"

4. **IF** el workflow define campos requeridos en el estado inicial
   **THEN** el sistema SHALL validar que data incluya esos campos
   **AND** el sistema SHALL retornar HTTP 400 Bad Request si faltan campos

5. **IF** se proporciona un parent_id en el request
   **THEN** el sistema SHALL crear la instancia como subproceso
   **AND** el sistema SHALL validar que la instancia padre exista
   **AND** el sistema SHALL establecer el parent_id en la nueva instancia

#### R2.2: Consultar Instancias

**User Story**
Como usuario del sistema, quiero consultar instancias de workflows con filtros múltiples, para monitorear el estado de los procesos.

**Acceptance Criteria**

1. **WHEN** el usuario consulta instancias mediante GET /api/v1/instances
   **THEN** el sistema SHALL retornar un array de instancias
   **AND** cada instancia SHALL incluir: id, workflow_id, current_state, status, version, created_at, updated_at

2. **IF** el usuario proporciona query parameters (workflow_id, states, actors, status, from_date, to_date)
   **THEN** el sistema SHALL aplicar todos los filtros especificados
   **AND** el sistema SHALL usar índices de base de datos para optimizar performance

3. **WHEN** el usuario especifica limit y offset
   **THEN** el sistema SHALL retornar resultados paginados
   **AND** el sistema SHALL incluir en la respuesta: total count, limit, offset
   **AND** el limit máximo SHALL ser 100

4. **WHEN** el usuario consulta una instancia específica mediante GET /api/v1/instances/:id
   **THEN** el sistema SHALL retornar todos los detalles de la instancia
   **AND** la respuesta SHALL incluir: data completo, variables, current_actor, previous_state

5. **WHEN** el usuario solicita una instancia que no existe
   **THEN** el sistema SHALL retornar HTTP 404 Not Found

#### R2.3: Consultar Historial de Transiciones

**User Story**
Como auditor del sistema, quiero ver el historial completo de transiciones de una instancia, para auditar y rastrear el flujo del proceso.

**Acceptance Criteria**

1. **WHEN** el usuario solicita el historial mediante GET /api/v1/instances/:id/history
   **THEN** el sistema SHALL retornar todas las transiciones de la instancia
   **AND** las transiciones SHALL estar ordenadas por created_at DESC (más reciente primero)

2. **WHEN** se retorna cada transición
   **THEN** cada registro SHALL incluir: id, event, from_state, to_state, actor, actor_role, data, duration_ms, created_at

3. **IF** la instancia no tiene transiciones aún
   **THEN** el sistema SHALL retornar un array vacío
   **AND** el HTTP status SHALL ser 200 OK

4. **IF** el usuario especifica paginación (limit, offset)
   **THEN** el sistema SHALL retornar transiciones paginadas
   **AND** SHALL incluir total count en la respuesta

---

### R3: Ejecución de Transiciones

#### R3.1: Ejecutar Transición de Estado

**User Story**
Como actor autorizado, quiero ejecutar transiciones de estado en instancias de workflow, para avanzar el proceso según las reglas de negocio.

**Acceptance Criteria**

1. **WHEN** el actor ejecuta una transición mediante POST /api/v1/instances/:id/events
   **THEN** el sistema SHALL validar que la instancia exista
   **AND** el sistema SHALL validar que el evento exista en el workflow
   **AND** el sistema SHALL validar que la transición sea válida desde el estado actual
   **AND** el sistema SHALL validar que el actor tenga permisos para ejecutar el evento

2. **WHEN** todas las validaciones pasan
   **THEN** el sistema SHALL adquirir un distributed lock sobre la instancia (TTL: 30 segundos)
   **AND** el sistema SHALL cargar la instancia con su versión actual
   **AND** el sistema SHALL cambiar current_state al estado destino
   **AND** el sistema SHALL guardar previous_state
   **AND** el sistema SHALL incrementar version en 1
   **AND** el sistema SHALL agregar la transición al historial
   **AND** el sistema SHALL actualizar updated_at

3. **WHEN** la transición se ejecuta exitosamente
   **THEN** el sistema SHALL persistir los cambios usando optimistic locking
   **AND** el sistema SHALL generar evento de dominio "StateChanged"
   **AND** el sistema SHALL publicar el evento al message queue
   **AND** el sistema SHALL liberar el distributed lock
   **AND** el sistema SHALL retornar HTTP 200 OK
   **AND** la respuesta SHALL incluir: previous_state, current_state, version

4. **WHEN** el evento no es válido desde el estado actual
   **THEN** el sistema SHALL retornar HTTP 409 Conflict
   **AND** el mensaje SHALL ser "Invalid transition from {current_state} with event {event_name}"
   **AND** el sistema SHALL NOT modificar la instancia

5. **WHEN** ocurre un version conflict (otra transición modificó la instancia concurrentemente)
   **THEN** el sistema SHALL retornar HTTP 409 Conflict
   **AND** el mensaje SHALL ser "Version conflict, please retry"
   **AND** el sistema SHALL NOT aplicar la transición

6. **IF** el estado destino es un estado final
   **THEN** el sistema SHALL establecer status = "completed"
   **AND** el sistema SHALL establecer completed_at al timestamp actual
   **AND** el sistema SHALL generar evento "InstanceCompleted"

7. **IF** el workflow define validators para el evento
   **THEN** el sistema SHALL ejecutar todos los validators antes de aplicar la transición
   **AND** el sistema SHALL retornar HTTP 422 Unprocessable Entity si alguna validación falla

8. **IF** el workflow define actions para el evento
   **THEN** el sistema SHALL ejecutar todas las actions después de aplicar la transición
   **AND** las actions SHALL ejecutarse de forma asíncrona

9. **IF** otro proceso tiene el lock de la instancia
   **THEN** el sistema SHALL esperar hasta timeout (5 segundos)
   **AND** el sistema SHALL retornar HTTP 409 Conflict si no puede adquirir el lock
   **AND** el mensaje SHALL ser "Instance is locked by another process"

#### R3.2: Optimistic Locking y Control de Concurrencia

**User Story**
Como sistema, quiero prevenir conflictos cuando múltiples procesos intentan modificar la misma instancia simultáneamente, para mantener la consistencia de datos.

**Acceptance Criteria**

1. **WHEN** el sistema persiste una transición
   **THEN** el sistema SHALL ejecutar UPDATE con WHERE clause incluyendo version
   **AND** el sistema SHALL verificar que rows_affected = 1
   **AND** el sistema SHALL retornar ErrVersionConflict si rows_affected = 0

2. **WHEN** ocurre un version conflict
   **THEN** el sistema SHALL NOT aplicar la transición
   **AND** el sistema SHALL invalidar el cache de la instancia
   **AND** el cliente SHALL recibir error 409 para reintentar

3. **IF** se usa distributed locking
   **THEN** el lock SHALL tener TTL de 30 segundos
   **AND** el lock SHALL usar Redis SET NX
   **AND** el lock SHALL ser liberado solo por el proceso que lo adquirió (verificación con UUID)

---

### R4: Gestión del Ciclo de Vida de Instancias

#### R4.1: Pausar Instancia

**User Story**
Como usuario autorizado, quiero pausar una instancia de workflow en ejecución, para suspender temporalmente el proceso.

**Acceptance Criteria**

1. **WHEN** el usuario pausa una instancia mediante POST /api/v1/instances/:id/pause
   **THEN** el sistema SHALL validar que la instancia exista
   **AND** el sistema SHALL validar que status = "running"
   **AND** el sistema SHALL cambiar status a "paused"
   **AND** el sistema SHALL generar evento "InstancePaused"

2. **WHEN** la instancia está pausada
   **THEN** el sistema SHALL rechazar intentos de ejecutar transiciones
   **AND** el sistema SHALL retornar HTTP 409 Conflict con mensaje "Instance is paused"

3. **IF** la instancia ya está pausada
   **THEN** el sistema SHALL retornar HTTP 409 Conflict
   **AND** el mensaje SHALL ser "Instance is already paused"

4. **IF** la instancia está en estado final (completed, canceled, failed)
   **THEN** el sistema SHALL retornar HTTP 409 Conflict
   **AND** el mensaje SHALL ser "Cannot pause completed instance"

#### R4.2: Reanudar Instancia

**User Story**
Como usuario autorizado, quiero reanudar una instancia pausada, para continuar la ejecución del proceso.

**Acceptance Criteria**

1. **WHEN** el usuario reanuda una instancia mediante POST /api/v1/instances/:id/resume
   **THEN** el sistema SHALL validar que status = "paused"
   **AND** el sistema SHALL cambiar status a "running"
   **AND** el sistema SHALL generar evento "InstanceResumed"

2. **IF** la instancia no está pausada
   **THEN** el sistema SHALL retornar HTTP 409 Conflict
   **AND** el mensaje SHALL ser "Instance is not paused"

#### R4.3: Cancelar Instancia

**User Story**
Como usuario autorizado, quiero cancelar una instancia de workflow, para terminar el proceso de forma anormal.

**Acceptance Criteria**

1. **WHEN** el usuario cancela una instancia mediante DELETE /api/v1/instances/:id
   **THEN** el sistema SHALL validar que la instancia no esté en estado final
   **AND** el sistema SHALL cambiar status a "canceled"
   **AND** el sistema SHALL establecer completed_at
   **AND** el sistema SHALL generar evento "InstanceCanceled"

2. **WHEN** la instancia es cancelada
   **THEN** el sistema SHALL cancelar todos los timers asociados
   **AND** el sistema SHALL cancelar todos los subprocesos activos (si existen)

3. **IF** la instancia ya está en estado final
   **THEN** el sistema SHALL retornar HTTP 409 Conflict

---

### R5: Timers y Escalamientos

#### R5.1: Configurar Timers en Estados

**User Story**
Como diseñador de workflows, quiero configurar timeouts automáticos en estados, para implementar SLAs y escalamientos.

**Acceptance Criteria**

1. **WHEN** un estado define un timeout en el archivo YAML del workflow
   **THEN** el sistema SHALL validar que timeout sea una duración válida (ej: "2h", "30m", "24h")
   **AND** el sistema SHALL validar que on_timeout referencie un evento válido del workflow

2. **WHEN** una instancia entra a un estado con timeout configurado
   **THEN** el sistema SHALL crear un timer en la tabla workflow_timers
   **AND** el timer SHALL incluir: instance_id, state, event_on_timeout, expires_at
   **AND** expires_at SHALL ser calculated como current_time + timeout

3. **WHEN** una instancia sale de un estado con timer activo
   **THEN** el sistema SHALL marcar el timer como cancelado
   **AND** el timer SHALL NOT disparar el evento

#### R5.2: Procesamiento de Timers Expirados

**User Story**
Como sistema, quiero procesar automáticamente timers expirados, para disparar eventos de timeout en las instancias correspondientes.

**Acceptance Criteria**

1. **WHEN** el timer scheduler ejecuta su ciclo (cada 10 segundos)
   **THEN** el scheduler SHALL consultar todos los timers donde expires_at <= NOW() AND fired_at IS NULL
   **AND** el scheduler SHALL procesar cada timer expirado

2. **WHEN** se procesa un timer expirado
   **THEN** el sistema SHALL cargar la instancia asociada
   **AND** el sistema SHALL validar que la instancia siga en el estado del timer
   **AND** el sistema SHALL ejecutar el evento configurado en event_on_timeout
   **AND** el sistema SHALL marcar fired_at = NOW()

3. **IF** la instancia ya no está en el estado del timer
   **THEN** el sistema SHALL marcar el timer como cancelado
   **AND** el sistema SHALL NOT disparar el evento

4. **IF** el evento de timeout falla
   **THEN** el sistema SHALL registrar el error en logs
   **AND** el sistema SHALL reintentar según retry policy (max 3 intentos)

5. **WHEN** el scheduler procesa timers
   **THEN** el procesamiento SHALL usar worker pool de goroutines (10 workers)
   **AND** cada timer SHALL procesarse de forma independiente
   **AND** un error en un timer SHALL NOT afectar el procesamiento de otros

---

### R6: Subprocesos Jerárquicos

#### R6.1: Crear Subproceso

**User Story**
Como diseñador de workflows, quiero crear subprocesos (workflows hijos) desde una instancia padre, para modelar procesos complejos con delegación.

**Acceptance Criteria**

1. **WHEN** se crea una instancia con parent_id especificado
   **THEN** el sistema SHALL validar que la instancia padre exista
   **AND** el sistema SHALL establecer parent_id en la nueva instancia
   **AND** el sistema SHALL permitir que el subproceso acceda a variables del padre (read-only)

2. **WHEN** un subproceso se completa
   **THEN** el sistema SHALL generar evento "SubprocessCompleted"
   **AND** el sistema SHALL incluir subprocess_id y parent_id en el evento

3. **IF** se especifica wait_for_completion = true
   **THEN** la operación de creación SHALL esperar hasta que el subproceso complete
   **AND** el sistema SHALL retornar los resultados del subproceso

4. **IF** se especifica timeout para el wait
   **THEN** el sistema SHALL retornar error si el subproceso no completa en el tiempo especificado
   **AND** el subproceso SHALL continuar ejecutándose en background

#### R6.2: Consultar Subprocesos

**User Story**
Como usuario, quiero consultar los subprocesos de una instancia padre, para monitorear procesos delegados.

**Acceptance Criteria**

1. **WHEN** el usuario consulta subprocesos mediante GET /api/v1/instances/:id/subprocesses
   **THEN** el sistema SHALL retornar todas las instancias donde parent_id = :id
   **AND** cada subproceso SHALL incluir: id, workflow_id, current_state, status, created_at

2. **WHEN** se consulta una instancia
   **THEN** la respuesta SHALL incluir un campo parent_id si es un subproceso
   **AND** la respuesta SHALL incluir un contador de subprocesos activos

---

### R7: Actores y Control de Acceso

#### R7.1: Validación de Roles en Transiciones

**User Story**
Como sistema, quiero validar que los actores tengan los roles necesarios antes de ejecutar transiciones, para garantizar control de acceso basado en roles.

**Acceptance Criteria**

1. **WHEN** un estado define allowed_roles en el workflow
   **THEN** el sistema SHALL validar que el actor tenga uno de los roles permitidos
   **AND** el sistema SHALL retornar HTTP 403 Forbidden si el actor no tiene el rol requerido

2. **WHEN** se ejecuta una transición
   **THEN** el sistema SHALL registrar actor y actor_role en la tabla workflow_transitions
   **AND** el sistema SHALL actualizar current_actor y current_role en la instancia

3. **IF** no se especifican allowed_roles para un estado
   **THEN** el sistema SHALL permitir que cualquier actor autenticado ejecute transiciones

#### R7.2: Asignación de Actores

**User Story**
Como gestor de procesos, quiero asignar actores específicos a instancias de workflow, para controlar quién puede trabajar en cada tarea.

**Acceptance Criteria**

1. **WHEN** se asigna un actor a una instancia
   **THEN** el sistema SHALL actualizar current_actor con el actor_id
   **AND** el sistema SHALL validar que el actor exista en el sistema

2. **WHEN** se reasigna un actor
   **THEN** el sistema SHALL generar evento "ActorReassigned"
   **AND** el evento SHALL incluir previous_actor y new_actor

---

### R8: Sistema de Eventos y Webhooks

#### R8.1: Publicación de Eventos de Dominio

**User Story**
Como sistema integrador, quiero recibir eventos cuando ocurran cambios significativos en workflows, para mantener sincronizados sistemas externos.

**Acceptance Criteria**

1. **WHEN** ocurre un cambio significativo en el sistema (instancia creada, estado cambiado, instancia completada)
   **THEN** el sistema SHALL generar un evento de dominio
   **AND** el evento SHALL incluir: type, aggregate_id, occurred_at, payload

2. **WHEN** se genera un evento de dominio
   **THEN** el sistema SHALL publicar el evento a RabbitMQ exchange
   **AND** el routing key SHALL ser el event type (ej: "instance.created", "state.changed")
   **AND** el mensaje SHALL ser persistent (delivery_mode = 2)
   **AND** el content_type SHALL ser "application/json"

3. **IF** la publicación a RabbitMQ falla
   **THEN** el sistema SHALL reintentar hasta 3 veces con exponential backoff
   **AND** el sistema SHALL registrar el error en logs
   **AND** la transición principal SHALL completarse exitosamente (eventos son async)

#### R8.2: Webhooks HTTP

**User Story**
Como administrador de integraciones, quiero configurar webhooks HTTP para workflows, para enviar notificaciones a sistemas externos vía HTTP.

**Acceptance Criteria**

1. **WHEN** se configura un webhook para un workflow
   **THEN** el sistema SHALL validar que la URL sea válida (https://)
   **AND** el sistema SHALL validar que events sea un array no vacío
   **AND** el sistema SHALL almacenar secret para generar HMAC signatures

2. **WHEN** ocurre un evento que coincide con un webhook configurado
   **THEN** el sistema SHALL enviar HTTP POST request al webhook URL
   **AND** el request body SHALL ser el evento serializado en JSON
   **AND** el request SHALL incluir headers:
   - Content-Type: application/json
   - X-FlowEngine-Signature: sha256={hmac_signature}
   - X-FlowEngine-Event: {event_type}
   - Custom headers configurados en el webhook

3. **WHEN** el webhook responde con status 2xx
   **THEN** el sistema SHALL considerar el envío exitoso
   **AND** el sistema SHALL registrar el success en logs

4. **WHEN** el webhook falla (status 4xx, 5xx, timeout, connection error)
   **THEN** el sistema SHALL reintentar según retry_config
   **AND** el sistema SHALL usar exponential backoff (2s, 4s, 8s)
   **AND** después de max_retries, el sistema SHALL marcar como failed

5. **IF** se especifica active = false en el webhook
   **THEN** el sistema SHALL NOT enviar notificaciones a ese webhook

6. **WHEN** se calcula la HMAC signature
   **THEN** el sistema SHALL usar HMAC-SHA256
   **AND** el secret SHALL ser el configurado en el webhook
   **AND** el input del HMAC SHALL ser el request body completo

---

### R9: Queries y Reportes

#### R9.1: Queries Avanzadas con Filtros

**User Story**
Como analista de procesos, quiero ejecutar queries complejas sobre instancias de workflows, para generar reportes y análisis.

**Acceptance Criteria**

1. **WHEN** se ejecuta una query mediante POST /api/v1/queries/instances
   **THEN** el sistema SHALL aceptar filtros: workflow_id, states, actors, status, from_date, to_date
   **AND** el sistema SHALL aplicar todos los filtros especificados usando AND lógico
   **AND** el sistema SHALL soportar múltiples valores en states y actors (OR lógico dentro del campo)

2. **WHEN** se ejecuta la query
   **THEN** el sistema SHALL usar índices compuestos para optimizar performance
   **AND** el sistema SHALL retornar resultados paginados (limit, offset)
   **AND** la respuesta SHALL incluir total count de registros que cumplen los filtros

3. **IF** se especifica limit > 100
   **THEN** el sistema SHALL usar limit = 100 (máximo)

4. **IF** no se especifican filtros
   **THEN** el sistema SHALL retornar todas las instancias (con paginación)

#### R9.2: Estadísticas y Métricas

**User Story**
Como gerente de operaciones, quiero ver estadísticas agregadas de workflows, para monitorear el desempeño del sistema.

**Acceptance Criteria**

1. **WHEN** se solicitan estadísticas mediante GET /api/v1/queries/statistics
   **THEN** el sistema SHALL retornar:
   - Total de instancias por workflow_id
   - Total de instancias por status (running, completed, paused, canceled, failed)
   - Duración promedio de completación por workflow
   - Transiciones por estado
   - Instancias creadas en últimas 24h, 7d, 30d

2. **IF** se especifica workflow_id como query parameter
   **THEN** el sistema SHALL filtrar estadísticas solo para ese workflow

3. **WHEN** se solicita workload de un actor mediante GET /api/v1/queries/actors/:id/workload
   **THEN** el sistema SHALL retornar:
   - Total de instancias asignadas al actor
   - Instancias por estado
   - Tareas pendientes (en estado activo)

---

### R10: Persistencia y Cache

#### R10.1: Hybrid Repository (Redis + PostgreSQL)

**User Story**
Como sistema, quiero usar cache Redis para lecturas frecuentes y PostgreSQL para persistencia durable, para optimizar performance manteniendo confiabilidad.

**Acceptance Criteria**

1. **WHEN** se consulta una instancia por ID
   **THEN** el sistema SHALL buscar primero en Redis cache
   **AND** el sistema SHALL retornar inmediatamente si hay cache hit
   **AND** el sistema SHALL buscar en PostgreSQL si hay cache miss
   **AND** el sistema SHALL popular el cache con el resultado de PostgreSQL (read-through)

2. **WHEN** se guarda una instancia
   **THEN** el sistema SHALL escribir a Redis cache primero
   **AND** el sistema SHALL escribir a PostgreSQL de forma sincrónica
   **AND** el sistema SHALL invalidar cache si la escritura a PostgreSQL falla

3. **WHEN** se establece un valor en Redis cache
   **THEN** el sistema SHALL configurar TTL de 5 minutos
   **AND** el cache key format SHALL ser "instance:{uuid}"

4. **IF** se habilita async_db_writes en configuración
   **THEN** el sistema SHALL escribir a PostgreSQL en goroutine background
   **AND** el sistema SHALL retornar inmediatamente después de escribir a cache

5. **WHEN** se mide cache performance
   **THEN** el sistema SHALL lograr >90% cache hit rate para lecturas de instancias activas
   **AND** el sistema SHALL exponer métrica "flowengine_cache_requests_total{result=hit|miss}"

#### R10.2: Connection Pooling

**User Story**
Como sistema, quiero gestionar eficientemente conexiones a base de datos, para manejar alta concurrencia sin degradación de performance.

**Acceptance Criteria**

1. **WHEN** se inicializa la conexión a PostgreSQL
   **THEN** el sistema SHALL configurar MaxOpenConns = 25
   **AND** el sistema SHALL configurar MaxIdleConns = 5
   **AND** el sistema SHALL configurar ConnMaxLifetime = 5 minutos
   **AND** el sistema SHALL configurar ConnMaxIdleTime = 1 minuto

2. **WHEN** se usa Redis
   **THEN** el sistema SHALL configurar connection pool de go-redis
   **AND** el sistema SHALL configurar MaxRetries = 3

---

### R11: Observabilidad y Monitoreo

#### R11.1: Métricas Prometheus

**User Story**
Como ingeniero de SRE, quiero exponer métricas del sistema en formato Prometheus, para monitorear performance y salud del sistema.

**Acceptance Criteria**

1. **WHEN** se solicitan métricas mediante GET /metrics
   **THEN** el sistema SHALL retornar métricas en formato Prometheus text exposition
   **AND** el sistema SHALL exponer las siguientes métricas:

   - `flowengine_transition_duration_seconds` (Histogram) - labels: workflow_id, event
   - `flowengine_lock_wait_duration_seconds` (Histogram)
   - `flowengine_cache_requests_total` (Counter) - labels: result=hit|miss
   - `flowengine_instances_total` (Gauge) - labels: status
   - `flowengine_http_requests_total` (Counter) - labels: method, path, status
   - `flowengine_http_request_duration_seconds` (Histogram) - labels: method, path
   - `flowengine_db_connections` (Gauge) - labels: state=open|idle

2. **WHEN** se ejecuta una transición
   **THEN** el sistema SHALL observar la duración en flowengine_transition_duration_seconds
   **AND** el sistema SHALL incluir labels workflow_id y event_name

3. **WHEN** se accede al cache
   **THEN** el sistema SHALL incrementar flowengine_cache_requests_total con label apropiado (hit/miss)

#### R11.2: Structured Logging

**User Story**
Como desarrollador, quiero logs estructurados en formato JSON, para facilitar análisis y troubleshooting.

**Acceptance Criteria**

1. **WHEN** el sistema genera logs
   **THEN** el formato SHALL ser JSON structured logging
   **AND** cada log entry SHALL incluir: timestamp, level, message, fields adicionales

2. **WHEN** se ejecuta un use case
   **THEN** el sistema SHALL logar: use_case, instance_id, actor, event, duration_ms
   **AND** el log level SHALL ser INFO para operaciones exitosas
   **AND** el log level SHALL ser ERROR para fallos

3. **IF** ocurre un error
   **THEN** el log SHALL incluir stack trace
   **AND** el log SHALL incluir error details y context

#### R11.3: Health Check

**User Story**
Como plataforma de orquestación (Kubernetes), quiero verificar la salud del servicio, para implementar liveness y readiness probes.

**Acceptance Criteria**

1. **WHEN** se solicita health check mediante GET /api/v1/health
   **THEN** el sistema SHALL verificar conectividad con PostgreSQL (ping)
   **AND** el sistema SHALL verificar conectividad con Redis (ping)
   **AND** el sistema SHALL verificar conectividad con RabbitMQ (si está configurado)

2. **WHEN** todas las dependencias están saludables
   **THEN** el sistema SHALL retornar HTTP 200 OK
   **AND** el response body SHALL ser: {"status": "healthy", "dependencies": {...}}

3. **WHEN** alguna dependencia falla
   **THEN** el sistema SHALL retornar HTTP 503 Service Unavailable
   **AND** el response body SHALL indicar qué dependencia falló
   **AND** el status SHALL ser "unhealthy"

---

### R12: Seguridad

#### R12.1: Autenticación JWT

**User Story**
Como administrador de seguridad, quiero que todos los endpoints de API requieran autenticación JWT, para proteger el acceso al sistema.

**Acceptance Criteria**

1. **WHEN** se hace un request a un endpoint protegido
   **THEN** el sistema SHALL validar el header Authorization: Bearer {token}
   **AND** el sistema SHALL validar la firma del JWT con secret configurado
   **AND** el sistema SHALL validar que el token no esté expirado

2. **WHEN** el token es inválido o falta
   **THEN** el sistema SHALL retornar HTTP 401 Unauthorized
   **AND** el mensaje SHALL ser "Missing or invalid authorization token"

3. **WHEN** el token es válido
   **THEN** el sistema SHALL extraer user_id y roles del token
   **AND** el sistema SHALL inyectar user_id en el context de la request

4. **IF** el endpoint GET /health está configurado como público
   **THEN** el sistema SHALL NOT requerir autenticación para ese endpoint

#### R12.2: Rate Limiting

**User Story**
Como administrador del sistema, quiero limitar la tasa de requests por cliente, para prevenir abuso y garantizar disponibilidad.

**Acceptance Criteria**

1. **WHEN** se configura rate_limit_rps en configuración
   **THEN** el sistema SHALL aplicar rate limiting de requests por segundo
   **AND** el sistema SHALL usar algoritmo token bucket

2. **WHEN** un cliente excede el rate limit
   **THEN** el sistema SHALL retornar HTTP 429 Too Many Requests
   **AND** el sistema SHALL incluir header Retry-After indicando segundos de espera

3. **IF** se configura rate limiting por IP
   **THEN** el sistema SHALL agrupar requests por IP address
   **AND** el sistema SHALL mantener contadores en Redis

#### R12.3: Input Validation

**User Story**
Como sistema, quiero validar todos los inputs de usuarios, para prevenir inyección de código y garantizar integridad de datos.

**Acceptance Criteria**

1. **WHEN** se recibe un request con body JSON
   **THEN** el sistema SHALL validar usando struct tags (binding:"required", etc)
   **AND** el sistema SHALL retornar HTTP 400 Bad Request si la validación falla
   **AND** el mensaje SHALL especificar qué campos son inválidos

2. **WHEN** se reciben campos de tipo string
   **THEN** el sistema SHALL sanitizar caracteres especiales
   **AND** el sistema SHALL validar longitud máxima

3. **WHEN** se usan inputs en queries SQL
   **THEN** el sistema SHALL usar exclusivamente prepared statements
   **AND** el sistema SHALL NEVER construir queries con string concatenation

---

### R13: Deployment y Operaciones

#### R13.1: Graceful Shutdown

**User Story**
Como ingeniero de plataforma, quiero que el servicio haga shutdown gracefully, para no perder requests en flight durante deployments.

**Acceptance Criteria**

1. **WHEN** el proceso recibe señal SIGTERM o SIGINT
   **THEN** el sistema SHALL dejar de aceptar nuevos requests
   **AND** el sistema SHALL esperar a que requests en flight completen (timeout: 30 segundos)
   **AND** el sistema SHALL cerrar conexiones de DB, Redis, RabbitMQ
   **AND** el sistema SHALL terminar con exit code 0 si shutdown exitoso

2. **IF** requests no completan en el timeout
   **THEN** el sistema SHALL forzar terminación
   **AND** el sistema SHALL logar advertencia sobre requests interrumpidos

#### R13.2: Configuración por Variables de Entorno

**User Story**
Como DevOps engineer, quiero configurar el servicio mediante variables de entorno, para facilitar deployment en diferentes ambientes.

**Acceptance Criteria**

1. **WHEN** el servicio inicia
   **THEN** el sistema SHALL leer variables de entorno con prefix FLOWENGINE_
   **AND** las variables de entorno SHALL tener precedencia sobre archivo config.yaml

2. **WHEN** una variable requerida falta
   **THEN** el sistema SHALL fallar el startup
   **AND** el sistema SHALL logar qué variable falta

3. **WHEN** se especifican variables de conexión a DB
   **THEN** el sistema SHALL soportar: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD

---

### R14: Performance

#### R14.1: Latencia de Transiciones

**User Story**
Como sistema de alta performance, quiero ejecutar transiciones en menos de 500ms (p95), para garantizar experiencia de usuario responsive.

**Acceptance Criteria**

1. **WHEN** se ejecutan transiciones bajo carga normal (50 req/s)
   **THEN** el percentil 95 (p95) de latencia SHALL ser < 500ms
   **AND** el percentil 99 (p99) de latencia SHALL ser < 1000ms
   **AND** la latencia mediana (p50) SHALL ser < 200ms

2. **WHEN** se mide la latencia
   **THEN** el sistema SHALL incluir: lock acquisition, DB read, transition logic, DB write, event publishing
   **AND** el sistema SHALL excluir el tiempo de ejecución de webhooks (son async)

#### R14.2: Throughput

**User Story**
Como sistema escalable, quiero soportar más de 10,000 transiciones por segundo en total agregado, para manejar alta carga.

**Acceptance Criteria**

1. **WHEN** se ejecuta load test con ramp up a 100 usuarios concurrentes
   **THEN** el sistema SHALL mantener throughput > 10,000 transiciones/segundo
   **AND** el error rate SHALL ser < 1%
   **AND** el sistema SHALL usar horizontal scaling (múltiples instancias stateless)

2. **IF** se agrega una nueva instancia del servicio
   **THEN** el throughput total SHALL aumentar proporcionalmente
   **AND** las instancias SHALL coordinarse usando distributed locks en Redis

---

### R15: Clonación de Instancias

La clonación es una característica configurable que permite distribuir una instancia de workflow entre múltiples usuarios para obtener respuestas parciales que serán consolidadas. Esta funcionalidad reutiliza el sistema de subprocesos (R6) con lógica y validaciones específicas.

#### R15.1: Crear Clonación de Instancia

**User Story**
Como gestor autorizado, quiero clonar una instancia de workflow a múltiples usuarios de diferentes oficinas, para obtener información o respuestas parciales que consolidaré posteriormente.

**Acceptance Criteria**

1. **WHEN** el gestor inicia una clonación mediante POST /api/v1/instances/:id/clone
   **THEN** el sistema SHALL validar que la instancia padre exista
   **AND** el sistema SHALL validar que el estado actual permita clonación (según configuración del workflow)
   **AND** el sistema SHALL validar que el actor tenga rol autorizado para clonar (configurable)
   **AND** el sistema SHALL validar que status = "running"

2. **WHEN** se envía el request de clonación
   **THEN** el request SHALL incluir:
   - `assignees`: array de objetos con `{user_id, office_id}` (mínimo 2 usuarios)
   - `consolidator_id`: ID del usuario que consolidará las respuestas
   - `reason`: motivo de la clonación (texto descriptivo)
   - `timeout_duration`: duración máxima para las clonaciones (ej: "24h", "3d")
   - `metadata`: datos opcionales adicionales

3. **WHEN** se valida el timeout_duration
   **THEN** el sistema SHALL calcular el tiempo restante del trámite principal
   **AND** el sistema SHALL validar que timeout_duration <= (tiempo_restante * 0.80)
   **AND** el sistema SHALL retornar HTTP 422 Unprocessable Entity si excede el límite
   **AND** el mensaje SHALL incluir advertencia: "Clone timeout cannot exceed 80% of remaining time ({calculated_max})"

4. **WHEN** se validan los assignees
   **THEN** el sistema SHALL validar que todos los user_id existan
   **AND** el sistema SHALL validar que office_id sea válido para cada usuario
   **AND** el sistema SHALL validar que la cantidad de assignees >= 2
   **AND** el sistema SHALL retornar HTTP 400 Bad Request si hay usuarios duplicados

5. **WHEN** todas las validaciones pasan
   **THEN** el sistema SHALL crear una instancia clonada (subproceso) por cada assignee
   **AND** cada instancia clonada SHALL incluir:
   - `parent_id`: ID de la instancia principal
   - `clone_type`: "cloned_instance"
   - `assigned_user_id`: ID del usuario asignado
   - `consolidator_id`: ID del consolidador
   - `current_state`: "pending"
   - `expires_at`: calculated como NOW() + timeout_duration
   - `data`: copia de datos relevantes del padre (read-write access)

6. **WHEN** se crean las instancias clonadas
   **THEN** el sistema SHALL cambiar el estado del trámite principal según configuración del workflow
   **AND** el sistema SHALL crear un registro de clonación con:
   - `clone_group_id`: UUID único para el grupo de clones
   - `parent_instance_id`: ID de la instancia principal
   - `consolidator_id`: ID del consolidador
   - `reason`: motivo de la clonación
   - `total_clones`: número total de clones creados
   - `status`: "active"
   - `created_at`, `expires_at`

7. **WHEN** la clonación se crea exitosamente
   **THEN** el sistema SHALL generar eventos de dominio:
   - "CloneGroupCreated" (con clone_group_id, parent_id, total_clones)
   - "ClonedInstanceCreated" por cada instancia clonada
   **AND** el sistema SHALL agregar registro al historial del trámite principal
   **AND** el sistema SHALL retornar HTTP 201 Created
   **AND** la respuesta SHALL incluir: clone_group_id, cloned_instances array, expires_at

8. **IF** el workflow no tiene clonación habilitada en su configuración
   **THEN** el sistema SHALL retornar HTTP 403 Forbidden
   **AND** el mensaje SHALL ser "Cloning is not enabled for this workflow"

9. **IF** el estado actual no permite clonación
   **THEN** el sistema SHALL retornar HTTP 409 Conflict
   **AND** el mensaje SHALL ser "Cloning is not allowed from state {current_state}"

#### R15.2: Aceptar o Rechazar Clonación

**User Story**
Como usuario asignado a una clonación, quiero aceptar o rechazar la solicitud, para tener control sobre mi carga de trabajo.

**Acceptance Criteria**

1. **WHEN** el usuario asignado consulta una clonación mediante GET /api/v1/clones/:id
   **THEN** el sistema SHALL retornar los detalles de la instancia clonada
   **AND** la respuesta SHALL incluir:
   - Información general del trámite principal (vista limitada según configuración)
   - `reason`: motivo de la clonación
   - `expires_at`: fecha límite para responder
   - `current_state`: estado actual (pending, accepted, responded, rejected)
   - `consolidator`: información del consolidador
   - `parent_summary`: resumen configurable del trámite padre

2. **WHEN** el usuario acepta la clonación mediante POST /api/v1/clones/:id/accept
   **THEN** el sistema SHALL validar que current_state = "pending"
   **AND** el sistema SHALL validar que el actor sea el usuario asignado
   **AND** el sistema SHALL cambiar current_state a "accepted"
   **AND** el sistema SHALL registrar accepted_at = NOW()

3. **WHEN** la clonación es aceptada
   **THEN** el sistema SHALL generar evento "ClonedInstanceAccepted"
   **AND** el sistema SHALL agregar registro al historial del trámite principal
   **AND** el sistema SHALL notificar al consolidador (según configuración de notificaciones)
   **AND** el sistema SHALL retornar HTTP 200 OK

4. **WHEN** el usuario rechaza la clonación mediante POST /api/v1/clones/:id/reject
   **THEN** el sistema SHALL validar que current_state = "pending"
   **AND** el sistema SHALL validar que el actor sea el usuario asignado
   **AND** el sistema SHALL cambiar current_state a "rejected"
   **AND** el sistema SHALL registrar rejected_at = NOW()
   **AND** el request SHALL incluir `rejection_reason` (opcional)

5. **WHEN** la clonación es rechazada
   **THEN** el sistema SHALL generar evento "ClonedInstanceRejected"
   **AND** el sistema SHALL agregar registro al historial del trámite principal
   **AND** el sistema SHALL notificar al gestor principal y al consolidador
   **AND** el evento SHALL incluir rejection_reason
   **AND** el sistema SHALL retornar HTTP 200 OK

6. **IF** el usuario intenta aceptar/rechazar una clonación que no le pertenece
   **THEN** el sistema SHALL retornar HTTP 403 Forbidden
   **AND** el mensaje SHALL ser "You are not assigned to this clone"

7. **IF** la clonación ya fue procesada (no está en estado pending)
   **THEN** el sistema SHALL retornar HTTP 409 Conflict
   **AND** el mensaje SHALL ser "Clone already {current_state}"

8. **IF** la clonación ha expirado (expires_at < NOW())
   **THEN** el sistema SHALL permitir el rechazo
   **AND** el sistema SHALL NOT permitir la aceptación
   **AND** el sistema SHALL retornar HTTP 410 Gone si se intenta aceptar

#### R15.3: Responder Clonación

**User Story**
Como usuario que aceptó una clonación, quiero diligenciar y enviar mi respuesta, para contribuir al proceso de consolidación.

**Acceptance Criteria**

1. **WHEN** el usuario envía su respuesta mediante POST /api/v1/clones/:id/respond
   **THEN** el sistema SHALL validar que current_state = "accepted"
   **AND** el sistema SHALL validar que el actor sea el usuario asignado
   **AND** el sistema SHALL validar que NOW() <= expires_at
   **AND** el request SHALL incluir `response_data` (JSONB con la respuesta)

2. **WHEN** se valida response_data
   **THEN** el sistema SHALL validar que contenga los campos requeridos según configuración del workflow
   **AND** el sistema SHALL retornar HTTP 422 Unprocessable Entity si faltan campos obligatorios

3. **WHEN** la respuesta es válida
   **THEN** el sistema SHALL cambiar current_state a "responded"
   **AND** el sistema SHALL almacenar response_data en la instancia clonada
   **AND** el sistema SHALL registrar responded_at = NOW()
   **AND** el sistema SHALL establecer status = "completed"

4. **WHEN** la respuesta se registra exitosamente
   **THEN** el sistema SHALL generar evento "ClonedInstanceResponded"
   **AND** el sistema SHALL agregar registro al historial del trámite principal
   **AND** el sistema SHALL notificar al consolidador
   **AND** el evento SHALL incluir un resumen de la respuesta
   **AND** el sistema SHALL retornar HTTP 200 OK

5. **IF** el usuario actualiza su respuesta antes de la consolidación
   **THEN** el sistema SHALL permitir POST /api/v1/clones/:id/respond nuevamente
   **AND** el sistema SHALL actualizar response_data
   **AND** el sistema SHALL registrar updated_at
   **AND** el sistema SHALL generar evento "ClonedInstanceResponseUpdated"

6. **IF** la clonación ha expirado
   **THEN** el sistema SHALL retornar HTTP 410 Gone
   **AND** el mensaje SHALL ser "Clone has expired, cannot submit response"

7. **WHEN** el usuario consulta el historial de su clonación mediante GET /api/v1/clones/:id/history
   **THEN** el sistema SHALL retornar todas las transiciones de la instancia clonada
   **AND** el sistema SHALL incluir acciones: accepted, responded, updated
   **AND** el sistema SHALL NOT incluir el historial completo del trámite principal

#### R15.4: Consolidar Respuestas

**User Story**
Como consolidador designado, quiero recibir y consolidar todas las respuestas de las clonaciones, para continuar con el proceso principal.

**Acceptance Criteria**

1. **WHEN** el consolidador consulta las clonaciones mediante GET /api/v1/instances/:id/clones
   **THEN** el sistema SHALL validar que el actor sea el consolidador asignado
   **AND** el sistema SHALL retornar todas las instancias clonadas del grupo
   **AND** cada clonación SHALL incluir: id, assigned_user, current_state, response_data, responded_at, expires_at

2. **WHEN** se retorna la lista de clonaciones
   **THEN** la respuesta SHALL incluir un resumen:
   - `total_clones`: total de clones creados
   - `accepted_count`: clones aceptados
   - `responded_count`: clones con respuesta
   - `rejected_count`: clones rechazados
   - `pending_count`: clones sin respuesta
   - `all_responded`: boolean indicando si todos respondieron

3. **WHEN** el consolidador consulta el detalle de una clonación específica
   **THEN** el sistema SHALL validar que la clonación pertenezca al grupo del trámite principal
   **AND** el sistema SHALL retornar todos los detalles incluyendo response_data completo

4. **WHEN** el consolidador marca la consolidación como completa mediante POST /api/v1/instances/:id/clones/consolidate
   **THEN** el sistema SHALL validar que el actor sea el consolidador asignado
   **AND** el sistema SHALL validar que el estado del trámite principal permita consolidación
   **AND** el request SHALL incluir `consolidated_data` (JSONB con datos consolidados)
   **AND** el request PUEDE incluir `notes` (opcional)

5. **WHEN** se ejecuta la consolidación
   **THEN** el sistema SHALL actualizar el trámite principal con consolidated_data
   **AND** el sistema SHALL cambiar el estado del trámite principal según configuración del workflow
   **AND** el sistema SHALL marcar el clone_group como status = "consolidated"
   **AND** el sistema SHALL registrar consolidated_at = NOW()

6. **WHEN** la consolidación se completa exitosamente
   **THEN** el sistema SHALL generar evento "CloneGroupConsolidated"
   **AND** el sistema SHALL agregar registro detallado al historial del trámite principal
   **AND** el evento SHALL incluir resumen de respuestas (total, aceptadas, rechazadas)
   **AND** el sistema SHALL retornar HTTP 200 OK
   **AND** la respuesta SHALL incluir el nuevo estado del trámite principal

7. **IF** el consolidador rechaza clonaciones específicas mediante POST /api/v1/clones/:id/consolidator-reject
   **THEN** el sistema SHALL validar que el actor sea el consolidador
   **AND** el request SHALL incluir `rejection_reason`
   **AND** el sistema SHALL marcar la clonación como "rejected_by_consolidator"
   **AND** el sistema SHALL generar evento "ClonedInstanceRejectedByConsolidator"
   **AND** el sistema SHALL notificar al usuario asignado

8. **IF** el consolidador intenta consolidar sin tener respuestas suficientes
   **THEN** el sistema SHALL emitir advertencia pero permitir la consolidación
   **AND** el sistema SHALL registrar en el historial qué clonaciones no respondieron

9. **IF** otro usuario (no consolidador) intenta acceder a funciones de consolidación
   **THEN** el sistema SHALL retornar HTTP 403 Forbidden
   **AND** el mensaje SHALL ser "Only the designated consolidator can perform this action"

#### R15.5: Gestión de Tiempos en Clonación

**User Story**
Como sistema, quiero gestionar automáticamente los tiempos de las clonaciones y sus notificaciones, para asegurar cumplimiento de SLAs.

**Acceptance Criteria**

1. **WHEN** se crea un grupo de clonaciones
   **THEN** el sistema SHALL crear un timer en la tabla workflow_timers
   **AND** el timer SHALL incluir: clone_group_id, event_on_timeout="clone_expired", expires_at

2. **WHEN** el timer scheduler detecta clonaciones próximas a vencer (24h antes)
   **THEN** el sistema SHALL generar evento "CloneExpirationWarning"
   **AND** el sistema SHALL notificar a usuarios asignados que no han respondido
   **AND** el sistema SHALL notificar al consolidador con el estado actual

3. **WHEN** una clonación expira (expires_at <= NOW() y current_state != "responded")
   **THEN** el sistema SHALL generar evento "ClonedInstanceExpired"
   **AND** el sistema SHALL notificar al consolidador
   **AND** el sistema SHALL marcar la clonación como "expired"
   **AND** el sistema SHALL agregar registro al historial del trámite principal

4. **WHEN** el consolidador o gestor solicita extensión de tiempo mediante POST /api/v1/clones/groups/:group_id/extend
   **THEN** el sistema SHALL validar que el actor sea consolidador o gestor principal
   **AND** el request SHALL incluir `additional_duration` (ej: "12h", "2d")
   **AND** el request SHALL incluir `extension_reason`

5. **WHEN** se valida la extensión
   **THEN** el sistema SHALL calcular nuevo expires_at = current_expires_at + additional_duration
   **AND** el sistema SHALL validar que nuevo_expires_at <= (tiempo_vencimiento_tramite_principal * 0.80)
   **AND** el sistema SHALL retornar HTTP 422 si excede el límite máximo

6. **WHEN** la extensión es aprobada
   **THEN** el sistema SHALL actualizar expires_at en todas las clonaciones del grupo con state != "responded"
   **AND** el sistema SHALL generar evento "CloneGroupExtended"
   **AND** el sistema SHALL notificar a todos los usuarios asignados pendientes
   **AND** el sistema SHALL agregar registro al historial del trámite principal
   **AND** el sistema SHALL retornar HTTP 200 OK

7. **IF** todas las clonaciones han respondido antes del vencimiento
   **THEN** el sistema SHALL cancelar el timer automáticamente
   **AND** el sistema SHALL generar evento "CloneGroupCompletedEarly"

8. **WHEN** el timer scheduler procesa grupos de clones expirados
   **THEN** el procesamiento SHALL ser independiente de otros timers de workflow
   **AND** un error en un grupo de clones SHALL NOT afectar otros grupos

#### R15.6: Notificaciones y Trazabilidad

**User Story**
Como participante del proceso de clonación, quiero recibir notificaciones oportunas y ver trazabilidad completa, para mantenerme informado del progreso.

**Acceptance Criteria**

1. **WHEN** ocurre cualquier evento de clonación
   **THEN** el sistema SHALL registrar la acción en el historial del trámite principal
   **AND** cada registro SHALL incluir: timestamp, actor, action_type, clone_id (si aplica), details

2. **WHEN** se consulta el historial del trámite principal mediante GET /api/v1/instances/:id/history
   **THEN** el sistema SHALL incluir todas las acciones de clonación:
   - "clone_group_created"
   - "cloned_instance_created"
   - "cloned_instance_accepted"
   - "cloned_instance_rejected"
   - "cloned_instance_responded"
   - "clone_group_consolidated"
   - "clone_group_extended"
   - "cloned_instance_expired"

3. **WHEN** se genera un evento de clonación
   **THEN** el evento SHALL incluir metadatos completos:
   - `parent_instance_id`
   - `clone_group_id`
   - `cloned_instance_id` (si aplica)
   - `actor_id`
   - `summary`: resumen legible de la acción

4. **IF** se configuran webhooks para eventos de clonación
   **THEN** el sistema SHALL enviar notificaciones HTTP según R8.2
   **AND** los eventos de clonación SHALL ser filtrables en la configuración de webhooks

5. **WHEN** se configura el workflow para usar notificaciones
   **THEN** el sistema SHALL enviar notificaciones en estos escenarios:
   - Usuario asignado recibe nueva clonación (state: pending)
   - Consolidador recibe notificación de aceptación/rechazo
   - Consolidador recibe notificación de nueva respuesta
   - Gestor principal recibe notificación de rechazo
   - Todos los participantes reciben advertencia de expiración
   - Usuarios asignados reciben notificación de extensión de tiempo

6. **IF** se habilita audit trail detallado en configuración
   **THEN** el sistema SHALL registrar todos los accesos a datos del trámite principal por usuarios clonados
   **AND** el sistema SHALL registrar todas las modificaciones a response_data

7. **WHEN** el consolidador genera el reporte final mediante GET /api/v1/instances/:id/clones/report
   **THEN** el sistema SHALL generar un reporte consolidado incluyendo:
   - Resumen del grupo de clones (total, aceptados, rechazados, respondidos)
   - Timeline de eventos
   - Todas las respuestas agregadas
   - Métricas de tiempo (tiempo promedio de respuesta, extensiones, etc.)
   - Lista de usuarios que no respondieron

#### R15.7: Configuración de Clonación en Workflow

**User Story**
Como diseñador de workflows, quiero configurar el comportamiento de clonación en el archivo YAML del workflow, para habilitar/deshabilitar y personalizar esta funcionalidad.

**Acceptance Criteria**

1. **WHEN** se define un workflow con clonación habilitada
   **THEN** el archivo YAML SHALL incluir sección `cloning_config`:

```yaml
cloning_config:
  enabled: true
  allowed_states: ["assigned", "in_progress"]  # Estados desde los cuales se puede clonar
  allowed_roles: ["manager", "assigner"]  # Roles que pueden iniciar clonación
  consolidation_state: "cloned_awaiting_consolidation"  # Estado del padre mientras espera consolidación
  post_consolidation_allowed_events: ["approve", "request_changes"]  # Eventos permitidos después de consolidar

  # Configuración de tiempos
  time_restrictions:
    max_percentage: 80  # Máximo 80% del tiempo restante del trámite principal
    warning_before_expiration: "24h"  # Advertir 24h antes del vencimiento
    allow_extensions: true
    max_extensions: 2  # Máximo 2 extensiones permitidas

  # Configuración de instancias clonadas
  cloned_instance_workflow:
    states:
      - id: "pending"
        type: "initial"
      - id: "accepted"
        type: "normal"
      - id: "responded"
        type: "final"
      - id: "rejected"
        type: "final"
      - id: "expired"
        type: "final"

  # Campos requeridos en las respuestas
  required_response_fields:
    - field: "response_text"
      type: "string"
      validation: "min_length:10"
    - field: "attachments"
      type: "array"
      validation: "optional"

  # Configuración de visibilidad
  parent_data_visibility:
    mode: "limited"  # "full" | "limited" | "custom"
    allowed_fields: ["case_number", "subject", "filed_date", "current_state"]

  # Notificaciones
  notifications:
    on_clone_created: true
    on_acceptance: true
    on_rejection: true
    on_response: true
    on_expiration_warning: true
    on_expired: true
    on_consolidated: true
```

2. **WHEN** se valida la configuración del workflow
   **THEN** el sistema SHALL verificar que:
   - `allowed_states` contenga estados existentes en el workflow
   - `consolidation_state` exista en el workflow
   - `max_percentage` esté entre 1 y 100
   - `cloned_instance_workflow.states` incluya al menos los estados básicos

3. **IF** `enabled = false` en cloning_config
   **THEN** todos los endpoints de clonación SHALL retornar HTTP 403 Forbidden para ese workflow

4. **IF** no se especifica cloning_config en el workflow
   **THEN** el sistema SHALL asumir que la clonación está deshabilitada por defecto

5. **WHEN** se actualiza un workflow con nueva configuración de clonación
   **THEN** los cambios SHALL aplicarse solo a nuevas clonaciones
   **AND** las clonaciones activas SHALL continuar usando la configuración original

#### R15.8: Permisos y Control de Acceso en Clonación

**User Story**
Como sistema, quiero validar permisos específicos para operaciones de clonación, para garantizar seguridad y control de acceso apropiado.

**Acceptance Criteria**

1. **WHEN** un actor intenta crear una clonación
   **THEN** el sistema SHALL validar que el actor tenga uno de los roles en `allowed_roles` del workflow
   **AND** el sistema SHALL validar que el estado actual esté en `allowed_states`
   **AND** el sistema SHALL validar que el actor sea el current_actor del trámite o tenga permisos de override

2. **WHEN** un usuario asignado accede a una instancia clonada
   **THEN** el sistema SHALL validar que user_id = assigned_user_id
   **AND** el sistema SHALL aplicar restricciones de visibilidad según `parent_data_visibility`

3. **WHEN** se aplica visibilidad limitada (mode: "limited")
   **THEN** el sistema SHALL retornar solo los campos especificados en `allowed_fields`
   **AND** el sistema SHALL ocultar datos sensibles del trámite principal
   **AND** el sistema SHALL NOT permitir modificación de datos del padre (solo lectura)

4. **WHEN** el consolidador accede a las clonaciones
   **THEN** el sistema SHALL validar que actor_id = consolidator_id
   **AND** el sistema SHALL permitir acceso completo a todas las clonaciones del grupo
   **AND** el sistema SHALL permitir acceso completo a datos del trámite principal

5. **IF** el gestor principal o consolidador puede reasignar una clonación rechazada
   **THEN** el endpoint POST /api/v1/clones/:id/reassign SHALL estar disponible
   **AND** el request SHALL incluir `new_assignee_id` y `reassignment_reason`
   **AND** el sistema SHALL crear una nueva instancia clonada con el nuevo usuario
   **AND** el sistema SHALL marcar la anterior como "reassigned"
   **AND** el sistema SHALL generar evento "ClonedInstanceReassigned"

6. **WHEN** se consultan métricas de clonación mediante GET /api/v1/queries/cloning-statistics
   **THEN** el sistema SHALL retornar estadísticas agregadas:
   - Total de grupos de clones por workflow
   - Tasa de aceptación promedio
   - Tasa de respuesta promedio
   - Tiempo promedio de respuesta
   - Número de extensiones otorgadas
   - Clonaciones expiradas vs completadas

7. **IF** se configura audit_trail = true para el workflow
   **THEN** todas las acciones de usuarios clonados SHALL registrarse en tabla de auditoría
   **AND** el registro SHALL incluir: user_id, action, timestamp, ip_address, accessed_data

---

## Priorización de Requerimientos

### Must Have (P0)
- R1.1, R1.2: Gestión básica de workflows
- R2.1, R2.2: Creación y consulta de instancias
- R3.1, R3.2: Ejecución de transiciones con optimistic locking
- R8.1: Publicación de eventos de dominio
- R10.1: Hybrid repository
- R11.3: Health check
- R13.1: Graceful shutdown

### Should Have (P1)
- R1.3: Actualización de workflows
- R2.3: Historial de transiciones
- R4: Ciclo de vida (pausar, reanudar, cancelar)
- R5: Timers y escalamientos
- R7: Actores y roles
- R8.2: Webhooks
- R9: Queries avanzadas
- R11.1, R11.2: Observabilidad completa
- R15.1, R15.2, R15.3, R15.4: Clonación básica de instancias

### Could Have (P2)
- R6: Subprocesos jerárquicos
- R9.2: Estadísticas y métricas
- R12: Seguridad avanzada
- R14: Performance optimizations
- R15.5, R15.6, R15.7, R15.8: Gestión avanzada de clonación (tiempos, notificaciones, configuración, permisos)

### Won't Have (v1.0)
- UI gráfica para diseñar workflows
- Multi-tenancy
- Workflow versioning con rollback automático
- Machine learning para optimización de rutas

---

**Versión**: 1.0
**Fecha**: 2025-01-15
**Próxima Revisión**: 2025-02-15
