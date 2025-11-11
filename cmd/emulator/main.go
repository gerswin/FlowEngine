package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
)

// Simple workflow emulator CLI
type WorkflowEmulator struct {
	workflows  map[string]*workflow.Workflow
	instances  map[string]*instance.Instance
	dispatcher *event.InMemoryDispatcher
	systemActor shared.ID
	scanner    *bufio.Scanner
}

func NewWorkflowEmulator() *WorkflowEmulator {
	return &WorkflowEmulator{
		workflows:  make(map[string]*workflow.Workflow),
		instances:  make(map[string]*instance.Instance),
		dispatcher: event.NewInMemoryDispatcher(),
		systemActor: shared.NewID(),
		scanner:    bufio.NewScanner(os.Stdin),
	}
}

func (we *WorkflowEmulator) Run() {
	we.printWelcome()

	for {
		fmt.Print("\n> ")
		if !we.scanner.Scan() {
			break
		}

		input := strings.TrimSpace(we.scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]

		switch command {
		case "help", "h":
			we.printHelp()
		case "create-workflow", "cw":
			we.createWorkflow()
		case "list-workflows", "lw":
			we.listWorkflows()
		case "show-workflow", "sw":
			if len(parts) < 2 {
				fmt.Println("❌ Usage: show-workflow <name>")
				continue
			}
			we.showWorkflow(parts[1])
		case "create-instance", "ci":
			if len(parts) < 2 {
				fmt.Println("❌ Usage: create-instance <workflow-name>")
				continue
			}
			we.createInstance(parts[1])
		case "list-instances", "li":
			we.listInstances()
		case "show-instance", "si":
			if len(parts) < 2 {
				fmt.Println("❌ Usage: show-instance <id>")
				continue
			}
			we.showInstance(parts[1])
		case "transition", "t":
			if len(parts) < 3 {
				fmt.Println("❌ Usage: transition <instance-id> <event>")
				continue
			}
			we.executeTransition(parts[1], parts[2])
		case "pause":
			if len(parts) < 2 {
				fmt.Println("❌ Usage: pause <instance-id>")
				continue
			}
			we.pauseInstance(parts[1])
		case "resume":
			if len(parts) < 2 {
				fmt.Println("❌ Usage: resume <instance-id>")
				continue
			}
			we.resumeInstance(parts[1])
		case "complete":
			if len(parts) < 2 {
				fmt.Println("❌ Usage: complete <instance-id>")
				continue
			}
			we.completeInstance(parts[1])
		case "cancel":
			if len(parts) < 2 {
				fmt.Println("❌ Usage: cancel <instance-id>")
				continue
			}
			we.cancelInstance(parts[1])
		case "history":
			if len(parts) < 2 {
				fmt.Println("❌ Usage: history <instance-id>")
				continue
			}
			we.showHistory(parts[1])
		case "events":
			we.showEvents()
		case "clear":
			fmt.Print("\033[H\033[2J")
		case "exit", "quit", "q":
			fmt.Println("\n👋 ¡Hasta luego!")
			return
		default:
			fmt.Printf("❌ Comando desconocido: %s (usa 'help' para ver comandos)\n", command)
		}
	}
}

func (we *WorkflowEmulator) printWelcome() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                            ║")
	fmt.Println("║        🎮 FlowEngine Interactive Emulator 🎮              ║")
	fmt.Println("║                                                            ║")
	fmt.Println("║  Create workflows, run instances, and test transitions    ║")
	fmt.Println("║  Type 'help' to see available commands                    ║")
	fmt.Println("║                                                            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
}

func (we *WorkflowEmulator) printHelp() {
	fmt.Println("\n📚 Available Commands:")
	fmt.Println("\n  Workflow Management:")
	fmt.Println("    create-workflow (cw)           - Create a new workflow interactively")
	fmt.Println("    list-workflows (lw)            - List all workflows")
	fmt.Println("    show-workflow <name> (sw)      - Show workflow details")
	fmt.Println("\n  Instance Management:")
	fmt.Println("    create-instance <workflow> (ci) - Create a new instance")
	fmt.Println("    list-instances (li)            - List all instances")
	fmt.Println("    show-instance <id> (si)        - Show instance details")
	fmt.Println("\n  Instance Operations:")
	fmt.Println("    transition <id> <event> (t)    - Execute a transition")
	fmt.Println("    pause <id>                     - Pause an instance")
	fmt.Println("    resume <id>                    - Resume a paused instance")
	fmt.Println("    complete <id>                  - Complete an instance")
	fmt.Println("    cancel <id>                    - Cancel an instance")
	fmt.Println("    history <id>                   - Show transition history")
	fmt.Println("\n  Events & Utilities:")
	fmt.Println("    events                         - Show all domain events")
	fmt.Println("    clear                          - Clear screen")
	fmt.Println("    help (h)                       - Show this help")
	fmt.Println("    exit (quit, q)                 - Exit emulator")
	fmt.Println()
}

