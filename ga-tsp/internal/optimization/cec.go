package optimization

import (
	"context"
	"embed"
	"fmt"
	"math"
	"strconv"
	"strings"
)

//go:embed data/data_*.txt
var shiftFiles embed.FS

type Benchmark struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Formula     string  `json:"formula"`
	Lower       float64 `json:"lower"`
	Upper       float64 `json:"upper"`
	Optimum     float64 `json:"optimum"`
	file        string
}

func Benchmarks() []Benchmark {
	return []Benchmark{
		{ID: "f1", Name: "F1 · 移位球函数", Description: "单峰、可分离；用于观察基本收敛能力。",
			Formula: "f(x) = Σ zᵢ² − 450，z = x − o", Lower: -100, Upper: 100, Optimum: -450, file: "data_sphere.txt"},
		{ID: "f2", Name: "F2 · 移位累积平方函数", Description: "单峰、不可分离；变量之间相互耦合。",
			Formula: "f(x) = Σᵢ (Σⱼ₌₁ⁱ zⱼ)² − 450，z = x − o", Lower: -100, Upper: 100, Optimum: -450,
			file: "data_schwefel_102.txt"},
		{ID: "f9", Name: "F9 · 移位拉斯特里金函数", Description: "多峰、可分离；用于观察局部最优与种群探索。",
			Formula: "f(x) = Σ [zᵢ² − 10 cos(2πzᵢ) + 10] − 330，z = x − o", Lower: -5, Upper: 5,
			Optimum: -330, file: "data_rastrigin.txt"},
	}
}

type CEC struct {
	Benchmark Benchmark `json:"benchmark"`
	Dimension int       `json:"dimension"`
	Shift     []float64 `json:"shift"`
}

func NewCEC(id string, dimension int) (CEC, error) {
	if dimension < 2 || dimension > 50 {
		return CEC{}, fmt.Errorf("维数须在 2～50 之间")
	}
	for _, b := range Benchmarks() {
		if b.ID != id {
			continue
		}
		data, err := shiftFiles.ReadFile("data/" + b.file)
		if err != nil {
			return CEC{}, err
		}
		fields := strings.Fields(string(data))
		if len(fields) < dimension {
			return CEC{}, fmt.Errorf("移位数据不完整")
		}
		shift := make([]float64, dimension)
		for i := range shift {
			shift[i], err = strconv.ParseFloat(fields[i], 64)
			if err != nil {
				return CEC{}, fmt.Errorf("移位数据解析失败: %w", err)
			}
		}
		return CEC{Benchmark: b, Dimension: dimension, Shift: shift}, nil
	}
	return CEC{}, fmt.Errorf("未知 CEC 函数，支持 f1、f2、f9")
}

// Evaluate 按 CEC 2005 定义计算函数；F2 累积项包括最后一维。
func (c CEC) Evaluate(x []float64) float64 {
	var sum, prefix float64
	for i, v := range x {
		z := v - c.Shift[i]
		switch c.Benchmark.ID {
		case "f1":
			sum += z * z
		case "f2":
			prefix += z
			sum += prefix * prefix
		case "f9":
			sum += z*z - 10*math.Cos(2*math.Pi*z) + 10
		}
	}
	return sum + c.Benchmark.Optimum
}

func SolveCEC(ctx context.Context, c CEC, params Params) (Result, error) {
	repair := func(x []float64) {
		for i, v := range x {
			x[i] = max(c.Benchmark.Lower, min(c.Benchmark.Upper, v))
		}
	}
	task := problem{kind: "cec", size: c.Dimension, lower: c.Benchmark.Lower, upper: c.Benchmark.Upper,
		optimal: c.Benchmark.Optimum, evaluate: c.Evaluate, repair: repair}
	return run(ctx, task, params)
}

// Landscape 返回其余维固定在最优位置的二维切片，供前端绘制热力图。
func (c CEC) Landscape() [][]float64 {
	const n = 65
	grid := make([][]float64, n)
	x := append([]float64{}, c.Shift...)
	for row := range grid {
		grid[row] = make([]float64, n)
		x[1] = c.Benchmark.Upper - float64(row)/float64(n-1)*(c.Benchmark.Upper-c.Benchmark.Lower)
		for col := range n {
			x[0] = c.Benchmark.Lower + float64(col)/float64(n-1)*(c.Benchmark.Upper-c.Benchmark.Lower)
			grid[row][col] = c.Evaluate(x)
		}
	}
	return grid
}
