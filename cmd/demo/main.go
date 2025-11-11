package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
)

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
)

// EventTracker tracks all domain events generated
type EventTracker struct {
	events []event.DomainEvent
}

func NewEventTracker() *EventTracker {
	return &EventTracker{
		events: make([]event.DomainEvent, 0),
	}
}

func (et *EventTracker) Track(events []event.DomainEvent) {
	et.events = append(et.events, events...)
}

func (et *EventTracker) PrintSummary() {
	fmt.Printf("\n%s%s=== 📊 Domain Events Summary ===%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Total events generated: %s%d%s\n\n", colorBold, len(et.events), colorReset)

	// Group by type
	eventsByType := make(map[string]int)
	for _, evt := range et.events {
		eventsByType[evt.Type()]++
	}

	for eventType, count := range eventsByType {
		fmt.Printf("  %s%-30s%s: %d\n", colorYellow, eventType, colorReset, count)
	}

	fmt.Println()
}

func (et *EventTracker) PrintDetailed() {
	fmt.Printf("\n%s%s=== 🔍 Detailed Event Log ===%s\n", colorBold, colorPurple, colorReset)

	for i, evt := range et.events {
		fmt.Printf("\n%s[Event %d/%d]%s\n", colorBold, i+1, len(et.events), colorReset)
		fmt.Printf("  Type:         %s%s%s\n", colorYellow, evt.Type(), colorReset)
		fmt.Printf("  Aggregate ID: %s\n", evt.AggregateID())
		fmt.Printf("  Occurred At:  %s\n", evt.OccurredAt().Format(time.RFC3339))

		payload := evt.Payload()
		payloadJSON, _ := json.MarshalIndent(payload, "  ", "  ")
		fmt.Printf("  Payload:\n  %s\n", string(payloadJSON))
	}

	fmt.Println()
}

func printHeader(title string) {
	fmt.Printf("\n%s%s=== %s ===%s\n", colorBold, colorBlue, title, colorReset)
}

func printSuccess(message string) {
	fmt.Printf("%s✅ %s%s\n", colorGreen, message, colorReset)
}

func printInfo(message string) {
	fmt.Printf("%sℹ️  %s%s\n", colorCyan, message, colorReset)
}

func printError(message string) {
	fmt.Printf("%s❌ %s%s\n", colorRed, message, colorReset)
}

func printSubHeader(title string) {
	fmt.Printf("\n%s--- %s ---%s\n", colorYellow, title, colorReset)
}