func (we *WorkflowEmulator) createWorkflow() {
	fmt.Println("\n📋 Creating New Workflow")

	// Get workflow name
	fmt.Print("  Workflow name: ")
	we.scanner.Scan()
	name := strings.TrimSpace(we.scanner.Text())
	if name == "" {
		fmt.Println("❌ Name cannot be empty")
		return
	}

	// Get initial state
	fmt.Print("  Initial state ID: ")
	we.scanner.Scan()
	initialStateID := strings.TrimSpace(we.scanner.Text())

	fmt.Print("  Initial state name: ")
	we.scanner.Scan()
	initialStateName := strings.TrimSpace(we.scanner.Text())

	initialState, err := workflow.NewState(initialStateID, initialStateName)
	if err != nil {
		fmt.Printf("❌ Error creating state: %v\n", err)
		return
	}

	// Create workflow
	wf, err := workflow.NewWorkflow(name, initialState, we.systemActor)
	if err != nil {
		fmt.Printf("❌ Error creating workflow: %v\n", err)
		return
	}

	// Track events
	we.dispatcher.DispatchBatch(nil, wf.DomainEvents())

	// Add more states
	fmt.Println("\n  Add additional states (enter empty ID to finish):")
	for {
		fmt.Print("    State ID: ")
		we.scanner.Scan()
		stateID := strings.TrimSpace(we.scanner.Text())
		if stateID == "" {
			break
		}

		fmt.Print("    State name: ")
		we.scanner.Scan()
		stateName := strings.TrimSpace(we.scanner.Text())

		fmt.Print("    Is final state? (y/n): ")
		we.scanner.Scan()
		isFinal := strings.ToLower(strings.TrimSpace(we.scanner.Text())) == "y"

		state, err := workflow.NewState(stateID, stateName)
		if err != nil {
			fmt.Printf("    ❌ Error: %v\n", err)
			continue
		}

		if isFinal {
			state = state.AsFinal()
		}

		if err := wf.AddState(state); err != nil {
			fmt.Printf("    ❌ Error adding state: %v\n", err)
			continue
		}

		fmt.Printf("    ✅ Added state: %s\n", stateID)
	}

	// Add events (transitions)
	fmt.Println("\n  Add events/transitions (enter empty event name to finish):")
	for {
		fmt.Print("    Event name: ")
		we.scanner.Scan()
		eventName := strings.TrimSpace(we.scanner.Text())
		if eventName == "" {
			break
		}

		fmt.Print("    From state ID: ")
		we.scanner.Scan()
		fromStateID := strings.TrimSpace(we.scanner.Text())

		fmt.Print("    To state ID: ")
		we.scanner.Scan()
		toStateID := strings.TrimSpace(we.scanner.Text())

		fromState, err := wf.GetState(fromStateID)
		if err != nil {
			fmt.Printf("    ❌ From state not found: %v\n", err)
			continue
		}

		toState, err := wf.GetState(toStateID)
		if err != nil {
			fmt.Printf("    ❌ To state not found: %v\n", err)
			continue
		}

		event, err := workflow.NewEvent(eventName, []workflow.State{fromState}, toState)
		if err != nil {
			fmt.Printf("    ❌ Error creating event: %v\n", err)
			continue
		}

		if err := wf.AddEvent(event); err != nil {
			fmt.Printf("    ❌ Error adding event: %v\n", err)
			continue
		}

		fmt.Printf("    ✅ Added event: %s (%s → %s)\n", eventName, fromStateID, toStateID)
	}

	we.workflows[name] = wf
	fmt.Printf("\n✅ Workflow '%s' created successfully! [ID: %s]\n", name, wf.ID().String()[:8])
}

func (we *WorkflowEmulator) listWorkflows() {
	if len(we.workflows) == 0 {
		fmt.Println("\n📋 No workflows created yet. Use 'create-workflow' to create one.")
		return
	}

	fmt.Println("\n📋 Available Workflows:")
	for name, wf := range we.workflows {
		fmt.Printf("  • %s [ID: %s, States: %d, Events: %d]\n",
			name,
			wf.ID().String()[:8],
			len(wf.States()),
			len(wf.Events()))
	}
}

