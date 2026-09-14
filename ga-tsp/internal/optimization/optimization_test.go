package optimization

import (
	"context"
	"math"
	"reflect"
	"testing"
)

func testParams(kind string) Params {
	p := Params{Population: 60, Generations: 100, CrossoverRate: .9, MutationRate: .08, Elitism: 2,
		Seed: 7, Selection: "tournament", Crossover: "onepoint", Mutation: "bitflip", Initialization: "random"}
	if kind == "cec" {
		p.Crossover = "blend"
		p.Mutation = "gaussian"
	} else {
		p.GreedyRepair = true // 背包测试默认开启贪心修复，保持现有测试期望
	}
	return p
}

func TestCECDefinitions(t *testing.T) {
	for _, id := range []string{"f1", "f2", "f9"} {
		for _, d := range []int{2, 10, 30, 50} {
			c, err := NewCEC(id, d)
			if err != nil {
				t.Fatal(err)
			}
			if got := c.Evaluate(c.Shift); got != c.Benchmark.Optimum {
				t.Fatalf("%s 最优点值: %v", id, got)
			}
			x := append([]float64{}, c.Shift...)
			for i := range x {
				x[i] += 1
			}
			want := float64(d)
			if id == "f2" {
				want = float64(d*(d+1)*(2*d+1)) / 6
			}
			if got := c.Evaluate(x) - c.Benchmark.Optimum; math.Abs(got-want) > 1e-7 {
				t.Fatalf("%s D=%d 公式不符: 得到 %v 期望 %v", id, d, got, want)
			}
		}
	}
}

func TestKnapsackAgainstEnumeration(t *testing.T) {
	for seed := range int64(10) {
		k, _ := RandomKnapsack(10, seed)
		var exact int
		for mask := range 1 << len(k.Items) {
			var weight, value int
			for i, item := range k.Items {
				if mask&(1<<i) != 0 {
					weight += item.Weight
					value += item.Value
				}
			}
			if weight <= k.Capacity {
				exact = max(exact, value)
			}
		}
		if k.ExactValue() != exact {
			t.Fatal("动态规划与穷举不一致")
		}
		for _, cross := range []string{"onepoint", "uniform"} {
			for _, mutation := range []string{"bitflip", "swap"} {
				p := testParams("knapsack")
				p.Crossover = cross
				p.Mutation = mutation
				result, err := SolveKnapsack(context.Background(), k, p)
				if err != nil {
					t.Fatal(err)
				}
				previous := -1.0
				for _, g := range result.Generations {
					var weight, value int
					for i, v := range g.Genes {
						if v != 0 && v != 1 {
							t.Fatal("非二进制基因")
						}
						if v == 1 {
							weight += k.Items[i].Weight
							value += k.Items[i].Value
						}
					}
					if weight > k.Capacity || float64(value) != g.Best {
						t.Fatal("不可行背包结果")
					}
					if g.Best < previous || g.Best > float64(exact) {
						t.Fatal("精英保留或最优界不符")
					}
					previous = g.Best
				}
			}
		}
	}
}

func TestKnapsackWithoutGreedyRepairIsHarder(t *testing.T) {
	// 关闭贪心修复后，超重解适应度=0；GA 必须自己找到可行的高质量解，
	// 收敛速度应明显慢于开启贪心修复（验证 DP 未注入、收敛先验来自修复函数）。
	k, _ := RandomKnapsack(20, 7)
	p := testParams("knapsack")
	p.Generations = 200
	p.GreedyRepair = true
	resOn, _ := SolveKnapsack(context.Background(), k, p)
	p.GreedyRepair = false
	resOff, _ := SolveKnapsack(context.Background(), k, p)
	// 关闭贪心修复后仍应改进初始种群（GA 能找到可行解）
	if resOff.Best <= 0 {
		t.Fatalf("关闭贪心修复后 GA 应至少找到非零可行解，实际 Best=%v", resOff.Best)
	}
	// 关闭贪心修复的最终最优不应超过开启贪心修复（贪心修复是性能上界）
	if resOff.Best > resOn.Best+1e-9 {
		t.Fatalf("关闭贪心修复不应优于开启：off=%v on=%v", resOff.Best, resOn.Best)
	}
	// 关闭贪心修复应显著更难：收敛代数更晚或最终值更差
	if resOff.Best >= resOn.Best-1e-9 && resOff.ConvergedGen <= resOn.ConvergedGen {
		t.Fatalf("关闭贪心修复应更难收敛：off(Best=%v,Gen=%v) on(Best=%v,Gen=%v)",
			resOff.Best, resOff.ConvergedGen, resOn.Best, resOn.ConvergedGen)
	}
}

func TestReproducibilityAndBounds(t *testing.T) {
	for _, selection := range []string{"rank", "tournament"} {
		for _, initial := range []string{"random", "stratified"} {
			for _, cross := range []string{"arithmetic", "blend"} {
				for _, mutation := range []string{"gaussian", "reset"} {
					c, _ := NewCEC("f9", 10)
					p := testParams("cec")
					p.Selection = selection
					p.Initialization = initial
					p.Crossover = cross
					p.Mutation = mutation
					a, err := SolveCEC(context.Background(), c, p)
					if err != nil {
						t.Fatal(err)
					}
					b, _ := SolveCEC(context.Background(), c, p)
					if !reflect.DeepEqual(a.Generations, b.Generations) {
						t.Fatal("种子不可复现")
					}
					for _, g := range a.Generations {
						for _, v := range g.Genes {
							if v < c.Benchmark.Lower || v > c.Benchmark.Upper {
								t.Fatal("基因越界")
							}
						}
					}
					if a.Best > a.Generations[0].Best || a.Evaluations != p.Population*p.Generations {
						t.Fatal("结果统计不符")
					}
				}
			}
		}
	}
}

func TestZeroParametersAndCancellation(t *testing.T) {
	p := testParams("cec")
	p.CrossoverRate = 0
	p.MutationRate = 0
	p.Elitism = 0
	c, _ := NewCEC("f1", 2)
	result, err := SolveCEC(context.Background(), c, p)
	if err != nil {
		t.Fatal(err)
	}
	if result.Params.CrossoverRate != 0 || result.Params.MutationRate != 0 || result.Params.Elitism != 0 {
		t.Fatal("零参数被覆盖")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := SolveCEC(ctx, c, p); err == nil {
		t.Fatal("未响应取消")
	}
	p.MutationRate = math.NaN()
	if err := p.Validate("cec"); err == nil {
		t.Fatal("接受非法概率")
	}
}