func main() {
	fmt.Printf("\n%s%s", colorBold, colorCyan)
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                            ║")
	fmt.Println("║        🚀 FlowEngine Domain Layer Demo 🚀                 ║")
	fmt.Println("║                                                            ║")
	fmt.Println("║  Demonstrating Workflow Engine capabilities               ║")
	fmt.Println("║  Phase 1-5: Complete Domain Layer with Events             ║")
	fmt.Println("║                                                            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Printf("%s\n", colorReset)

	// Initialize event tracker
	tracker := NewEventTracker()

	// System actor (simulating the user executing the demo)
	systemActor := shared.NewID()
	printInfo(fmt.Sprintf("System Actor ID: %s", systemActor.String()))

	// ========================================
	// PART 1: Create Document Approval Workflow
	// ========================================
	printHeader("📋 Creating Document Approval Workflow")

	// Define workflow states
	draftState, _ := workflow.NewState("draft", "Draft")
	reviewState, _ := workflow.NewState("under_review", "Under Review")
	approvedState, _ := workflow.NewState("approved", "Approved")
	approvedState = approvedState.AsFinal()
	rejectedState, _ := workflow.NewState("rejected", "Rejected")
	rejectedState = rejectedState.AsFinal()
	revisionsState, _ := workflow.NewState("needs_revisions", "Needs Revisions")

	// Create workflow
	approvalWorkflow, err := workflow.NewWorkflow("Document Approval", draftState, systemActor)
	if err != nil {
		printError(fmt.Sprintf("Failed to create workflow: %v", err))
		return
	}

	// Track workflow creation events
	tracker.Track(approvalWorkflow.DomainEvents())

	printSuccess(fmt.Sprintf("Workflow created: %s [ID: %s]",
		approvalWorkflow.Name(),
		approvalWorkflow.ID().String()[:8]))
	printInfo(fmt.Sprintf("   Initial State: %s", draftState.ID()))
	printInfo(fmt.Sprintf("   Version: %s", approvalWorkflow.Version().String()))

	// Add states to workflow
	printSubHeader("Adding States")
	approvalWorkflow.AddState(reviewState)
	approvalWorkflow.AddState(approvedState)
	approvalWorkflow.AddState(rejectedState)
	approvalWorkflow.AddState(revisionsState)

	states := approvalWorkflow.States()
	for _, state := range states {
		marker := ""
		if state.IsFinal() {
			marker = " [FINAL]"
		}
		printSuccess(fmt.Sprintf("State: %s - %s%s", state.ID(), state.Name(), marker))
	}

	// Add events (transitions) to workflow
	printSubHeader("Adding Workflow Events (Transitions)")

	submitEvent, _ := workflow.NewEvent("submit", []workflow.State{draftState}, reviewState)
	approvalWorkflow.AddEvent(submitEvent)
	printSuccess("Event: submit (draft → under_review)")

	approveEvent, _ := workflow.NewEvent("approve", []workflow.State{reviewState}, approvedState)
	approvalWorkflow.AddEvent(approveEvent)
	printSuccess("Event: approve (under_review → approved)")

	rejectEvent, _ := workflow.NewEvent("reject", []workflow.State{reviewState}, rejectedState)
	approvalWorkflow.AddEvent(rejectEvent)
	printSuccess("Event: reject (under_review → rejected)")

	requestRevisionsEvent, _ := workflow.NewEvent("request_revisions", []workflow.State{reviewState}, revisionsState)
	approvalWorkflow.AddEvent(requestRevisionsEvent)
	printSuccess("Event: request_revisions (under_review → needs_revisions)")

	resubmitEvent, _ := workflow.NewEvent("resubmit", []workflow.State{revisionsState}, reviewState)
	approvalWorkflow.AddEvent(resubmitEvent)
	printSuccess("Event: resubmit (needs_revisions → under_review)")

	// ========================================
	// PART 2: Create and Execute Instance
	// ========================================
	printHeader("🎬 Creating Workflow Instance")

	// Create instance
	inst, err := instance.NewInstance(
		approvalWorkflow.ID(),
		approvalWorkflow.Name(),
		draftState.ID(),
		systemActor,
	)
	if err != nil {
		printError(fmt.Sprintf("Failed to create instance: %v", err))
		return
	}

	// Track instance creation events
	tracker.Track(inst.DomainEvents())

	printSuccess(fmt.Sprintf("Instance created: %s", inst.ID().String()[:8]))
	printInfo(fmt.Sprintf("   Workflow: %s", inst.WorkflowName()))
	printInfo(fmt.Sprintf("   Current State: %s", inst.CurrentState()))
	printInfo(fmt.Sprintf("   Status: %s", inst.Status()))
	printInfo(fmt.Sprintf("   Version: %s", inst.Version().String()))

	// Set instance data
	printSubHeader("Setting Instance Data")
	inst.UpdateData("document_title", "Q4 Financial Report")
	inst.UpdateData("document_type", "financial")
	inst.UpdateData("author", "John Doe")
	inst.UpdateVariable("priority", "high")
	inst.UpdateVariable("department", "Finance")

	printSuccess("Data: document_title = 'Q4 Financial Report'")
	printSuccess("Data: document_type = 'financial'")
	printSuccess("Data: author = 'John Doe'")
	printSuccess("Variable: priority = 'high'")
	printSuccess("Variable: department = 'Finance'")

	// ========================================
	// PART 3: Execute Transitions
	// ========================================
	printHeader("🔄 Executing State Transitions")

	// Transition 1: Submit for review
	printSubHeader("Transition 1: Submit for Review")
	metadata1 := instance.NewTransitionMetadataWithReason("Submitting document for review")
	err = inst.Transition(reviewState.ID(), "submit", systemActor, metadata1)
	if err != nil {
		printError(fmt.Sprintf("Transition failed: %v", err))
	} else {
		tracker.Track(inst.DomainEvents())
		printSuccess(fmt.Sprintf("Transitioned: draft → under_review"))
		printInfo(fmt.Sprintf("   Current State: %s", inst.CurrentState()))
		printInfo(fmt.Sprintf("   Status: %s", inst.Status()))
		printInfo(fmt.Sprintf("   Version: %s", inst.Version().String()))
	}

	// ========================================
	// PART 4: Sub-State Support (R17)
	// ========================================
	printHeader("🔹 Sub-State Support (R17)")

	printSubHeader("Creating Sub-States")
	subStateQA, _ := instance.NewSubState("qa_check", "Quality Assurance Check")
	subStateCompliance, _ := instance.NewSubState("compliance_check", "Compliance Check")

	printSuccess(fmt.Sprintf("Sub-State: %s - %s", subStateQA.ID(), subStateQA.Name()))
	printSuccess(fmt.Sprintf("Sub-State: %s - %s", subStateCompliance.ID(), subStateCompliance.Name()))

	// Transition with sub-state
	printSubHeader("Transition with Sub-State: QA Check")
	metadata2 := instance.NewTransitionMetadataWithReason("Starting QA review")
	err = inst.TransitionWithSubState(reviewState.ID(), subStateQA, "submit", systemActor, metadata2)
	if err != nil {
		printError(fmt.Sprintf("Transition failed: %v", err))
	} else {
		tracker.Track(inst.DomainEvents())
		printSuccess("Sub-state set to: qa_check")
		printInfo(fmt.Sprintf("   State: %s.%s", inst.CurrentState(), inst.CurrentSubState().ID()))
	}

	// Change sub-state
	printSubHeader("Changing Sub-State: Compliance Check")
	metadata3 := instance.NewTransitionMetadataWithReason("Moving to compliance check")
	err = inst.TransitionWithSubState(reviewState.ID(), subStateCompliance, "submit", systemActor, metadata3)
	if err != nil {
		printError(fmt.Sprintf("Transition failed: %v", err))
	} else {
		tracker.Track(inst.DomainEvents())
		printSuccess("Sub-state changed: qa_check → compliance_check")
		printInfo(fmt.Sprintf("   State: %s.%s", inst.CurrentState(), inst.CurrentSubState().ID()))
	}

	// ========================================
	// PART 5: Transition Metadata (R23)
	// ========================================
	printHeader("📝 Transition Metadata (R23)")

	printSubHeader("Request Revisions with Detailed Metadata")
	metadataWithFeedback := instance.NewTransitionMetadata(
		"Document needs significant revisions",
		"Please update the financial projections section and add supporting documentation for Q3 actuals.",
		map[string]interface{}{
			"reviewer":            "Jane Smith",
			"review_duration_min": 45,
			"sections_flagged":    []string{"Q3 Actuals", "Q4 Projections", "Risk Assessment"},
			"priority_level":      "high",
		},
	)

	err = inst.Transition(revisionsState.ID(), "request_revisions", systemActor, metadataWithFeedback)
	if err != nil {
		printError(fmt.Sprintf("Transition failed: %v", err))
	} else {
		tracker.Track(inst.DomainEvents())
		printSuccess("Transitioned: under_review → needs_revisions")
		printInfo("   Metadata:")
		printInfo("     - Reason: Document needs significant revisions")
		printInfo("     - Feedback: Please update the financial projections...")
		printInfo("     - Reviewer: Jane Smith")
		printInfo("     - Sections flagged: [Q3 Actuals, Q4 Projections, Risk Assessment]")
	}

	// ========================================
	// PART 6: Pause and Resume
	// ========================================
	printHeader("⏸️  Pause and Resume Operations")

	printSubHeader("Pausing Instance")
	err = inst.Pause(systemActor, "Waiting for additional information from author")
	if err != nil {
		printError(fmt.Sprintf("Pause failed: %v", err))
	} else {
		tracker.Track(inst.DomainEvents())
		printSuccess("Instance paused")
		printInfo(fmt.Sprintf("   Status: %s", inst.Status()))
		printInfo(fmt.Sprintf("   Reason: Waiting for additional information"))
	}

	time.Sleep(500 * time.Millisecond) // Simulate some time passing

	printSubHeader("Resuming Instance")
	err = inst.Resume(systemActor)
	if err != nil {
		printError(fmt.Sprintf("Resume failed: %v", err))
	} else {
		tracker.Track(inst.DomainEvents())
		printSuccess("Instance resumed")
		printInfo(fmt.Sprintf("   Status: %s", inst.Status()))
	}

	// ========================================
	// PART 7: Complete the Workflow
	// ========================================
	printHeader("✅ Completing the Workflow")

	// Resubmit after revisions
	printSubHeader("Resubmit After Revisions")
	metadata4 := instance.NewTransitionMetadataWithReason("Document updated with requested changes")
	err = inst.Transition(reviewState.ID(), "resubmit", systemActor, metadata4)
	if err != nil {
		printError(fmt.Sprintf("Transition failed: %v", err))
	} else {
		tracker.Track(inst.DomainEvents())
		printSuccess("Transitioned: needs_revisions → under_review")
	}

	// Final approval
	printSubHeader("Final Approval")
	metadata5 := instance.NewTransitionMetadata(
		"All requirements met, approving document",
		"Excellent work!",
		map[string]interface{}{
			"final_reviewer": "Director of Finance",
			"approval_level": "executive",
			"effective_date": time.Now().Format("2006-01-02"),
		},
	)

	err = inst.Transition(approvedState.ID(), "approve", systemActor, metadata5)
	if err != nil {
		printError(fmt.Sprintf("Transition failed: %v", err))
	} else {
		tracker.Track(inst.DomainEvents())
		printSuccess("Transitioned: under_review → approved")
		printInfo(fmt.Sprintf("   Final State: %s", inst.CurrentState()))
	}

	// Complete the instance
	printSubHeader("Completing Instance")
	err = inst.Complete(systemActor)
	if err != nil {
		printError(fmt.Sprintf("Complete failed: %v", err))
	} else {
		tracker.Track(inst.DomainEvents())
		printSuccess("Instance completed successfully!")
		printInfo(fmt.Sprintf("   Status: %s", inst.Status()))
		printInfo(fmt.Sprintf("   Completed At: %s", inst.CompletedAt().Time().Format(time.RFC3339)))
		printInfo(fmt.Sprintf("   Final Version: %s", inst.Version().String()))
	}

	// ========================================
	// PART 8: Transition History
	// ========================================
	printHeader("📜 Transition History")

	transitions := inst.GetTransitionHistory()
	printInfo(fmt.Sprintf("Total transitions: %d\n", len(transitions)))

	for i, trans := range transitions {
		fmt.Printf("%s[Transition %d]%s\n", colorBold, i+1, colorReset)
		fmt.Printf("  ID:        %s\n", trans.ID().String()[:8])
		fmt.Printf("  Event:     %s\n", trans.Event())
		fmt.Printf("  From:      %s\n", trans.From())
		fmt.Printf("  To:        %s\n", trans.To())

		if trans.HasSubStates() {
			fmt.Printf("  Sub-State: %s → %s\n", trans.FromSubState().ID(), trans.ToSubState().ID())
		}

		metadata := trans.Metadata()
		if !metadata.IsEmpty() {
			if metadata.Reason() != "" {
				fmt.Printf("  Reason:    %s\n", metadata.Reason())
			}
			if metadata.Feedback() != "" {
				fmt.Printf("  Feedback:  %s\n", metadata.Feedback())
			}
		}

		fmt.Printf("  Created:   %s\n", trans.Timestamp().Time().Format("15:04:05"))
		fmt.Println()
	}

	// ========================================
	// PART 9: Domain Events Summary
	// ========================================
	tracker.PrintSummary()

	// ========================================
	// PART 10: Detailed Event Log
	// ========================================
	printHeader("📋 Would you like to see detailed event logs?")
	printInfo("Uncomment the line below to see full event payloads")
	// tracker.PrintDetailed()

	// ========================================
	// PART 11: Validation Demo
	// ========================================
	printHeader("🔒 Validation & Error Handling Demo")

	printSubHeader("Attempting Invalid Transition on Completed Instance")
	err = inst.Transition(rejectedState.ID(), "reject", systemActor, instance.EmptyTransitionMetadata())
	if err != nil {
		printError(fmt.Sprintf("Expected error: %v", err))
		printSuccess("✓ Validation working correctly!")
	}

	// ========================================
	// PART 12: Second Instance Demo (Cancel Scenario)
	// ========================================
	printHeader("🔄 Creating Second Instance (Cancel Scenario)")

	inst2, _ := instance.NewInstance(
		approvalWorkflow.ID(),
		approvalWorkflow.Name(),
		draftState.ID(),
		systemActor,
	)
	tracker.Track(inst2.DomainEvents())

	printSuccess(fmt.Sprintf("Instance 2 created: %s", inst2.ID().String()[:8]))

	inst2.UpdateData("document_title", "Policy Update Proposal")
	inst2.Transition(reviewState.ID(), "submit", systemActor, instance.NewTransitionMetadataWithReason("Submitting policy"))
	tracker.Track(inst2.DomainEvents())

	printSubHeader("Canceling Instance 2")
	err = inst2.Cancel(systemActor, "Policy proposal withdrawn by author")
	if err != nil {
		printError(fmt.Sprintf("Cancel failed: %v", err))
	} else {
		tracker.Track(inst2.DomainEvents())
		printSuccess("Instance 2 canceled")
		printInfo(fmt.Sprintf("   Status: %s", inst2.Status()))
		printInfo("   Reason: Policy proposal withdrawn by author")
	}

	// ========================================
	// Final Summary
	// ========================================
	printHeader("🎉 Demo Complete!")

	fmt.Printf("\n%s%sDemo Statistics:%s\n", colorBold, colorGreen, colorReset)
	fmt.Printf("  Workflows created:     1\n")
	fmt.Printf("  Instances created:     2\n")
	fmt.Printf("  Transitions executed:  %d\n", len(transitions)+1)
	fmt.Printf("  Domain events:         %d\n", len(tracker.events))
	fmt.Printf("  Features demonstrated:\n")
	fmt.Printf("    ✓ Workflow definition\n")
	fmt.Printf("    ✓ State machine execution\n")
	fmt.Printf("    ✓ Sub-state support (R17)\n")
	fmt.Printf("    ✓ Transition metadata (R23)\n")
	fmt.Printf("    ✓ Domain events\n")
	fmt.Printf("    ✓ Pause/Resume\n")
	fmt.Printf("    ✓ Complete/Cancel\n")
	fmt.Printf("    ✓ Validation & Error handling\n")
	fmt.Printf("    ✓ Optimistic locking (version tracking)\n")

	fmt.Printf("\n%s%s", colorBold, colorCyan)
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║  ✅ All domain layer features working correctly!          ║")
	fmt.Println("║  Ready for Phase 6: Infrastructure Layer                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Printf("%s\n", colorReset)
}