func (we *WorkflowEmulator) showWorkflow(name string) {
	wf, exists := we.workflows[name]
	if !exists {
		fmt.Printf("❌ Workflow '%s' not found\n", name)
		return
	}

	fmt.Printf("\n📋 Workflow: %s\n", wf.Name())
	fmt.Printf("  ID: %s\n", wf.ID().String())
	fmt.Printf("  Version: %s\n", wf.Version().String())
	fmt.Printf("  Initial State: %s\n", wf.InitialState().ID())

	fmt.Println("\n  States:")
	for _, state := range wf.States() {
		marker := ""
		if state.IsFinal() {
			marker = " [FINAL]"
		}
		fmt.Printf("    • %s - %s%s\n", state.ID(), state.Name(), marker)
	}

	fmt.Println("\n  Events:")
	for _, event := range wf.Events() {
		sources := event.Sources()
		sourceIDs := make([]string, len(sources))
		for i, s := range sources {
			sourceIDs[i] = s.ID()
		}
		fmt.Printf("    • %s: %v → %s\n",
			event.Name(),
			sourceIDs,
			event.Destination().ID())
	}
}

func (we *WorkflowEmulator) createInstance(workflowName string) {
	wf, exists := we.workflows[workflowName]
	if !exists {
		fmt.Printf("❌ Workflow '%s' not found\n", workflowName)
		return
	}

	inst, err := instance.NewInstance(
		wf.ID(),
		wf.Name(),
		wf.InitialState().ID(),
		we.systemActor,
	)
	if err != nil {
		fmt.Printf("❌ Error creating instance: %v\n", err)
		return
	}

	// Track events
	we.dispatcher.DispatchBatch(nil, inst.DomainEvents())

	shortID := inst.ID().String()[:8]
	we.instances[shortID] = inst

	fmt.Printf("\n✅ Instance created: %s\n", shortID)
	fmt.Printf("  Workflow: %s\n", inst.WorkflowName())
	fmt.Printf("  State: %s\n", inst.CurrentState())
	fmt.Printf("  Status: %s\n", inst.Status())
}

func (we *WorkflowEmulator) listInstances() {
	if len(we.instances) == 0 {
		fmt.Println("\n🎬 No instances created yet. Use 'create-instance <workflow>' to create one.")
		return
	}

	fmt.Println("\n🎬 Active Instances:")
	for id, inst := range we.instances {
		fmt.Printf("  • %s: %s [State: %s, Status: %s]\n",
			id,
			inst.WorkflowName(),
			inst.CurrentState(),
			inst.Status())
	}
}

func (we *WorkflowEmulator) showInstance(id string) {
	inst, exists := we.instances[id]
	if !exists {
		fmt.Printf("❌ Instance '%s' not found\n", id)
		return
	}

	fmt.Printf("\n🎬 Instance: %s\n", id)
	fmt.Printf("  Workflow: %s [%s]\n", inst.WorkflowName(), inst.WorkflowID().String()[:8])
	fmt.Printf("  Current State: %s\n", inst.CurrentState())
	if inst.HasSubState() {
		fmt.Printf("  Sub-State: %s\n", inst.CurrentSubState().ID())
	}
	fmt.Printf("  Status: %s\n", inst.Status())
	fmt.Printf("  Version: %s\n", inst.Version().String())
	fmt.Printf("  Transitions: %d\n", inst.TransitionCount())
	fmt.Printf("  Created: %s\n", inst.CreatedAt().Time().Format("2006-01-02 15:04:05"))
	if !inst.CompletedAt().IsZero() {
		fmt.Printf("  Completed: %s\n", inst.CompletedAt().Time().Format("2006-01-02 15:04:05"))
	}
}

