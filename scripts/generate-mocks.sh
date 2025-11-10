#!/bin/bash

# Script para generar mocks con mockery
# Genera mocks para todas las interfaces en el proyecto

set -e

echo "Generando mocks para FlowEngine..."

# Domain repositories
echo "Generando mocks de repositories..."
mockery --dir=internal/domain/workflow --name=Repository --output=internal/domain/workflow/mocks --outpkg=mocks
mockery --dir=internal/domain/instance --name=Repository --output=internal/domain/instance/mocks --outpkg=mocks

# Event dispatcher
echo "Generando mocks de event dispatcher..."
mockery --dir=internal/domain/event --name=Dispatcher --output=internal/domain/event/mocks --outpkg=mocks

# Application use cases (si se necesitan mocks)
# mockery --dir=internal/application/workflow --name=CreateWorkflowUseCase --output=internal/application/workflow/mocks --outpkg=mocks

echo "✓ Mocks generados exitosamente"
