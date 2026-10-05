package world_test

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/ui/world"
)

// Person and Name are the two columns from the Bevy quick start.
// https://bevy.org/learn/quick-start/getting-started/apps/
type Person struct{}

type Name struct{ Text string }

func addPeople(_ context.Context, w *world.World) {
	world.Spawn2(w, Person{}, Name{Text: "Elaina Proctor"})
	world.Spawn2(w, Person{}, Name{Text: "Renzo Hume"})
	world.Spawn2(w, Person{}, Name{Text: "Zayna Nieves"})
}

func updatePeople(_ context.Context, w *world.World) {
	stopped := false
	world.With[Name, Person](world.Query[Name](w)).Each(func(_ world.Entity, name *Name) {
		if stopped || name.Text != "Elaina Proctor" {
			return
		}
		name.Text = "Elaina Hume"
		stopped = true
	})
}

func greetPeople(_ context.Context, w *world.World) {
	world.With[Name, Person](world.Query[Name](w)).Read(func(_ world.Entity, name Name) {
		fmt.Println("hello", name.Text)
	})
}

// Example is that quick start: a startup system spawns the people, then
// a chain renames Elaina and greets the column.
func Example() {
	sim := world.New()
	sim.System(world.Startup, addPeople)
	sim.Chain(world.Update, updatePeople, greetPeople)
	sim.Frame(context.Background())
	// Output:
	// hello Elaina Hume
	// hello Renzo Hume
	// hello Zayna Nieves
}