func (we *WorkflowEmulator) executeTransition(id, eventName string) {
	inst, exists := we.instances[id]
	if !exists {
		fmt.Printf("❌ Instance '%s' not found\n", id)
		return
	}

	// Find workflow
	var wf *workflow.Workflow
	for _, w := range we.workflows {
		if w.ID().Equals(inst.WorkflowID()) {
			wf = w
			break
		}
	}

	if wf == nil {
		fmt.Println("❌ Workflow not found")
		return
	}

	// Find event
	evt, err := wf.FindEvent(eventName)
	if err != nil {
		fmt.Printf("❌ Event '%s' not found in workflow\n", eventName)
		return
	}

	// Execute transition
	metadata := instance.NewTransitionMetadataWithReason(fmt.Sprintf("Manual transition: %s", eventName))
	err = inst.Transition(evt.Destination().ID(), eventName, we.systemActor, metadata)
	if err != nil {
		fmt.Printf("❌ Transition failed: %v\n", err)
		return
	}

	// Track events
	we.dispatcher.DispatchBatch(nil, inst.DomainEvents())

	fmt.Printf("✅ Transition executed: %s\n", eventName)
	fmt.Printf("  New State: %s\n", inst.CurrentState())
	fmt.Printf("  Status: %s\n", inst.Status())
	fmt.Printf("  Version: %s\n", inst.Version().String())
}

func (we *WorkflowEmulator) pauseInstance(id string) {
	inst, exists := we.instances[id]
	if !exists {
		fmt.Printf("❌ Instance '%s' not found\n", id)
		return
	}

	err := inst.Pause(we.systemActor, "Manual pause from emulator")
	if err != nil {
		fmt.Printf("❌ Pause failed: %v\n", err)
		return
	}

	we.dispatcher.DispatchBatch(nil, inst.DomainEvents())
	fmt.Printf("✅ Instance paused: %s\n", id)
}

func (we *WorkflowEmulator) resumeInstance(id string) {
	inst, exists := we.instances[id]
	if !exists {
		fmt.Printf("❌ Instance '%s' not found\n", id)
		return
	}

	err := inst.Resume(we.systemActor)
	if err != nil {
		fmt.Printf("❌ Resume failed: %v\n", err)
		return
	}

	we.dispatcher.DispatchBatch(nil, inst.DomainEvents())
	fmt.Printf("✅ Instance resumed: %s\n", id)
}

func (we *WorkflowEmulator) completeInstance(id string) {
	inst, exists := we.instances[id]
	if !exists {
		fmt.Printf("❌ Instance '%s' not found\n", id)
		return
	}

	err := inst.Complete(we.systemActor)
	if err != nil {
		fmt.Printf("❌ Complete failed: %v\n", err)
		return
	}

	we.dispatcher.DispatchBatch(nil, inst.DomainEvents())
	fmt.Printf("✅ Instance completed: %s\n", id)
}

func (we *WorkflowEmulator) cancelInstance(id string) {
	inst, exists := we.instances[id]
	if !exists {
		fmt.Printf("❌ Instance '%s' not found\n", id)
		return
	}

	err := inst.Cancel(we.systemActor, "Manual cancellation from emulator")
	if err != nil {
		fmt.Printf("❌ Cancel failed: %v\n", err)
		return
	}

	we.dispatcher.DispatchBatch(nil, inst.DomainEvents())
	fmt.Printf("✅ Instance canceled: %s\n", id)
}

func (we *WorkflowEmulator) showHistory(id string) {
	inst, exists := we.instances[id]
	if !exists {
		fmt.Printf("❌ Instance '%s' not found\n", id)
		return
	}

	transitions := inst.GetTransitionHistory()
	if len(transitions) == 0 {
		fmt.Println("\n📜 No transitions yet")
		return
	}

	fmt.Printf("\n📜 Transition History for %s (%d transitions):\n", id, len(transitions))
	for i, trans := range transitions {
		fmt.Printf("\n  [%d] %s\n", i+1, trans.Event())
		fmt.Printf("      %s → %s\n", trans.From(), trans.To())
		fmt.Printf("      Time: %s\n", trans.Timestamp().Time().Format("15:04:05"))

		metadata := trans.Metadata()
		if !metadata.IsEmpty() && metadata.Reason() != "" {
			fmt.Printf("      Reason: %s\n", metadata.Reason())
		}
	}
}

func (we *WorkflowEmulator) showEvents() {
	events := we.dispatcher.Events()
	if len(events) == 0 {
		fmt.Println("\n📊 No events recorded yet")
		return
	}

	fmt.Printf("\n📊 Domain Events (%d total):\n", len(events))

	// Group by type
	byType := make(map[string]int)
	for _, evt := range events {
		byType[evt.Type()]++
	}

	for eventType, count := range byType {
		fmt.Printf("  • %-30s: %d\n", eventType, count)
	}
}

func main() {
	emulator := NewWorkflowEmulator()
	emulator.Run()
}
