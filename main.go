// main.go
// Copyright (c) 2026 Ichijo Hodaka
// WPT Parameter Search 2（ランダム探索）
//
// - params[] に定義された変数を、Linear / Log でサンプリング
// - f(x) の結果 y が yRange に入れば OK
// - OK/NG をそれぞれ最大 N 件保存（枠が埋まっても探索は継続）
// - 終了条件：繰り返し回数到達 or Ctrl-C
//
// 表示は output.go 側で params の DisplayScale/Label を使って自動化する

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

// ParamSpec: 変数の定義（探索範囲 + サンプリング方式 + 表示用メタ）
type ParamSpec struct {
	Key          string  // map のキー（例: "f"）
	Label        string  // 表示ヘッダ（例: "f [kHz]"）
	Min          float64 // 探索範囲 min（元単位）
	Max          float64 // 探索範囲 max（元単位）
	Scale        Scale   // Linear / Log（サンプリング用）
	DisplayScale float64 // 表示用スケール（例: Hz→kHz は 1e-3）
}

type Sample struct {
	Values map[string]float64 // 元単位で保持
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
		return 0, fmt.Errorf("param %s: Max < Min", p.Key)
	}
	switch p.Scale {
	case Linear:
		u := rng.Float64()
		return p.Min + u*(p.Max-p.Min), nil
	case Log:
		if p.Min <= 0 || p.Max <= 0 {
			return 0, fmt.Errorf("param %s: log sampling requires Min>0 and Max>0 (got Min=%g Max=%g)", p.Key, p.Min, p.Max)
		}
		lnMin := math.Log(p.Min)
		lnMax := math.Log(p.Max)
		u := rng.Float64()
		return math.Exp(lnMin + u*(lnMax-lnMin)), nil
	default:
		return 0, fmt.Errorf("param %s: unknown scale", p.Key)
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

	// params のキー重複チェック
	{
		seen := map[string]bool{}
		for _, p := range params {
			if p.Key == "" {
				panic("param key is empty")
			}
			if seen[p.Key] {
				panic("duplicate param key: " + p.Key)
			}
			seen[p.Key] = true
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
		var pct float64
		if maxIters > 0 {
			pct = float64(i) / float64(maxIters) * 100.0
		}
		okh := atomic.LoadInt64(&okHits)
		ngh := atomic.LoadInt64(&ngHits)

		line := fmt.Sprintf(
			"\riter=%12d (%6.2f%%)  OK_hits=%12d  NG_hits=%12d",
			i, pct, okh, ngh,
		)
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
			vals[p.Key] = v
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

	total := atomic.LoadInt64(&iters)
	okc := atomic.LoadInt64(&okHits)
	ngc := atomic.LoadInt64(&ngHits)

	PrintSummary(seed, yRange, total, okc, ngc)

	PrintSampleTable("=== OK (saved) ===", params, okList, cfg.MaxPrint)
	fmt.Println()
	PrintSampleTable("=== NG (saved) ===", params, ngList, cfg.MaxPrint)

	if xlsxFile != "" {
		if err := SaveToXLSX(xlsxFile, params, okList, ngList, total, okc, ngc); err != nil {
			fmt.Println("xlsx save error:", err)
		} else {
			fmt.Println("xlsx saved:", xlsxFile)
		}
	}

	if cfg.OKTSVFile != "" {
		if err := SaveListToTSV(cfg.OKTSVFile, params, okList); err != nil {
			fmt.Println("tsv save error (OK):", err)
		} else {
			fmt.Println("tsv saved (OK):", cfg.OKTSVFile)
		}
	}

	if cfg.NGTSVFile != "" {
		if err := SaveListToTSV(cfg.NGTSVFile, params, ngList); err != nil {
			fmt.Println("tsv save error (NG):", err)
		} else {
			fmt.Println("tsv saved (NG):", cfg.NGTSVFile)
		}
	}
}
