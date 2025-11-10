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

9. **WHEN** el estado destino es el mismo que el estado origen (loop/stay)
   **THEN** el sistema SHALL permitir la transición (reentrada)
   **AND** el sistema SHALL registrar la transición en el historial normalmente
   **AND** el sistema SHALL considerar esto como transición válida (NO es un error)
   **EXAMPLE**: Estado "in_progress" con evento "reclassify" que retorna a "in_progress"

10. **WHEN** el estado destino es un estado previamente visitado (reentrada desde otro estado)
    **THEN** el sistema SHALL permitir la transición sin restricciones
    **AND** el historial SHALL mostrar claramente la secuencia (ej: A→B→C→B indica reentrada a B)
    **AND** las métricas SHALL contar esto como transición adicional
    **EXAMPLE**: "in_review" → "in_progress" (reject) es válido aunque ya se estuvo en "in_progress"

11. **IF** otro proceso tiene el lock de la instancia
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

#### R15.9: Clonación con Aprobación Previa (Integración con R18)

**User Story**
Como administrador de procesos, quiero que ciertas clonaciones requieran aprobación de un departamento o supervisor antes de crearse, para mantener control sobre operaciones sensibles o costosas.

**Acceptance Criteria**

1. **WHEN** se configura un workflow con clonación que requiere aprobación
   **THEN** el archivo YAML SHALL incluir en `cloning_config`:
   ```yaml
   cloning_config:
     enabled: true
     requires_approval: true
     approval_config:
       type: "escalation"  # Usa el sistema de escalamientos (R18)
       department_id: "supervisors"  # Departamento que debe aprobar
       approval_guard: has_role:supervisor  # Guard adicional
       auto_approve_conditions:  # Opcional: casos de auto-aprobación
         - field_equals:data.priority:low
         - clone_count_less_than:3
   ```

2. **WHEN** un gestor inicia una clonación que requiere aprobación mediante POST /api/v1/instances/:id/clone
   **THEN** el sistema SHALL validar todas las condiciones normales de R15.1
   **AND** el sistema SHALL verificar `requires_approval` en la configuración del workflow
   **AND** si `requires_approval = true`, el sistema SHALL proceder al flujo de aprobación

3. **WHEN** se determina que la clonación requiere aprobación
   **THEN** el sistema SHALL evaluar `auto_approve_conditions` (si existen)
   **AND** si todas las condiciones de auto-aprobación se cumplen, el sistema SHALL crear las instancias clonadas inmediatamente
   **AND** si NO se cumplen las condiciones, el sistema SHALL crear un escalamiento (R18) antes de crear las clonaciones

4. **WHEN** se crea el escalamiento de aprobación
   **THEN** el sistema SHALL usar el endpoint interno equivalente a POST /api/v1/instances/:id/escalate
   **AND** el escalamiento SHALL incluir:
   - `department_id`: del approval_config
   - `reason`: "Clone approval request: {clone_reason}"
   - `metadata`: incluye toda la información del request de clonación:
     ```json
     {
       "clone_request": {
         "assignees": [...],
         "consolidator_id": "...",
         "reason": "...",
         "timeout_duration": "...",
         "metadata": {...}
       },
       "approval_type": "clone_creation"
     }
     ```
   **AND** el sistema SHALL cambiar el subestado de la instancia principal a "awaiting_clone_approval"
   **AND** el sistema SHALL retornar HTTP 202 Accepted (en lugar de 201 Created)
   **AND** la respuesta SHALL incluir:
   ```json
   {
     "status": "pending_approval",
     "escalation_id": "uuid-del-escalamiento",
     "approval_required_from": "supervisors",
     "message": "Clone request submitted for approval"
   }
   ```

5. **WHEN** el departamento aprueba el escalamiento mediante POST /api/v1/instances/:id/escalation-reply
   **THEN** el sistema SHALL validar que response contenga aprobación explícita
   **AND** el sistema SHALL extraer el clone_request original del metadata del escalamiento
   **AND** el sistema SHALL ejecutar la creación de clonaciones automáticamente con los parámetros originales
   **AND** el sistema SHALL cambiar el subestado de la instancia a "clone_approved_in_progress"
   **AND** el sistema SHALL generar evento de dominio "CloneApprovedAndCreated"
   **AND** el sistema SHALL notificar al gestor original que las clonaciones fueron aprobadas y creadas

6. **WHEN** el departamento rechaza el escalamiento
   **THEN** el sistema SHALL marcar el escalamiento como "closed" sin crear las clonaciones
   **AND** el sistema SHALL cambiar el subestado de la instancia de vuelta al anterior
   **AND** el sistema SHALL generar evento "CloneRequestRejected"
   **AND** el sistema SHALL notificar al gestor original con el motivo del rechazo
   **AND** el sistema SHALL registrar en el historial del trámite principal el rechazo de la clonación

7. **WHEN** se consulta el estado de una solicitud de clonación pendiente de aprobación
   **THEN** el endpoint GET /api/v1/instances/:id/clone-requests SHALL estar disponible
   **AND** el sistema SHALL retornar:
   ```json
   {
     "pending_requests": [
       {
         "escalation_id": "...",
         "requested_at": "...",
         "requested_by": "...",
         "status": "pending_approval",
         "approval_department": "supervisors",
         "assignees_count": 5,
         "timeout_duration": "24h"
       }
     ]
   }
   ```

8. **IF** el gestor cancela la solicitud de clonación antes de la aprobación
   **THEN** el endpoint DELETE /api/v1/escalations/:escalation_id SHALL permitir la cancelación
   **AND** el sistema SHALL validar que metadata.approval_type = "clone_creation"
   **AND** el sistema SHALL validar que el actor sea el solicitante original
   **AND** el sistema SHALL cerrar el escalamiento con status = "canceled"
   **AND** el sistema SHALL restaurar el subestado de la instancia

9. **WHEN** se configura tiempo de expiración para aprobaciones
   **THEN** el escalamiento SHALL tener su propio expires_at independiente
   **AND** si el escalamiento expira sin respuesta, el sistema SHALL rechazar automáticamente la clonación
   **AND** el sistema SHALL generar evento "CloneRequestExpiredWithoutApproval"

10. **WHEN** el departamento aprobador solicita modificaciones antes de aprobar
    **THEN** el sistema SHALL permitir respuesta con status = "requires_changes"
    **AND** el metadata de la respuesta SHALL incluir `requested_changes`
    **AND** el sistema SHALL notificar al gestor original
    **AND** el gestor SHALL poder reenviar la solicitud con POST /api/v1/instances/:id/clone/resubmit
    **AND** el resubmit SHALL actualizar el escalamiento existente (no crear uno nuevo)

11. **WHEN** se consultan estadísticas de aprobaciones de clonación
    **THEN** GET /api/v1/queries/clone-approval-statistics SHALL incluir:
    - Total de solicitudes de clonación
    - Tasa de aprobación
    - Tiempo promedio de aprobación
    - Solicitudes rechazadas vs aprobadas
    - Solicitudes expiradas sin respuesta

**Ejemplo de Flujo Completo**:

```
1. Gestor solicita clonar a 5 usuarios
   → Sistema detecta requires_approval: true
   → Sistema crea escalamiento a "supervisors"
   → Instancia pasa a subestado "awaiting_clone_approval"
   → HTTP 202 Accepted

2. Supervisor revisa y aprueba
   → POST /api/v1/instances/:id/escalation-reply
   → Sistema lee clone_request del metadata
   → Sistema crea automáticamente 5 instancias clonadas
   → Instancia pasa a subestado "clone_approved_in_progress"
   → Notifica al gestor original

3. Usuarios clonados reciben sus asignaciones
   → Flujo normal de R15.2-R15.6 continúa
```

**Separación de Conceptos Mantenida**:
- Escalamiento sigue siendo consulta/aprobación externa (R18)
- Clonación sigue siendo distribución de trabajo (R15)
- La integración es opcional y configurable
- Escalamiento es un prerequisito, no el mismo concepto
- Ambos sistemas funcionan independientemente cuando no hay integración

---

