package ga

import (
	"context"
	"reflect"
	"testing"

	"simple_tuan/pkg/tsp"
)

func TestEncodingVariants(t *testing.T) {
	inst, err := tsp.Random(12, 42)
	if err != nil {
		t.Fatal(err)
	}
	for _, encoding := range []string{"permutation", "random-key"} {
		for _, initialization := range []string{"random", "mixed"} {
			t.Run(encoding+"/"+initialization, func(t *testing.T) {
				p := Params{Population: 20, Generations: 25, Seed: 7, Encoding: encoding, Initialization: initialization}
				a, err := Solve(inst, p)
				if err != nil {
					t.Fatal(err)
				}
				b, err := Solve(inst, p)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(a.Generations, b.Generations) {
					t.Fatal("not deterministic")
				}
				for _, g := range a.Generations {
					if err := inst.ValidateTour(g.BestTour); err != nil {
						t.Fatal(err)
					}
				}
				p.LocalSearch = true
				c, err := Solve(inst, p)
				if err != nil {
					t.Fatal(err)
				}
				if err := inst.ValidateTour(c.BestTour); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
func TestKeysTieAndCompatibility(t *testing.T) {
	if got := decodeKeys([]float64{0.5, 0.2, 0.5, 0.1}); !reflect.DeepEqual(got, []int{3, 1, 0, 2}) {
		t.Fatal(got)
	}
	p := Params{}
	if err := p.Normalize(); err != nil {
		t.Fatal(err)
	}
	if p.Encoding != "permutation" || p.Initialization != "random" {
		t.Fatal(p)
	}
	p.Encoding = "random-key"
	if err := p.Normalize(); err == nil {
		t.Fatal("incompatible operator accepted")
	}
	inst, _ := tsp.Random(5, 42)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := SolveContext(ctx, inst, Params{}); err == nil {
		t.Fatal("cancellation ignored")
	}
}
