// main.go
// Copyright (c) 2026 Ichijo Hodaka
// WPT Parameter Search 2（ランダム探索）
// - 線形一様 / 対数一様で各引数をサンプリング
// - 関数値が範囲に入れば OK、入らなければ NG
// - OK/NG をそれぞれ最大 N 件保存（保存枠が埋まっても探索は継続）
// - 終了条件：繰り返し回数到達 or Ctrl-C
// - 最後に OK/NG の割合（iters に対する比率）を表示
//
// 表示は有効数字4桁（%.4g）

package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"sync/atomic"
)

type Scale int

const (
	Linear Scale = iota
	Log
)

type ParamSpec struct {
	Name  string
	Min   float64
	Max   float64
	Scale Scale
}

type Sample struct {
	Values map[string]float64
	Y      float64
	OK     bool
}

type Range struct {
	Min float64
	Max float64
}

func inRange(x float64, r Range) bool {
	return r.Min <= x && x <= r.Max
}

func sampleOne(rng *rand.Rand, p ParamSpec) (float64, error) {
	if p.Max < p.Min {
		return 0, fmt.Errorf("param %s: Max < Min", p.Name)
	}
	switch p.Scale {
	case Linear:
		u := rng.Float64()
		return p.Min + u*(p.Max-p.Min), nil
	case Log:
		if p.Min <= 0 || p.Max <= 0 {
			return 0, fmt.Errorf("param %s: log sampling requires Min>0 and Max>0 (got Min=%g Max=%g)", p.Name, p.Min, p.Max)
		}
		lnMin := math.Log(p.Min)
		lnMax := math.Log(p.Max)
		u := rng.Float64()
		return math.Exp(lnMin + u*(lnMax-lnMin)), nil
	default:
		return 0, fmt.Errorf("param %s: unknown scale", p.Name)
	}
}

func main() {

	cfg := DefaultConfig()

	params := cfg.Params
	yRange := cfg.YRange
	maxIters := cfg.MaxIters
	maxOKSave := cfg.MaxOKSave
	maxNGSave := cfg.MaxNGSave
	printEvery := cfg.PrintEvery
	seed := cfg.Seed
	xlsxFile := cfg.XLSXFile
	f := cfg.F

	// ============================================================
	// 探索本体
	// ============================================================

	order := make([]string, 0, len(params))
	{
		seen := map[string]bool{}
		for _, p := range params {
			if seen[p.Name] {
				panic("duplicate param name: " + p.Name)
			}
			seen[p.Name] = true
			order = append(order, p.Name)
		}
	}

	// Ctrl-C 対応
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		fmt.Println("\n[Ctrl-C] interrupt received. stopping...")
		cancel()
	}()

	rng := rand.New(rand.NewSource(seed))

	okList := make([]Sample, 0, maxOKSave)
	ngList := make([]Sample, 0, maxNGSave)

	var iters int64
	var okHits int64
	var ngHits int64

	// 進捗表示（固定幅・行の残りを消す）
	printProgress := func(i int64) {
		// % は固定幅 6 桁（例: " 80.00"）
		var pct float64
		if maxIters > 0 {
			pct = float64(i) / float64(maxIters) * 100.0
		}

		okh := atomic.LoadInt64(&okHits)
		ngh := atomic.LoadInt64(&ngHits)

		// 固定幅で表示（桁が増えても位置が動かない）
		// iters: 12 桁幅、OK/NG: 12 桁幅（必要なら増やしてOK）
		line := fmt.Sprintf(
			"\riter=%12d (%6.2f%%)  OK_hits=%12d  NG_hits=%12d",
			i, pct, okh, ngh,
		)

		// 前回表示より短くなった場合に備えて行末を消す（十分な空白を付ける）
		fmt.Print(line + "                                    ")
	}

	for {
		i := atomic.LoadInt64(&iters)
		if i >= maxIters {
			break
		}
		select {
		case <-ctx.Done():
			goto DONE
		default:
		}

		vals := make(map[string]float64, len(params))
		for _, p := range params {
			v, err := sampleOne(rng, p)
			if err != nil {
				fmt.Println("\nerror:", err)
				return
			}
			vals[p.Name] = v
		}

		y := f(vals)
		ok := !math.IsNaN(y) && !math.IsInf(y, 0) && inRange(y, yRange)

		if ok {
			atomic.AddInt64(&okHits, 1)
		} else {
			atomic.AddInt64(&ngHits, 1)
		}

		// 保存は「枠が空いているときだけ」。枠が埋まっても探索は続行。
		s := Sample{Values: vals, Y: y, OK: ok}
		if ok {
			if maxOKSave > 0 && len(okList) < maxOKSave {
				okList = append(okList, s)
			}
		} else {
			if maxNGSave > 0 && len(ngList) < maxNGSave {
				ngList = append(ngList, s)
			}
		}

		n := atomic.AddInt64(&iters, 1)
		if printEvery > 0 && (n%printEvery == 0) {
			printProgress(n)
		}
	}

DONE:
	fmt.Println()
	printProgress(atomic.LoadInt64(&iters))

	total := atomic.LoadInt64(&iters)
	okc := atomic.LoadInt64(&okHits)
	ngc := atomic.LoadInt64(&ngHits)

	PrintSummary(seed, yRange, total, okc, ngc)

	PrintSampleTable("=== OK (saved) ===", order, okList)
	fmt.Println()
	PrintSampleTable("=== NG (saved) ===", order, ngList)

	if xlsxFile != "" {
		if err := SaveToXLSX(xlsxFile, order, okList, ngList, total, okc, ngc); err != nil {
			fmt.Println("xlsx save error:", err)
		} else {
			fmt.Println("xlsx saved:", xlsxFile)
		}
	}

	if cfg.OKTSVFile != "" {
		if err := SaveListToTSV(cfg.OKTSVFile, order, okList); err != nil {
			fmt.Println("tsv save error (OK):", err)
		} else {
			fmt.Println("tsv saved (OK):", cfg.OKTSVFile)
		}
	}

	if cfg.NGTSVFile != "" {
		if err := SaveListToTSV(cfg.NGTSVFile, order, ngList); err != nil {
			fmt.Println("tsv save error (NG):", err)
		} else {
			fmt.Println("tsv saved (NG):", cfg.NGTSVFile)
		}
	}

}