### R16: Formato de API - JSON:API Specification

El sistema debe adoptar la especificación JSON:API (https://jsonapi.org/) para todos los responses de la API REST, proporcionando consistencia, estandarización y mejor manejo de relaciones complejas.

#### R16.1: Estructura de Responses

**User Story**
Como cliente de la API, quiero recibir responses en formato JSON:API estándar, para tener consistencia y poder usar librerías cliente existentes.

**Acceptance Criteria**

1. **WHEN** la API retorna un recurso individual exitosamente
   **THEN** el response SHALL seguir la estructura JSON:API:
   ```json
   {
     "data": {
       "type": "instance",
       "id": "550e8400-e29b-41d4-a716-446655440000",
       "attributes": {
         "workflow_id": "radicacion",
         "current_state": "gestionar",
         "previous_state": "asignar",
         "status": "running",
         "version": 2,
         "created_at": "2025-01-15T10:00:00Z",
         "updated_at": "2025-01-15T10:05:00Z"
       },
       "relationships": {
         "workflow": {
           "data": { "type": "workflow", "id": "radicacion" },
           "links": {
             "self": "/api/v1/instances/550e8400-.../relationships/workflow",
             "related": "/api/v1/workflows/radicacion"
           }
         },
         "parent": {
           "data": null
         },
         "subprocesses": {
           "data": [],
           "meta": { "count": 0 }
         },
         "clones": {
           "data": [
             { "type": "clone", "id": "abc-123-..." }
           ],
           "meta": { "count": 1 }
         }
       },
       "links": {
         "self": "/api/v1/instances/550e8400-..."
       }
     },
     "meta": {
       "request_id": "req-xyz-789",
       "timestamp": "2025-01-15T10:05:30Z"
     }
   }
   ```

2. **WHEN** la API retorna una colección de recursos
   **THEN** el response SHALL usar array en `data`:
   ```json
   {
     "data": [
       {
         "type": "instance",
         "id": "...",
         "attributes": { ... }
       },
       {
         "type": "instance",
         "id": "...",
         "attributes": { ... }
       }
     ],
     "meta": {
       "total": 95,
       "page": {
         "number": 1,
         "size": 10,
         "total_pages": 10
       }
     },
     "links": {
       "self": "/api/v1/instances?page[number]=1&page[size]=10",
       "first": "/api/v1/instances?page[number]=1&page[size]=10",
       "prev": null,
       "next": "/api/v1/instances?page[number]=2&page[size]=10",
       "last": "/api/v1/instances?page[number]=10&page[size]=10"
     }
   }
   ```

3. **WHEN** se incluyen recursos relacionados
   **THEN** el sistema SHALL usar el top-level member `included`:
   ```json
   {
     "data": {
       "type": "instance",
       "id": "550e8400-...",
       "attributes": { ... },
       "relationships": {
         "workflow": {
           "data": { "type": "workflow", "id": "radicacion" }
         }
       }
     },
     "included": [
       {
         "type": "workflow",
         "id": "radicacion",
         "attributes": {
           "name": "Flujo de Radicación",
           "description": "Proceso completo de radicación",
           "version": "1.0"
         }
       }
     ]
   }
   ```

4. **WHEN** el cliente especifica `include` query parameter
   **THEN** el sistema SHALL incluir los recursos relacionados especificados
   **AND** ejemplos válidos: `?include=workflow`, `?include=workflow,clones`, `?include=clones.responses`

5. **WHEN** el cliente especifica `fields` query parameter (sparse fieldsets)
   **THEN** el sistema SHALL retornar solo los campos solicitados
   **AND** ejemplo: `?fields[instance]=current_state,status,version`

#### R16.2: Estructura de Errores

**User Story**
Como cliente de la API, quiero recibir errores en formato JSON:API consistente, para manejar errores de forma uniforme.

**Acceptance Criteria**

1. **WHEN** ocurre un error en la API
   **THEN** el response SHALL usar el top-level member `errors` (array):
   ```json
   {
     "errors": [
       {
         "id": "err-xyz-123",
         "status": "409",
         "code": "VERSION_CONFLICT",
         "title": "Optimistic Lock Conflict",
         "detail": "Instance was modified by another process. Please retry with the latest version.",
         "source": {
           "pointer": "/data/attributes/version"
         },
         "meta": {
           "expected_version": 2,
           "current_version": 3
         }
       }
     ],
     "meta": {
       "request_id": "req-xyz-789",
       "timestamp": "2025-01-15T10:05:30Z"
     }
   }
   ```

2. **WHEN** ocurren múltiples errores de validación
   **THEN** el sistema SHALL retornar todos los errores en el array:
   ```json
   {
     "errors": [
       {
         "status": "400",
         "code": "VALIDATION_ERROR",
         "title": "Invalid Input",
         "detail": "workflow_id is required",
         "source": { "pointer": "/data/attributes/workflow_id" }
       },
       {
         "status": "400",
         "code": "VALIDATION_ERROR",
         "title": "Invalid Input",
         "detail": "actor_id must be a valid UUID",
         "source": { "pointer": "/data/attributes/actor_id" }
       }
     ]
   }
   ```

3. **WHEN** se mapean errores de dominio a HTTP status codes
   **THEN** el sistema SHALL usar los siguientes códigos estándar:
   - `400` Bad Request: Validación de input fallida
   - `401` Unauthorized: Token JWT inválido o faltante
   - `403` Forbidden: Permisos insuficientes, rol no autorizado
   - `404` Not Found: Recurso no existe
   - `409` Conflict: Version conflict, invalid transition, instance locked
   - `410` Gone: Recurso expirado (clonaciones)
   - `422` Unprocessable Entity: Validación de negocio fallida
   - `429` Too Many Requests: Rate limit excedido
   - `500` Internal Server Error: Error inesperado del sistema
   - `503` Service Unavailable: Dependencia (DB, Redis) no disponible

4. **WHEN** se retorna un error
   **THEN** cada error object SHALL incluir:
   - `status`: HTTP status code como string
   - `code`: Código de error específico del sistema (ej: "VERSION_CONFLICT", "INVALID_TRANSITION")
   - `title`: Resumen legible del error
   - `detail`: Descripción detallada específica de esta ocurrencia
   - `source` (opcional): Ubicación del error (pointer para JSON, parameter para query params)
   - `meta` (opcional): Metadatos adicionales del error

#### R16.3: Paginación

**User Story**
Como cliente de la API, quiero paginar resultados usando el estándar JSON:API, para manejar grandes volúmenes de datos eficientemente.

**Acceptance Criteria**

1. **WHEN** el cliente solicita una colección paginada
   **THEN** el sistema SHALL soportar query parameters:
   - `page[number]`: Número de página (base 1)
   - `page[size]`: Tamaño de página (default: 20, max: 100)
   - Ejemplo: `/api/v1/instances?page[number]=2&page[size]=50`

2. **WHEN** se retorna una colección paginada
   **THEN** el response SHALL incluir:
   ```json
   {
     "data": [ ... ],
     "meta": {
       "total": 250,
       "page": {
         "number": 2,
         "size": 50,
         "total_pages": 5
       }
     },
     "links": {
       "self": "/api/v1/instances?page[number]=2&page[size]=50",
       "first": "/api/v1/instances?page[number]=1&page[size]=50",
       "prev": "/api/v1/instances?page[number]=1&page[size]=50",
       "next": "/api/v1/instances?page[number]=3&page[size]=50",
       "last": "/api/v1/instances?page[number]=5&page[size]=50"
     }
   }
   ```

3. **IF** no hay página siguiente
   **THEN** `links.next` SHALL ser `null`

4. **IF** no hay página anterior
   **THEN** `links.prev` SHALL ser `null`

5. **WHEN** se especifica `page[size]` > 100
   **THEN** el sistema SHALL usar `page[size] = 100` (máximo)
   **AND** el sistema SHALL incluir advertencia en `meta.warnings`

#### R16.4: Filtrado y Sorting

**User Story**
Como cliente de la API, quiero filtrar y ordenar colecciones usando el estándar JSON:API, para obtener exactamente los datos que necesito.

**Acceptance Criteria**

1. **WHEN** el cliente aplica filtros
   **THEN** el sistema SHALL soportar sintaxis `filter[campo]=valor`:
   - `/api/v1/instances?filter[workflow_id]=radicacion`
   - `/api/v1/instances?filter[status]=running`
   - `/api/v1/instances?filter[current_state]=gestionar,revisar` (OR)

2. **WHEN** el cliente ordena resultados
   **THEN** el sistema SHALL soportar `sort` parameter:
   - `/api/v1/instances?sort=created_at` (ascendente)
   - `/api/v1/instances?sort=-created_at` (descendente con -)
   - `/api/v1/instances?sort=-created_at,status` (múltiples campos)

3. **WHEN** se combinan filtros, sorting y paginación
   **THEN** todos los parameters SHALL funcionar juntos:
   ```
   /api/v1/instances?filter[workflow_id]=radicacion&filter[status]=running&sort=-created_at&page[number]=1&page[size]=20
   ```

4. **IF** se especifica un campo de ordenamiento inválido
   **THEN** el sistema SHALL retornar error 400 con detalle del campo inválido

#### R16.5: Tipos de Recursos

**User Story**
Como desarrollador de la API, quiero definir tipos de recursos consistentes, para mantener coherencia en toda la API.

**Acceptance Criteria**

1. **WHEN** se implementan recursos en la API
   **THEN** el sistema SHALL usar los siguientes tipos de recursos:
   - `workflow`: Definiciones de workflows
   - `instance`: Instancias de workflows
   - `transition`: Registros de transiciones en el historial
   - `clone`: Instancias clonadas
   - `clone-group`: Grupo de clonaciones
   - `actor`: Actores del sistema
   - `webhook`: Configuraciones de webhooks
   - `timer`: Timers activos

2. **WHEN** se retorna un recurso
   **THEN** el campo `type` SHALL ser singular y en kebab-case
   **AND** el campo `type` SHALL ser consistente en toda la API

3. **WHEN** se define un relationship
   **THEN** el nombre del relationship SHALL ser singular o plural según corresponda:
   - Singular: `workflow`, `parent`, `consolidator`
   - Plural: `subprocesses`, `clones`, `transitions`

#### R16.6: Content Negotiation

**User Story**
Como cliente de la API, quiero usar headers estándar JSON:API, para asegurar compatibilidad con librerías cliente.

**Acceptance Criteria**

1. **WHEN** el cliente envía un request con body
   **THEN** el cliente SHALL incluir header `Content-Type: application/vnd.api+json`
   **AND** el sistema SHALL retornar error 415 Unsupported Media Type si el Content-Type es incorrecto

2. **WHEN** el servidor retorna un response
   **THEN** el servidor SHALL incluir header `Content-Type: application/vnd.api+json`

3. **WHEN** el cliente envía header `Accept: application/vnd.api+json`
   **THEN** el servidor SHALL retornar responses en formato JSON:API

4. **IF** el cliente envía `Accept` con media type parameters no soportados
   **THEN** el servidor SHALL retornar error 406 Not Acceptable

#### R16.7: Operaciones de Escritura (POST, PATCH, DELETE)

**User Story**
Como cliente de la API, quiero enviar operaciones de escritura en formato JSON:API, para mantener consistencia en requests y responses.

**Acceptance Criteria**

1. **WHEN** se crea un recurso mediante POST
   **THEN** el request body SHALL seguir formato JSON:API:
   ```json
   {
     "data": {
       "type": "instance",
       "attributes": {
         "workflow_id": "radicacion",
         "actor_id": "user123",
         "actor_role": "radicador",
         "data": {
           "tipo": "PQRD",
           "remitente": "Juan Pérez"
         }
       }
     }
   }
   ```
   **AND** el sistema SHALL retornar HTTP 201 Created
   **AND** el response SHALL incluir el recurso creado con su `id` asignado

2. **WHEN** se actualiza un recurso mediante PATCH
   **THEN** el request body SHALL incluir solo los campos a actualizar:
   ```json
   {
     "data": {
       "type": "instance",
       "id": "550e8400-...",
       "attributes": {
         "status": "paused"
       }
     }
   }
   ```
   **AND** el sistema SHALL retornar HTTP 200 OK con el recurso actualizado

3. **WHEN** se ejecuta una acción (no CRUD estándar) como trigger evento
   **THEN** el endpoint SHALL seguir convención JSON:API para actions:
   ```
   POST /api/v1/instances/550e8400-.../events
   {
     "data": {
       "type": "event-trigger",
       "attributes": {
         "event": "generar_radicado",
         "actor": "user123",
         "data": { ... }
       }
     }
   }
   ```

4. **WHEN** se elimina un recurso mediante DELETE
   **THEN** el sistema SHALL retornar HTTP 204 No Content sin body
   **OR** HTTP 200 OK con meta information sobre la eliminación

#### R16.8: Metadatos Globales

**User Story**
Como cliente de la API, quiero recibir metadatos útiles en cada response, para debugging y tracking de requests.

**Acceptance Criteria**

1. **WHEN** la API retorna cualquier response exitoso
   **THEN** el response SHALL incluir `meta` con:
   ```json
   {
     "data": { ... },
     "meta": {
       "request_id": "req-abc-123-xyz",
       "timestamp": "2025-01-15T10:05:30.123Z",
       "api_version": "1.0",
       "performance": {
         "db_queries": 3,
         "cache_hits": 2,
         "duration_ms": 45
       }
     }
   }
   ```

2. **IF** el request tiene warnings (no errores)
   **THEN** el sistema SHALL incluir `meta.warnings`:
   ```json
   {
     "data": { ... },
     "meta": {
       "warnings": [
         {
           "code": "PAGE_SIZE_CAPPED",
           "message": "Requested page size 150 exceeds maximum 100, capped to 100"
         }
       ]
     }
   }
   ```

3. **IF** se incluyen estadísticas o agregaciones
   **THEN** el sistema SHALL usar `meta` para datos no representados como recursos:
   ```json
   {
     "data": [ ... ],
     "meta": {
       "statistics": {
         "total_instances": 1250,
         "by_status": {
           "running": 800,
           "completed": 400,
           "paused": 50
         }
       }
     }
   }
   ```

---

### R17: Sistema de Subestados

Los subestados permiten modelar estados internos complejos sin fragmentar el flujo principal del workflow, proporcionando tracking detallado de transiciones dentro de un mismo estado.

#### R17.1: Definir Subestados dentro de Estados

**User Story**
Como diseñador de workflows, quiero definir subestados dentro de estados principales, para modelar procesos complejos con tracking detallado sin fragmentar el estado principal.

**Acceptance Criteria**

1. **WHEN** se define un estado en el workflow YAML
   **THEN** el sistema SHALL soportar campo opcional `substates: []`
   **AND** cada subestado SHALL tener: id, name, description
   **AND** el sistema SHALL validar que los IDs de subestados sean únicos dentro del estado

2. **WHEN** se valida un workflow con subestados
   **THEN** el sistema SHALL verificar que cada estado tenga máximo un nivel de subestados (no anidados)
   **AND** el sistema SHALL validar que si se definen subestados, el primero sea el subestado por defecto

3. **IF** el workflow YAML no define subestados para un estado
   **THEN** ese estado SHALL NOT soportar subestados
   **AND** intentar establecer un subestado SHALL retornar error

#### R17.2: Transiciones con Subestados

**User Story**
Como sistema, quiero gestionar transiciones que cambien subestados sin cambiar el estado principal, para tracking detallado de subprocesos.

**Acceptance Criteria**

1. **WHEN** una instancia entra a un estado con subestados
   **THEN** el sistema SHALL establecer `current_sub_state` al primer subestado definido
   **AND** el sistema SHALL persistir `current_sub_state` en tabla `workflow_instances`
   **AND** `previous_sub_state` SHALL ser NULL en la primera entrada

2. **WHEN** se ejecuta una transición que cambia de estado principal
   **THEN** el sistema SHALL:
   - Registrar `from_sub_state` en el historial (si aplica)
   - Establecer `to_sub_state` según el nuevo estado (si tiene subestados)
   - Actualizar `current_sub_state` en la instancia

3. **WHEN** se ejecuta una transición que permanece en el mismo estado principal
   **THEN** el sistema SHALL poder cambiar solo el subestado
   **AND** el sistema SHALL validar que la transición de subestado sea válida según el workflow
   **AND** el sistema SHALL registrar `from_sub_state` y `to_sub_state` en `workflow_transitions`

4. **WHEN** una instancia sale de un estado con subestados
   **THEN** el sistema SHALL limpiar `current_sub_state` a NULL (si el nuevo estado no tiene subestados)
   **AND** el sistema SHALL preservar `previous_sub_state` para auditoría

5. **IF** se intenta establecer un subestado inválido
   **THEN** el sistema SHALL retornar HTTP 400 Bad Request
   **AND** el mensaje SHALL indicar "Invalid substate for current state"

#### R17.3: Persistencia de Subestados

**User Story**
Como sistema, quiero persistir subestados en la base de datos, para auditoría completa y consultas.

**Acceptance Criteria**

1. **WHEN** se actualiza el schema de base de datos
   **THEN** la tabla `workflow_instances` SHALL incluir:
   - `current_sub_state VARCHAR(100)` (nullable)
   - `previous_sub_state VARCHAR(100)` (nullable)

2. **WHEN** se actualiza el schema de transiciones
   **THEN** la tabla `workflow_transitions` SHALL incluir:
   - `from_sub_state VARCHAR(100)` (nullable)
   - `to_sub_state VARCHAR(100)` (nullable)

3. **WHEN** se consulta el historial de una instancia
   **THEN** cada transición SHALL incluir campos de subestados (si aplican)
   **AND** el sistema SHALL mostrar claramente transiciones de solo subestado vs estado completo

4. **IF** se crean índices para queries
   **THEN** el sistema SHALL crear índice compuesto: `(current_state, current_sub_state, status)`

#### R17.4: API para Subestados

**User Story**
Como cliente de la API, quiero consultar y filtrar instancias por subestados, para reporting detallado.

**Acceptance Criteria**

1. **WHEN** se consulta una instancia mediante GET /api/v1/instances/:id
   **THEN** la respuesta SHALL incluir:
   ```json
   {
     "current_state": "in_progress",
     "current_sub_state": "escalated_awaiting_response",
     "previous_state": "assigned",
     "previous_sub_state": null
   }
   ```

2. **WHEN** se ejecuta query de instancias mediante POST /api/v1/queries/instances
   **THEN** el sistema SHALL soportar filtro `sub_states: ["working", "escalated"]`
   **AND** el filtro SHALL aplicarse solo a instancias en estados que soporten subestados

3. **WHEN** se solicita el historial mediante GET /api/v1/instances/:id/history
   **THEN** cada transición SHALL incluir:
   ```json
   {
     "from_state": "in_progress",
     "from_sub_state": "working",
     "to_state": "in_progress",
     "to_sub_state": "escalated_awaiting_response",
     "event": "escalate"
   }
   ```

---

### R18: Escalamientos Manuales

El sistema debe permitir escalar instancias a departamentos o usuarios externos para consulta especializada, manteniendo el estado principal y creando un registro de auditoría.

#### R18.1: Escalar Instancia a Departamento

**User Story**
Como gestionador, quiero escalar un documento a otro departamento para consulta especializada, sin cambiar el estado principal del workflow.

**Acceptance Criteria**

1. **WHEN** el usuario ejecuta escalamiento mediante POST /api/v1/instances/:id/escalate
   **THEN** el sistema SHALL validar que la instancia exista
   **AND** el sistema SHALL validar que el estado actual permita escalamientos (según workflow config)
   **AND** el request SHALL incluir: `department_id`, `reason`
   **AND** el sistema SHALL validar que el actor tenga permisos para escalar

2. **WHEN** se configura escalamiento en workflow YAML
   **THEN** el sistema SHALL soportar configuración:
   ```yaml
   states:
     - id: in_progress
       allow_escalation: true
       escalation_guard: has_role:gestionador
   ```

3. **WHEN** se crea un escalamiento válido
   **THEN** el sistema SHALL crear registro en tabla `workflow_escalations` con:
   - `id` (UUID)
   - `instance_id` (FK a workflow_instances)
   - `department_id`
   - `reason` (TEXT)
   - `escalated_by` (actor que escaló)
   - `escalated_at` (timestamp)
   - `status` = "pending"
   **AND** el sistema SHALL cambiar `current_sub_state` a configuración del workflow (ej: "escalated_awaiting_response")
   **AND** el sistema SHALL generar evento de dominio "DocumentEscalated"

4. **WHEN** el escalamiento se crea exitosamente
   **THEN** el sistema SHALL retornar HTTP 201 Created
   **AND** la respuesta SHALL incluir:
   ```json
   {
     "escalation_id": "esc-uuid",
     "instance_id": "inst-uuid",
     "department_id": "legal",
     "status": "pending",
     "escalated_at": "2025-11-05T10:00:00Z"
   }
   ```

5. **IF** el estado actual no permite escalamientos
   **THEN** el sistema SHALL retornar HTTP 409 Conflict
   **AND** el mensaje SHALL ser "Escalation not allowed from current state"

6. **IF** ya existe un escalamiento pendiente al mismo departamento
   **THEN** el sistema SHALL permitir crear otro escalamiento (no limitar)
   **OR** el sistema SHALL retornar error si la configuración lo indica

#### R18.2: Responder Escalamiento

**User Story**
Como departamento especializado, quiero responder a escalamientos enviando mi conclusión, para que el gestor continúe el proceso.

**Acceptance Criteria**

1. **WHEN** se responde un escalamiento mediante POST /api/v1/instances/:id/escalation-reply
   **THEN** el sistema SHALL validar que el escalamiento exista
   **AND** el sistema SHALL validar que `status = "pending"`
   **AND** el request SHALL incluir: `escalation_id`, `response` (TEXT)

2. **WHEN** la respuesta es válida
   **THEN** el sistema SHALL actualizar el registro de escalamiento:
   - `response` = contenido de la respuesta
   - `responded_by` = actor que responde
   - `responded_at` = NOW()
   - `status` = "responded"
   **AND** el sistema SHALL cambiar `current_sub_state` según configuración (ej: "escalation_responded")
   **AND** el sistema SHALL generar evento "EscalationReplied"

3. **WHEN** se genera el evento "EscalationReplied"
   **THEN** el evento SHALL incluir:
   - `instance_id`
   - `escalation_id`
   - `department_id`
   - `response` (resumen o completo según config)
   - `responded_by`

4. **IF** se configuran notificaciones
   **THEN** el sistema SHALL notificar al gestor original del escalamiento
   **AND** la notificación SHALL incluir el response del departamento

5. **IF** el escalamiento ya fue respondido
   **THEN** el sistema SHALL retornar HTTP 409 Conflict
   **AND** el mensaje SHALL ser "Escalation already responded"

#### R18.3: Consultar Escalamientos

**User Story**
Como usuario, quiero consultar los escalamientos de una instancia, para seguimiento del proceso.

**Acceptance Criteria**

1. **WHEN** se consultan escalamientos mediante GET /api/v1/instances/:id/escalations
   **THEN** el sistema SHALL retornar todos los escalamientos de la instancia
   **AND** los escalamientos SHALL estar ordenados por `escalated_at DESC`
   **AND** cada escalamiento SHALL incluir:
   ```json
   {
     "id": "esc-uuid",
     "department_id": "legal",
     "reason": "Requiere revisión legal",
     "status": "responded",
     "escalated_by": "user-123",
     "escalated_at": "2025-11-05T10:00:00Z",
     "response": "Aprobado legalmente",
     "responded_by": "user-legal-01",
     "responded_at": "2025-11-05T14:00:00Z"
   }
   ```

2. **WHEN** se consulta un escalamiento específico mediante GET /api/v1/escalations/:escalation_id
   **THEN** el sistema SHALL retornar detalles completos del escalamiento
   **AND** la respuesta SHALL incluir link a la instancia relacionada

3. **WHEN** se consultan escalamientos pendientes por departamento
   **THEN** el endpoint GET /api/v1/queries/escalations?department_id=legal&status=pending
   **SHALL** retornar todos los escalamientos pendientes para ese departamento
   **AND** soportar paginación con limit/offset

4. **IF** se implementan métricas
   **THEN** el sistema SHALL exponer:
   - `escalations_total{department_id, status}`
   - `escalation_response_duration_seconds` (tiempo de respuesta)

#### R18.4: Cerrar o Cancelar Escalamientos

**User Story**
Como gestionador, quiero poder cerrar o cancelar escalamientos que ya no son necesarios.

**Acceptance Criteria**

1. **WHEN** se cierra un escalamiento mediante POST /api/v1/escalations/:id/close
   **THEN** el sistema SHALL cambiar `status` a "closed"
   **AND** el sistema SHALL registrar `closed_by` y `closed_at`
   **AND** el sistema SHALL generar evento "EscalationClosed"

2. **WHEN** se cancela un escalamiento mediante DELETE /api/v1/escalations/:id
   **THEN** el sistema SHALL validar que `status = "pending"` (solo pendientes pueden cancelarse)
   **AND** el sistema SHALL cambiar `status` a "canceled"
   **AND** el sistema SHALL retornar el subestado al valor previo al escalamiento

3. **IF** el escalamiento ya fue respondido
   **THEN** no se puede cancelar, solo cerrar

---

### R19: Reclasificación de Instancias

El sistema debe permitir cambiar el tipo o categoría de una instancia sin alterar su estado en el workflow, para corregir clasificaciones erróneas o adaptarse a cambios de normativa.

#### R19.1: Reclasificar Tipo de Documento

**User Story**
Como gestionador senior, quiero reclasificar un documento a otro tipo (ej: PQRD → Control Interno), sin cambiar su estado en el workflow.

**Acceptance Criteria**

1. **WHEN** se ejecuta reclasificación mediante POST /api/v1/instances/:id/reclassify
   **THEN** el sistema SHALL validar que la instancia exista
   **AND** el request SHALL incluir: `new_type`, `reason`
   **AND** el sistema SHALL validar que `new_type` sea diferente del tipo actual

2. **WHEN** se configura reclasificación en workflow YAML
   **THEN** el sistema SHALL soportar:
   ```yaml
   reclassification:
     enabled: true
     allowed_types: ["PQRD", "Control", "Queja", "Reclamo", "Sugerencia"]
     allowed_from_states: ["in_progress", "in_review"]
     required_role: gestionador
     requires_senior: true  # Solo gestionadores senior
   ```

3. **WHEN** se valida la reclasificación
   **THEN** el sistema SHALL ejecutar guard personalizado (si configurado)
   **AND** el guard SHALL verificar:
   - Actor tiene rol requerido
   - Estado actual permite reclasificación
   - new_type está en la lista de tipos permitidos
   - Actor tiene flag `is_senior = true` (si requires_senior)

4. **WHEN** la reclasificación es válida
   **THEN** el sistema SHALL:
   - Actualizar `data.tipo` (o campo configurado) con new_type
   - Mantener `current_state` sin cambios
   - Incrementar `version`
   - Crear registro en `workflow_transitions` con `event = "reclassify"`
   - Incluir `reason` en el campo `reason` de la transición
   - Agregar metadata: `{"from_type": "...", "to_type": "..."}`
   **AND** el sistema SHALL generar evento "DocumentReclassified"

5. **WHEN** se genera el evento "DocumentReclassified"
   **THEN** el evento SHALL incluir:
   ```json
   {
     "type": "document.reclassified",
     "instance_id": "inst-uuid",
     "from_type": "Consulta",
     "to_type": "PQRD",
     "reason": "Análisis indica que es petición formal",
     "actor_id": "user-senior-01",
     "occurred_at": "2025-11-05T10:00:00Z"
   }
   ```

6. **WHEN** la reclasificación se completa exitosamente
   **THEN** el sistema SHALL retornar HTTP 200 OK
   **AND** la respuesta SHALL incluir:
   ```json
   {
     "instance_id": "inst-uuid",
     "from_type": "Consulta",
     "to_type": "PQRD",
     "current_state": "in_progress",
     "version": 5,
     "reclassified_at": "2025-11-05T10:00:00Z"
   }
   ```

7. **IF** el actor no tiene permisos
   **THEN** el sistema SHALL retornar HTTP 403 Forbidden
   **AND** el mensaje SHALL indicar el permiso faltante

8. **IF** el estado actual no permite reclasificación
   **THEN** el sistema SHALL retornar HTTP 409 Conflict
   **AND** el mensaje SHALL ser "Reclassification not allowed from state {current_state}"

#### R19.2: Historial de Reclasificaciones

**User Story**
Como auditor, quiero ver el historial completo de reclasificaciones de una instancia, para verificar cambios de categoría.

**Acceptance Criteria**

1. **WHEN** se consulta el historial mediante GET /api/v1/instances/:id/history
   **THEN** las transiciones de tipo `reclassify` SHALL aparecer claramente identificadas
   **AND** cada reclasificación SHALL mostrar:
   - `from_type` y `to_type` en metadata
   - `reason` del cambio
   - Actor que ejecutó
   - Timestamp

2. **WHEN** se consulta específicamente reclasificaciones mediante GET /api/v1/instances/:id/reclassifications
   **THEN** el sistema SHALL retornar solo las transiciones de tipo `reclassify`
   **AND** la respuesta SHALL incluir estadísticas: total de reclasificaciones, tipos únicos

3. **IF** se implementan métricas
   **THEN** el sistema SHALL exponer:
   - `reclassifications_total{from_type, to_type}`
   - Tasa de reclasificación por workflow

---

### R20: Guards de Transición Avanzados

El sistema debe soportar validadores customizados más allá de roles simples, permitiendo implementar lógica de negocio compleja en los permisos de transiciones.

#### R20.1: Guards Personalizados

**User Story**
Como diseñador de workflows, quiero definir validators customizados más allá de roles simples, para implementar lógica de negocio compleja en permisos de transiciones.

**Acceptance Criteria**

1. **WHEN** se define un evento en el workflow YAML
   **THEN** el sistema SHALL soportar campo `guards: []` con múltiples validators
   **AND** cada guard puede ser:
   - Guard simple por nombre: `has_role:gestionador`
   - Guard con parámetros: `field_equals:status:active`
   - Guard personalizado registrado: `can_approve_large_amounts`

2. **WHEN** se ejecuta una transición con guards
   **THEN** el sistema SHALL evaluar TODOS los guards en el orden definido
   **AND** TODOS los guards deben pasar (AND lógico) para permitir la transición
   **AND** el sistema SHALL detener la evaluación en el primer guard que falle (short-circuit)

3. **WHEN** un guard falla
   **THEN** el sistema SHALL retornar HTTP 403 Forbidden
   **AND** la respuesta SHALL incluir:
   ```json
   {
     "error": "Guard validation failed",
     "code": "GUARD_FAILED",
     "details": {
       "guard": "is_assigned_to_actor",
       "message": "Instance is not assigned to the current actor"
     }
   }
   ```

4. **WHEN** se configuran múltiples guards
   **THEN** el workflow YAML SHALL soportar:
   ```yaml
   events:
     - name: approve_review
       from: [in_review]
       to: approved
       guards:
         - has_role:revisor
         - field_not_empty:data.review_notes
         - instance_age_less_than:72h
   ```

#### R20.2: Guards Pre-definidos

**User Story**
Como sistema, quiero proveer un conjunto de guards comunes pre-definidos, para facilitar la configuración de workflows sin código custom.

**Acceptance Criteria**

1. **WHEN** el sistema se inicializa
   **THEN** el sistema SHALL registrar los siguientes guards pre-definidos:

   **Guards de Roles:**
   - `has_role:{role}` - Verifica que el actor tenga el rol especificado
   - `has_any_role:{role1,role2}` - Verifica que el actor tenga al menos uno de los roles

   **Guards de Asignación:**
   - `is_assigned_to_actor` - Verifica que `current_actor == actor_id`
   - `is_not_assigned` - Verifica que `current_actor IS NULL`

   **Guards de Campos:**
   - `field_equals:{path}:{value}` - Verifica `data.path == value`
   - `field_not_empty:{path}` - Verifica que `data.path` no esté vacío
   - `field_exists:{path}` - Verifica que `data.path` exista
   - `field_matches:{path}:{regex}` - Verifica que `data.path` cumpla regex

   **Guards de Tiempo:**
   - `instance_age_less_than:{duration}` - Verifica `NOW() - created_at < duration`
   - `instance_age_more_than:{duration}` - Verifica `NOW() - created_at > duration`
   - `before_time:{HH:MM}` - Verifica que la hora actual sea antes de HH:MM
   - `after_time:{HH:MM}` - Verifica que la hora actual sea después de HH:MM
   - `on_weekday` - Verifica que sea día laboral (lun-vie)

   **Guards de Estado:**
   - `substate_equals:{substate}` - Verifica `current_sub_state == substate`
   - `parent_state_equals:{state}` - Verifica estado del padre (para subprocesos)

   **Guards de Datos:**
   - `data_size_less_than:{size_kb}` - Verifica tamaño del JSONB data
   - `has_attachments` - Verifica que existan attachments (si aplica)

2. **WHEN** se usa un guard pre-definido
   **THEN** el sistema SHALL parsear el nombre y parámetros
   **AND** el sistema SHALL ejecutar la validación correspondiente
   **AND** el sistema SHALL retornar mensaje descriptivo en caso de fallo

3. **IF** se usa un guard no registrado
   **THEN** el sistema SHALL fallar la validación del workflow durante carga
   **AND** el error SHALL indicar "Unknown guard: {guard_name}"

#### R20.3: Guards Personalizados con Código

**User Story**
Como desarrollador, quiero registrar guards personalizados vía código, para implementar lógica de negocio específica del dominio.

**Acceptance Criteria**

1. **WHEN** se implementa un guard personalizado
   **THEN** el guard SHALL implementar la interface:
   ```go
   type Guard interface {
       Validate(ctx context.Context, instance *Instance, actor *Actor) error
   }
   ```

2. **WHEN** se registra un guard personalizado
   **THEN** el sistema SHALL proveer función de registro:
   ```go
   RegisterGuard(name string, guard Guard) error
   ```
   **AND** el sistema SHALL validar que el nombre no esté duplicado

3. **WHEN** se usa un guard personalizado en el workflow YAML
   **THEN** el workflow YAML puede referenciarlo por nombre:
   ```yaml
   guards:
     - can_approve_large_amounts
     - custom_business_rule_xyz
   ```

4. **WHEN** un guard personalizado falla
   **THEN** el error retornado SHALL ser descriptivo
   **AND** el error SHALL incluir contexto del guard (nombre, instancia, actor)

#### R20.4: Guards con OR Lógico

**User Story**
Como diseñador de workflows, quiero combinar guards con lógica OR además de AND, para mayor flexibilidad.

**Acceptance Criteria**

1. **WHEN** se requiere lógica OR entre guards
   **THEN** el workflow YAML SHALL soportar sintaxis:
   ```yaml
   guards:
     - has_role:gestionador
     - or:
         - is_assigned_to_actor
         - has_role:supervisor
   ```
   **Significado:** `has_role:gestionador AND (is_assigned_to_actor OR has_role:supervisor)`

2. **WHEN** se evalúa un bloque OR
   **THEN** el sistema SHALL evaluar todos los guards dentro del bloque
   **AND** al menos UNO debe pasar para que el bloque OR pase
   **AND** si TODOS fallan, el bloque OR falla

3. **IF** se anidan múltiples niveles de OR/AND
   **THEN** el sistema SHALL soportar hasta 3 niveles de anidación
   **AND** profundidades mayores SHALL retornar error de validación

---

### R21: Plantillas de Workflows

El sistema debe permitir crear, gestionar y reutilizar plantillas de workflows, facilitando la creación rápida de procesos similares.

#### R21.1: Crear Workflow desde Plantilla

**User Story**
Como administrador, quiero marcar workflows como plantillas reutilizables, para acelerar la creación de procesos similares.

**Acceptance Criteria**

1. **WHEN** se crea un workflow mediante POST /api/v1/workflows
   **THEN** el request PUEDE incluir campo `is_template: true`
   **AND** el sistema SHALL marcar el workflow como plantilla

2. **WHEN** se marca un workflow como plantilla
   **THEN** el sistema SHALL:
   - Agregar campo `is_template BOOLEAN DEFAULT FALSE` a tabla workflows
   - Permitir que el workflow sea clonado múltiples veces
   - NO permitir crear instancias directamente de un template (retornar error)

3. **WHEN** se lista workflows mediante GET /api/v1/workflows
   **THEN** el sistema SHALL soportar filtro `?is_template=true`
   **AND** SHALL retornar solo plantillas
   **AND** el sistema SHALL soportar filtro `?is_template=false` para workflows normales

#### R21.2: Clonar Workflow desde Plantilla

**User Story**
Como administrador, quiero clonar un workflow desde una plantilla, personalizando ciertos parámetros, para crear workflows derivados rápidamente.

**Acceptance Criteria**

1. **WHEN** se clona una plantilla mediante POST /api/v1/workflows/from-template
   **THEN** el request SHALL incluir:
   ```json
   {
     "template_id": "template-workflow-id",
     "new_id": "new-workflow-id",
     "name": "Nuevo Workflow Personalizado",
     "description": "Descripción específica",
     "overrides": {
       "states.filed.timeout": "48h",
       "events.assign_document.guards": ["has_role:admin"]
     }
   }
   ```

2. **WHEN** el sistema clona la plantilla
   **THEN** el sistema SHALL:
   - Copiar toda la configuración del template
   - Aplicar los overrides especificados
   - Generar nuevo UUID si `new_id` no se proporciona
   - Establecer `is_template = false` en el nuevo workflow
   - Establecer `template_id` = ID del template origen
   - Establecer `created_at` = NOW()

3. **WHEN** se aplican overrides
   **THEN** el sistema SHALL soportar notación dot para rutas:
   - `states.{state_id}.{property}`
   - `events.{event_name}.{property}`
   - `webhooks.0.url`

4. **WHEN** la clonación se completa exitosamente
   **THEN** el sistema SHALL retornar HTTP 201 Created
   **AND** la respuesta SHALL incluir el workflow completo clonado

5. **IF** el template_id no existe o no es una plantilla
   **THEN** el sistema SHALL retornar HTTP 404 Not Found

6. **IF** new_id ya existe
   **THEN** el sistema SHALL retornar HTTP 409 Conflict

#### R21.3: Gestionar Plantillas

**User Story**
Como administrador, quiero gestionar plantillas (actualizar, versionar, eliminar), para mantener una biblioteca actualizada.

**Acceptance Criteria**

1. **WHEN** se actualiza una plantilla mediante PUT /api/v1/workflows/:template_id
   **THEN** el sistema SHALL verificar que `is_template = true`
   **AND** el sistema SHALL incrementar versión de la plantilla
   **AND** los workflows derivados existentes NO se actualizan automáticamente

2. **WHEN** se elimina una plantilla mediante DELETE /api/v1/workflows/:template_id
   **THEN** el sistema SHALL verificar que no haya workflows derivados activos
   **OR** el sistema SHALL permitir eliminación con query param `?force=true`
   **AND** el sistema SHALL hacer soft delete (deleted_at)

3. **WHEN** se consulta workflows derivados
   **THEN** el endpoint GET /api/v1/workflows?template_id={id}
   **SHALL** retornar todos los workflows creados desde esa plantilla

4. **IF** se implementa versionado de plantillas
   **THEN** el sistema SHALL mantener versiones históricas de templates
   **AND** permitir clonar desde versión específica: `?template_version=2`

---

### R22: Import/Export de Workflows

El sistema debe permitir exportar e importar workflows en formato YAML/JSON, facilitando portabilidad entre ambientes y backup de configuraciones.

#### R22.1: Exportar Workflow

**User Story**
Como administrador, quiero exportar workflows a archivos YAML/JSON, para backup, compartir entre ambientes, o control de versiones.

**Acceptance Criteria**

1. **WHEN** se exporta un workflow mediante GET /api/v1/workflows/:id/export
   **THEN** el sistema SHALL soportar query parameter `?format=yaml` o `?format=json`
   **AND** el sistema SHALL generar archivo con configuración completa del workflow

2. **WHEN** se genera el archivo de exportación
   **THEN** el archivo SHALL incluir:
   - Versión del esquema de FlowEngine
   - Metadatos del workflow (id, name, description, version, created_at)
   - Configuración completa (states, events, webhooks, etc.)
   - Timestamp de exportación
   - Checksums para validación de integridad

3. **WHEN** se exporta en formato YAML
   **THEN** el sistema SHALL generar archivo válido según spec del workflow YAML
   **AND** el archivo SHALL ser directamente importable

4. **WHEN** se exporta en formato JSON
   **THEN** el sistema SHALL usar formato JSON:API para consistencia
   **AND** el sistema SHALL incluir schema validation information

5. **WHEN** se exportan múltiples workflows
   **THEN** el endpoint POST /api/v1/workflows/export-batch
   **SHALL** aceptar lista de IDs y generar archivo ZIP con todos los workflows

6. **IF** el workflow tiene referencias a recursos externos (webhooks, etc.)
   **THEN** la exportación SHALL incluir advertencias sobre configuraciones específicas del ambiente

#### R22.2: Importar Workflow

**User Story**
Como administrador, quiero importar workflows desde archivos YAML/JSON, para restaurar configuraciones o migrar entre ambientes.

**Acceptance Criteria**

1. **WHEN** se importa un workflow mediante POST /api/v1/workflows/import
   **THEN** el request SHALL aceptar:
   - Archivo YAML/JSON como multipart/form-data
   - O JSON directo en el body

2. **WHEN** se procesa la importación
   **THEN** el sistema SHALL:
   - Validar formato del archivo (YAML/JSON válido)
   - Validar versión del esquema (compatibilidad)
   - Validar integridad (checksum si está presente)
   - Validar configuración del workflow (estados, eventos, etc.)

3. **WHEN** la validación pasa
   **THEN** el sistema SHALL verificar si el workflow ID ya existe:
   - Si NO existe: crear nuevo workflow
   - Si existe: retornar HTTP 409 Conflict

4. **WHEN** se usa query param `?mode=update`
   **THEN** el sistema SHALL actualizar el workflow existente si los IDs coinciden
   **AND** el sistema SHALL incrementar la versión del workflow

5. **WHEN** se usa query param `?mode=force`
   **THEN** el sistema SHALL sobrescribir el workflow existente sin importar versión
   **AND** el sistema SHALL generar advertencia en logs

6. **WHEN** se importa con `?generate_new_id=true`
   **THEN** el sistema SHALL ignorar el ID del archivo y generar uno nuevo
   **AND** el sistema SHALL mantener name y otros metadatos

7. **WHEN** la importación se completa exitosamente
   **THEN** el sistema SHALL retornar HTTP 201 Created (nuevo) o 200 OK (actualizado)
   **AND** la respuesta SHALL incluir el workflow importado completo

8. **IF** la validación falla
   **THEN** el sistema SHALL retornar HTTP 400 Bad Request
   **AND** el mensaje SHALL incluir lista detallada de errores:
   ```json
   {
     "error": "Import validation failed",
     "details": [
       {
         "field": "states.2.timeout",
         "error": "Invalid duration format"
       },
       {
         "field": "events.3.from",
         "error": "State 'invalid_state' does not exist"
       }
     ]
   }
   ```

#### R22.3: Validación de Compatibilidad

**User Story**
Como sistema, quiero validar compatibilidad de versiones al importar workflows, para prevenir importaciones incompatibles.

**Acceptance Criteria**

1. **WHEN** se detecta versión de esquema en el archivo importado
   **THEN** el sistema SHALL verificar compatibilidad con versión actual
   **AND** el sistema SHALL definir matriz de compatibilidad:
   - `1.0` compatible con `1.x`
   - `2.0` requiere migración desde `1.x`

2. **WHEN** la versión es incompatible
   **THEN** el sistema SHALL retornar HTTP 400 Bad Request
   **AND** el mensaje SHALL indicar versión requerida vs versión del archivo
   **AND** el sistema PUEDE sugerir endpoint de migración: `/api/v1/workflows/migrate`

3. **IF** no se especifica versión en el archivo
   **THEN** el sistema SHALL asumir versión más reciente
   **AND** el sistema SHALL registrar advertencia en logs

#### R22.4: Exportación Bulk y Backup

**User Story**
Como administrador, quiero exportar todos los workflows del sistema para backup completo.

**Acceptance Criteria**

1. **WHEN** se exporta todo mediante GET /api/v1/workflows/export-all
   **THEN** el sistema SHALL generar archivo ZIP conteniendo:
   - Todos los workflows en formato YAML
   - Archivo manifest.json con lista de workflows y checksums
   - README con instrucciones de importación

2. **WHEN** se importa un backup completo mediante POST /api/v1/workflows/import-bulk
   **THEN** el sistema SHALL:
   - Extraer archivo ZIP
   - Validar manifest
   - Importar workflows en orden de dependencias (templates primero)
   - Generar reporte de importación

3. **IF** algunos workflows fallan en importación bulk
   **THEN** el sistema SHALL continuar con los demás (no atómico)
   **AND** el reporte final SHALL listar éxitos y fallos

---

### R23: Metadata Extendida en Transiciones

El sistema debe permitir capturar información adicional en transiciones más allá del data JSONB básico, como reason, feedback, y campos configurables por workflow.

#### R23.1: Campos Adicionales en Transiciones

**User Story**
Como diseñador de workflows, quiero capturar información adicional en transiciones (reason, feedback, attachments), configurable por tipo de evento.

**Acceptance Criteria**

1. **WHEN** se actualiza el schema de workflow_transitions
   **THEN** la tabla SHALL incluir campos adicionales:
   - `reason TEXT` (motivo de la transición)
   - `feedback TEXT` (retroalimentación o comentarios)
   - `metadata JSONB` (campos adicionales configurables)

2. **WHEN** se define un evento en workflow YAML
   **THEN** el sistema SHALL soportar configuración de metadata:
   ```yaml
   events:
     - name: reject
       from: [in_review]
       to: in_progress
       metadata_schema:
         required:
           - reason: string
           - feedback: string
         optional:
           - correction_notes: string
           - priority: enum[low,medium,high]
   ```

3. **WHEN** se ejecuta una transición con metadata_schema definido
   **THEN** el sistema SHALL validar que los campos required estén presentes
   **AND** el sistema SHALL validar tipos de datos según schema
   **AND** el sistema SHALL retornar HTTP 422 si la validación falla

4. **WHEN** se envía un evento con metadata mediante POST /api/v1/instances/:id/events
   **THEN** el request PUEDE incluir:
   ```json
   {
     "event": "reject",
     "actor": "user-123",
     "reason": "Documentación incompleta",
     "feedback": "Faltan adjuntos: cédula, comprobante",
     "metadata": {
       "correction_notes": "Contactado al solicitante",
       "priority": "high"
     }
   }
   ```

5. **WHEN** se persiste la transición
   **THEN** el sistema SHALL almacenar:
   - `reason` en columna `reason`
   - `feedback` en columna `feedback`
   - `metadata` (campos adicionales) en columna `metadata` JSONB

#### R23.2: Validación de Metadata Schema

**User Story**
Como sistema, quiero validar metadata de transiciones según schemas configurados, para garantizar calidad de datos.

**Acceptance Criteria**

1. **WHEN** se valida metadata según schema
   **THEN** el sistema SHALL soportar tipos:
   - `string` (con min_length, max_length, pattern)
   - `number` (con min, max)
   - `boolean`
   - `enum` (lista de valores permitidos)
   - `array` (con item_type)
   - `object` (con nested schema)

2. **WHEN** un campo required falta
   **THEN** el sistema SHALL retornar HTTP 422 Unprocessable Entity
   **AND** el error SHALL indicar: "Missing required metadata field: {field}"

3. **WHEN** un valor no cumple el tipo
   **THEN** el sistema SHALL retornar HTTP 422
   **AND** el error SHALL indicar: "Invalid type for field {field}: expected {expected}, got {actual}"

4. **WHEN** un enum recibe valor inválido
   **THEN** el sistema SHALL retornar HTTP 422
   **AND** el error SHALL indicar: "Invalid value for {field}: must be one of {allowed_values}"

#### R23.3: Consultar Transiciones con Metadata

**User Story**
Como usuario, quiero consultar y filtrar transiciones por su metadata, para análisis y reporting.

**Acceptance Criteria**

1. **WHEN** se consulta el historial mediante GET /api/v1/instances/:id/history
   **THEN** cada transición SHALL incluir campos de metadata:
   ```json
   {
     "id": "trans-123",
     "event": "reject",
     "from_state": "in_review",
     "to_state": "in_progress",
     "reason": "Documentación incompleta",
     "feedback": "Faltan adjuntos",
     "metadata": {
       "priority": "high"
     },
     "actor": "user-123",
     "created_at": "2025-11-05T10:00:00Z"
   }
   ```

2. **WHEN** se filtra historial por metadata
   **THEN** el endpoint GET /api/v1/instances/:id/history
   **SHALL** soportar filtros:
   - `?event=reject` - solo transiciones de tipo reject
   - `?has_feedback=true` - solo transiciones con feedback
   - `?metadata.priority=high` - filtrar por campos de metadata

3. **WHEN** se consultan transiciones globalmente
   **THEN** el endpoint POST /api/v1/queries/transitions
   **SHALL** permitir búsqueda avanzada:
   ```json
   {
     "workflow_id": "person_document_flow",
     "events": ["reject", "escalate"],
     "from_date": "2025-11-01",
     "to_date": "2025-11-30",
     "has_feedback": true,
     "metadata_filter": {
       "priority": "high"
     }
   }
   ```

4. **IF** se implementan métricas
   **THEN** el sistema SHALL exponer:
   - `transitions_with_reason_total` (transiciones con reason)
   - `transitions_with_feedback_total` (transiciones con feedback)
   - Agregaciones por valores de metadata (ej: prioridad)

#### R23.4: Reason y Feedback Obligatorios

**User Story**
Como diseñador de workflows, quiero hacer reason y feedback obligatorios para ciertos eventos, para garantizar documentación.

**Acceptance Criteria**

1. **WHEN** se configura un evento en workflow YAML
   **THEN** el sistema SHALL soportar:
   ```yaml
   events:
     - name: reject
       require_reason: true
       require_feedback: true
   ```

2. **WHEN** se ejecuta transición con `require_reason: true`
   **THEN** el sistema SHALL validar que el campo `reason` esté presente y no vacío
   **AND** el sistema SHALL retornar HTTP 422 si falta

3. **WHEN** se ejecuta transición con `require_feedback: true`
   **THEN** el sistema SHALL validar que el campo `feedback` esté presente y no vacío
   **AND** el sistema SHALL retornar HTTP 422 si falta

4. **IF** reason y feedback son opcionales (default)
   **THEN** el sistema SHALL aceptar transiciones sin estos campos
   **AND** los campos quedarán NULL en la base de datos

---

## Priorización de Requerimientos

### Must Have (P0) - Core Functionality
Estos requerimientos son esenciales para el funcionamiento básico del sistema y deben implementarse en la primera versión.

- R1.1, R1.2: Gestión básica de workflows
- R2.1, R2.2: Creación y consulta de instancias
- R3.1, R3.2: Ejecución de transiciones con optimistic locking
- **R17.1, R17.2, R17.3, R17.4**: **Sistema de Subestados** (crítico - ya usado en person_states)
- **R18.1, R18.2, R18.3**: **Escalamientos Manuales básicos** (crítico - ya usado en person_states)
- **R20.1, R20.2**: **Guards básicos y pre-definidos** (crítico para lógica de negocio)
- **R23.1, R23.2**: **Metadata extendida básica** (reason, feedback - ya usado)
- R8.1: Publicación de eventos de dominio
- R10.1: Hybrid repository
- R11.3: Health check
- R13.1: Graceful shutdown
- R16.1, R16.2: Formato JSON:API básico (responses y errores)

### Should Have (P1) - Enhanced Features
Funcionalidades importantes que mejoran significativamente la usabilidad y flexibilidad del sistema.

- R1.3: Actualización de workflows
- R2.3: Historial de transiciones
- R4: Ciclo de vida (pausar, reanudar, cancelar)
- R5: Timers y escalamientos automáticos
- R7: Actores y roles
- R8.2: Webhooks
- R9: Queries avanzadas
- R11.1, R11.2: Observabilidad completa
- R15.1, R15.2, R15.3, R15.4: Clonación básica de instancias
- R16.3, R16.4, R16.5: JSON:API paginación, filtrado y tipos de recursos
- R16.6, R16.7: JSON:API content negotiation y operaciones de escritura
- **R18.4**: **Gestión avanzada de escalamientos** (cerrar/cancelar)
- **R19.1, R19.2**: **Reclasificación de Instancias**
- **R20.3, R20.4**: **Guards personalizados con código y lógica OR**
- **R21.1, R21.2**: **Plantillas de Workflows básicas** (crear y clonar)
- **R23.3, R23.4**: **Consultas y validación avanzada de metadata**

### Could Have (P2) - Nice to Have
Funcionalidades que agregan valor pero no son críticas para el lanzamiento inicial.

- R6: Subprocesos jerárquicos
- R9.2: Estadísticas y métricas
- R12: Seguridad avanzada
- R14: Performance optimizations
- R15.5, R15.6, R15.7, R15.8: Gestión avanzada de clonación (tiempos, notificaciones, configuración, permisos)
- R16.8: JSON:API metadatos globales avanzados
- **R21.3**: **Gestión avanzada de plantillas** (versionado, soft delete)
- **R22.1, R22.2, R22.3, R22.4**: **Import/Export de Workflows completo**

### Won't Have (v1.0) - Future Considerations
Funcionalidades que no se implementarán en la versión 1.0 pero podrían considerarse para versiones futuras.

- UI gráfica para diseñar workflows (drag-and-drop)
- Multi-tenancy con aislamiento de datos
- Workflow versioning con rollback automático de instancias
- Machine learning para optimización de rutas y predicción de tiempos
- Gestión de adjuntos integrada (se espera que cada implementación use su propio object storage)
- Sistema de comentarios/notas integrado (se puede implementar vía webhooks externos)

---

## Historial de Versiones

### Versión 2.0 (2025-11-05)
**Cambios Mayores:**
- **R17**: Agregado sistema de subestados jerárquicos (crítico para person_states)
- **R18**: Agregado sistema de escalamientos manuales completo
- **R19**: Agregado sistema de reclasificación de instancias
- **R20**: Agregado sistema de guards avanzados con lógica compleja
- **R21**: Agregado sistema de plantillas de workflows
- **R22**: Agregado sistema de import/export de workflows
- **R23**: Agregado sistema de metadata extendida en transiciones
- **R3.1**: Clarificado soporte explícito de reentrada de estados (loops)
- Actualizada sección de priorización con nuevos requerimientos clasificados por criticidad

**Motivación:** Estos requerimientos reflejan funcionalidades ya implementadas en el sistema de estados de personas (person_states) y son críticos para garantizar que el sistema sea suficientemente genérico para implementar múltiples tipos de flujos de trabajo.

### Versión 1.0 (2025-01-15)
- Versión inicial con 16 requerimientos funcionales básicos
- Cobertura de workflows, instancias, persistencia, seguridad y observabilidad

---

**Versión Actual**: 2.0
**Fecha de Última Actualización**: 2025-11-05
**Próxima Revisión**: 2025-12-05
