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
	"strings"
	"sync/atomic"

	"github.com/xuri/excelize/v2"
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

func fmt4(x float64) string { return fmt.Sprintf("%.4g", x) }

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

func printSampleTable(title string, order []string, list []Sample) {
	fmt.Println(title)
	if len(list) == 0 {
		fmt.Println("(none)")
		return
	}

	// ヘッダ（No + params + y）
	headers := append([]string{"No"}, order...)
	headers = append(headers, "y")

	// 各セルの文字列を先に作る
	rows := make([][]string, len(list))
	for i, s := range list {
		row := make([]string, 0, len(headers))
		row = append(row, fmt.Sprintf("%d", i+1))
		for _, k := range order {
			row = append(row, fmt4(s.Values[k]))
		}
		row = append(row, fmt4(s.Y))
		rows[i] = row
	}

	// 列幅を決定（ヘッダ or 中身の最大）
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for j, cell := range row {
			if len(cell) > widths[j] {
				widths[j] = len(cell)
			}
		}
	}

	// 罫線
	printLine := func() {
		fmt.Print("+")
		for _, w := range widths {
			fmt.Print(strings.Repeat("-", w+2) + "+")
		}
		fmt.Println()
	}

	// ヘッダ行
	printLine()
	fmt.Print("|")
	for i, h := range headers {
		fmt.Printf(" %-*s |", widths[i], h)
	}
	fmt.Println()
	printLine()

	// データ行
	for _, row := range rows {
		fmt.Print("|")
		for j, cell := range row {
			fmt.Printf(" %*s |", widths[j], cell) // 右寄せ
		}
		fmt.Println()
	}
	printLine()
	fmt.Println()
}

func saveToXLSX(
	filename string,
	order []string,
	okList []Sample,
	ngList []Sample,
	total, okc, ngc int64,
) error {

	f := excelize.NewFile()

	// --------------------
	// Summary シート
	// --------------------
	summary := "Summary"
	f.SetSheetName("Sheet1", summary)

	f.SetCellValue(summary, "A1", "Type")
	f.SetCellValue(summary, "B1", "Count")
	f.SetCellValue(summary, "C1", "Ratio")

	okRatio := float64(okc) / float64(total)
	ngRatio := float64(ngc) / float64(total)

	f.SetCellValue(summary, "A2", "OK")
	f.SetCellValue(summary, "B2", okc)
	f.SetCellValue(summary, "C2", okRatio)

	f.SetCellValue(summary, "A3", "NG")
	f.SetCellValue(summary, "B3", ngc)
	f.SetCellValue(summary, "C3", ngRatio)

	f.SetCellValue(summary, "A4", "ALL")
	f.SetCellValue(summary, "B4", total)
	f.SetCellValue(summary, "C4", 1.0)

	// --------------------
	// OK / NG シート
	// --------------------
	writeList := func(sheet string, list []Sample) {
		f.NewSheet(sheet)

		// ヘッダ
		col := 1
		f.SetCellValue(sheet, "A1", "No")
		col++

		for _, k := range order {
			cell, _ := excelize.CoordinatesToCellName(col, 1)
			f.SetCellValue(sheet, cell, k)
			col++
		}
		cell, _ := excelize.CoordinatesToCellName(col, 1)
		f.SetCellValue(sheet, cell, "y")

		// データ
		for i, s := range list {
			row := i + 2
			col = 1

			cell, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellValue(sheet, cell, i+1)
			col++

			for _, k := range order {
				cell, _ := excelize.CoordinatesToCellName(col, row)
				f.SetCellValue(sheet, cell, s.Values[k])
				col++
			}
			cell, _ = excelize.CoordinatesToCellName(col, row)
			f.SetCellValue(sheet, cell, s.Y)
		}
	}

	writeList("OK", okList)
	writeList("NG", ngList)

	// 保存
	return f.SaveAs(filename)
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

	var okRatio, ngRatio float64
	if total > 0 {
		okRatio = float64(okc) / float64(total)
		ngRatio = float64(ngc) / float64(total)
	}

	fmt.Printf("\nseed=%d\n", seed)
	fmt.Printf("yRange=[%s, %s]\n", fmt4(yRange.Min), fmt4(yRange.Max))
	fmt.Printf("iters=%d  OK_hits=%d  NG_hits=%d\n", total, okc, ngc)
	fmt.Printf("OK_ratio=%s  NG_ratio=%s\n\n", fmt4(okRatio), fmt4(ngRatio))

	printSampleTable("=== OK (saved) ===", order, okList)
	fmt.Println()
	printSampleTable("=== NG (saved) ===", order, ngList)

	if xlsxFile != "" {
		err := saveToXLSX(
			xlsxFile,
			order,
			okList,
			ngList,
			total,
			okc,
			ngc,
		)
		if err != nil {
			fmt.Println("xlsx save error:", err)
		} else {
			fmt.Println("xlsx saved:", xlsxFile)
		}
	}

}
