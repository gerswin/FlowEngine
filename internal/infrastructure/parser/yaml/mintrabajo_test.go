package yaml

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseMintrabajoWorkflow(t *testing.T) {
	// Find the project root and the mintrabajo workflow file
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Navigate up to find the config/templates directory
	projectRoot := wd
	for i := 0; i < 5; i++ {
		yamlPath := filepath.Join(projectRoot, "config", "templates", "mintrabajo_flow.yaml")
		if _, err := os.Stat(yamlPath); err == nil {
			// Found the file
			parser := NewParser()

			// Register custom validators used in mintrabajo
			parser.RegisterGuard("validate_tiempo_clonacion_menor_a_limite")
			parser.RegisterGuard("validate_tiempo_escalamiento_menor_a_limite")
			parser.RegisterGuard("validate_rechazos_clonacion_menor_a_3")
			parser.RegisterGuard("validate_rechazos_escalamiento_menor_a_3")

			wf, err := parser.ParseFile(yamlPath)

			require.NoError(t, err, "Should parse mintrabajo workflow without errors")
			assert.NotNil(t, wf)
			assert.Equal(t, "Flujo Ministerio del Trabajo", wf.Name())
			assert.Equal(t, "radicado", wf.InitialState().ID)

			// Verify states
			states := wf.States()
			t.Logf("Total states: %d", len(states))

			// Main states
			expectedStates := []string{
				"radicado",
				"por_asignar",
				"en_asignacion",
				"para_gestion",
				"en_edicion",
				"por_revisar",
				"revision_rechazada",
				"revision_aprobada",
				"por_aprobar",
				"aprobado",
				"reclasificado",
				"clonacion_asignada",
				"clonacion_respondida_final",
				"clonacion_cancelada",
				"escalamiento_asignado",
				"escalamiento_respondido_final",
				"escalamiento_cancelado",
			}

			for _, stateID := range expectedStates {
				found := false
				for _, s := range states {
					if s.ID == stateID {
						found = true
						break
					}
				}
				assert.True(t, found, "State %s should exist", stateID)
			}

			// Verify events
			events := wf.Events()
			t.Logf("Total events: %d", len(events))

			// Key events
			expectedEvents := []string{
				"radicar_tramite",
				"enviar_a_asignador",
				"asignar_abogado",
				"reclasificar_a_entes_control",
				"reclasificar_a_pqrd",
				"iniciar_gestion",
				"solicitar_clonacion",
				"solicitar_escalamiento",
				"enviar_a_revision",
				"aprobar_revision",
				"rechazar_revision",
				"aprobar_tramite",
			}

			for _, eventName := range expectedEvents {
				found := false
				for _, e := range events {
					if e.Name == eventName {
						found = true
						break
					}
				}
				assert.True(t, found, "Event %s should exist", eventName)
			}

			t.Logf("✅ Successfully parsed mintrabajo workflow with %d states and %d events", len(states), len(events))
			return
		}
		projectRoot = filepath.Dir(projectRoot)
	}

	t.Skip("Could not find mintrabajo_flow.yaml file")
}

func TestParser_MintrabajoStatesHaveCorrectProperties(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	projectRoot := wd
	for i := 0; i < 5; i++ {
		yamlPath := filepath.Join(projectRoot, "config", "templates", "mintrabajo_flow.yaml")
		if _, err := os.Stat(yamlPath); err == nil {
			parser := NewParser()
			parser.RegisterGuard("validate_tiempo_clonacion_menor_a_limite")
			parser.RegisterGuard("validate_tiempo_escalamiento_menor_a_limite")
			parser.RegisterGuard("validate_rechazos_clonacion_menor_a_3")
			parser.RegisterGuard("validate_rechazos_escalamiento_menor_a_3")

			wf, err := parser.ParseFile(yamlPath)
			require.NoError(t, err)

			// Check final states
			finalStates := []string{"aprobado", "reclasificado", "clonacion_respondida_final", "clonacion_cancelada", "escalamiento_respondido_final", "escalamiento_cancelado"}
			for _, stateID := range finalStates {
				state, err := wf.GetState(stateID)
				require.NoError(t, err, "State %s should exist", stateID)
				assert.True(t, state.IsFinal, "State %s should be final", stateID)
			}

			// Check states with timeouts
			statesWithTimeout := []string{"por_asignar", "en_asignacion", "para_gestion", "por_revisar", "por_aprobar"}
			for _, stateID := range statesWithTimeout {
				state, err := wf.GetState(stateID)
				require.NoError(t, err, "State %s should exist", stateID)
				assert.Greater(t, state.Timeout.Seconds(), float64(0), "State %s should have timeout", stateID)
				assert.NotEmpty(t, state.OnTimeout, "State %s should have on_timeout event", stateID)
			}

			return
		}
		projectRoot = filepath.Dir(projectRoot)
	}

	t.Skip("Could not find mintrabajo_flow.yaml file")
}
