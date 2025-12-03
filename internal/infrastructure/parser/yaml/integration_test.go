package yaml

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseRealWorkflowFile(t *testing.T) {
	// Find the project root and the real workflow file
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Navigate up to find the config/templates directory
	projectRoot := wd
	for i := 0; i < 5; i++ {
		yamlPath := filepath.Join(projectRoot, "config", "templates", "person_document_flow.yaml")
		if _, err := os.Stat(yamlPath); err == nil {
			// Found the file
			parser := NewParser()
			wf, err := parser.ParseFile(yamlPath)

			require.NoError(t, err)
			assert.NotNil(t, wf)
			assert.Equal(t, "Flujo de Radicación de Documentos (Personas)", wf.Name())
			assert.Equal(t, "filed", wf.InitialState().ID)

			// Verify we have the expected states
			states := wf.States()
			assert.GreaterOrEqual(t, len(states), 6, "Should have at least 6 states")

			// Verify we have the expected events
			events := wf.Events()
			assert.GreaterOrEqual(t, len(events), 10, "Should have at least 10 events")

			t.Logf("Successfully parsed workflow with %d states and %d events", len(states), len(events))
			return
		}
		projectRoot = filepath.Dir(projectRoot)
	}

	t.Skip("Could not find person_document_flow.yaml file")
}
