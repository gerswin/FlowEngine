package main

import (
	"fmt"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
)

// Quick test script - modify and run to test different scenarios
func main() {
	fmt.Println("🧪 FlowEngine Quick Test")
	fmt.Println("========================\n")

	actor := shared.NewID()

	// ============================================
	// 1. Define tu workflow aquí
	// ============================================
	fmt.Println("📋 Creating Workflow...")

	// Estados
	draft, _ := workflow.NewState("draft", "Borrador")
	review, _ := workflow.NewState("review", "En Revisión")
	approved, _ := workflow.NewState("approved", "Aprobado")
	approved = approved.AsFinal()
	rejected, _ := workflow.NewState("rejected", "Rechazado")
	rejected = rejected.AsFinal()

	// Crear workflow
	wf, _ := workflow.NewWorkflow("Aprobación Simple", draft, actor)
	wf.AddState(review)
	wf.AddState(approved)
	wf.AddState(rejected)

	// Eventos (transiciones)
	submitEvent, _ := workflow.NewEvent("submit", []workflow.State{draft}, review)
	approveEvent, _ := workflow.NewEvent("approve", []workflow.State{review}, approved)
	rejectEvent, _ := workflow.NewEvent("reject", []workflow.State{review}, rejected)

	wf.AddEvent(submitEvent)
	wf.AddEvent(approveEvent)
	wf.AddEvent(rejectEvent)

	fmt.Printf("✅ Workflow: %s\n", wf.Name())
	fmt.Printf("   Estados: %d\n", len(wf.States()))
	fmt.Printf("   Eventos: %d\n\n", len(wf.Events()))

	// ============================================
	// 2. Crear instancia y ejecutar transiciones
	// ============================================
	fmt.Println("🎬 Creating Instance...")

	inst, _ := instance.NewInstance(wf.ID(), wf.Name(), draft.ID(), actor)
	inst.UpdateData("title", "Documento de Prueba")
	inst.UpdateVariable("priority", "high")

	fmt.Printf("✅ Instance ID: %s\n", inst.ID().String()[:8])
	fmt.Printf("   Estado inicial: %s\n", inst.CurrentState())
	fmt.Printf("   Status: %s\n\n", inst.Status())

	// ============================================
	// 3. Ejecutar transiciones (modifica aquí!)
	// ============================================
	fmt.Println("🔄 Executing Transitions...")

	// Transición 1: Submit
	fmt.Println("\n→ Transition: submit")
	metadata1 := instance.NewTransitionMetadataWithReason("Enviando para revisión")
	err := inst.Transition(review.ID(), "submit", actor, metadata1)
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
	} else {
		fmt.Printf("   ✅ Estado: %s → %s\n", draft.ID(), inst.CurrentState())
		fmt.Printf("   Version: %s\n", inst.Version().String())
	}

	// Transición 2: Approve
	fmt.Println("\n→ Transition: approve")
	metadata2 := instance.NewTransitionMetadata(
		"Documento aprobado",
		"Excelente trabajo",
		map[string]interface{}{
			"reviewer": "Juan Pérez",
			"score":    95,
		},
	)
	err = inst.Transition(approved.ID(), "approve", actor, metadata2)
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
	} else {
		fmt.Printf("   ✅ Estado: %s → %s\n", review.ID(), inst.CurrentState())
		fmt.Printf("   Version: %s\n", inst.Version().String())
	}

	// Completar instancia
	fmt.Println("\n→ Completing instance")
	err = inst.Complete(actor)
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
	} else {
		fmt.Printf("   ✅ Status: %s\n", inst.Status())
		fmt.Printf("   Completado en: %s\n", inst.CompletedAt().Time().Format("15:04:05"))
	}

	// ============================================
	// 4. Ver resultados
	// ============================================
	fmt.Println("\n📊 Results:")
	fmt.Println("===========")

	fmt.Printf("\nEstado Final: %s\n", inst.CurrentState())
	fmt.Printf("Status: %s\n", inst.Status())
	fmt.Printf("Versión: %s\n", inst.Version().String())
	fmt.Printf("Transiciones: %d\n", inst.TransitionCount())

	fmt.Println("\n📜 Transition History:")
	for i, trans := range inst.GetTransitionHistory() {
		fmt.Printf("  [%d] %s: %s → %s\n",
			i+1,
			trans.Event(),
			trans.From(),
			trans.To())

		metadata := trans.Metadata()
		if !metadata.IsEmpty() && metadata.Reason() != "" {
			fmt.Printf("      Reason: %s\n", metadata.Reason())
			if metadata.Feedback() != "" {
				fmt.Printf("      Feedback: %s\n", metadata.Feedback())
			}
		}
	}

	// Eventos generados
	events := inst.DomainEvents()
	fmt.Printf("\n📬 Domain Events Generated: %d\n", len(events))
	for _, evt := range events {
		fmt.Printf("  • %s\n", evt.Type())
	}

	fmt.Println("\n✅ Test completed successfully!")
}
