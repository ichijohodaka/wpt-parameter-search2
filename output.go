// output.go
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

func fmt4(x float64) string { return fmt.Sprintf("%10.4g", x) }

// 表示用：パラメータ名に応じて単位変換してから固定幅で文字列化する
func fmtParam(name string, x float64) string {
	switch name {
	case "f":
		return fmt4(x / 1e3) // Hz -> kHz
	case "L1", "L2":
		return fmt4(x * 1e6) // H -> µH
	case "C1", "C2":
		return fmt4(x * 1e9) // F -> nF
	default:
		return fmt4(x)
	}
}

// 表のヘッダ用（単位ラベル）
func labelFor(name string) string {
	switch name {
	case "f":
		return "f [kHz]"
	case "R1", "R2":
		return name + " [Ω]"
	case "L1", "L2":
		return name + " [µH]"
	case "C1", "C2":
		return name + " [nF]"
	default:
		return name
	}
}

func PrintSummary(seed int64, yRange Range, total, okc, ngc int64) {
	var okRatio, ngRatio float64
	if total > 0 {
		okRatio = float64(okc) / float64(total)
		ngRatio = float64(ngc) / float64(total)
	}

	fmt.Printf("\nseed=%d\n", seed)
	fmt.Printf("yRange=[%s, %s]\n", fmt4(yRange.Min), fmt4(yRange.Max))
	fmt.Printf("iters=%d  OK_hits=%d  NG_hits=%d\n", total, okc, ngc)
	fmt.Printf("OK_ratio=%s  NG_ratio=%s\n\n", fmt4(okRatio), fmt4(ngRatio))
}

func PrintSampleTable(title string, order []string, list []Sample) {
	fmt.Println(title)
	if len(list) == 0 {
		fmt.Println("(none)")
		return
	}

	// ヘッダ（No + params + y）
	headers := make([]string, 0, len(order)+2)
	headers = append(headers, "No")
	for _, k := range order {
		headers = append(headers, labelFor(k))
	}
	headers = append(headers, "y")

	// 各セルの文字列を先に作る
	rows := make([][]string, len(list))
	for i, s := range list {
		row := make([]string, 0, len(headers))
		row = append(row, fmt.Sprintf("%d", i+1))
		for _, k := range order {
			row = append(row, fmtParam(k, s.Values[k]))
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

	// No以外は1文字詰める（右側の余白を削る）ための pad
	pad := func(col int) int {
		if col == 0 {
			return 2 // No 列だけ従来通り
		}
		return 1
	}

	printLine := func() {
		fmt.Print("+")
		for i, w := range widths {
			fmt.Print(strings.Repeat("-", w+pad(i)) + "+")
		}
		fmt.Println()
	}

	// ヘッダ行
	printLine()
	fmt.Print("|")
	for i, h := range headers {
		if i == 0 {
			fmt.Printf(" %-*s |", widths[i], h)
		} else {
			fmt.Printf(" %-*s|", widths[i], h)
		}
	}
	fmt.Println()
	printLine()

	// データ行
	for _, row := range rows {
		fmt.Print("|")
		for j, cell := range row {
			if j == 0 {
				fmt.Printf(" %*s |", widths[j], cell)
			} else {
				fmt.Printf(" %*s|", widths[j], cell)
			}
		}
		fmt.Println()
	}
	printLine()
	fmt.Println()
}

func SaveToXLSX(
	filename string,
	order []string,
	okList []Sample,
	ngList []Sample,
	total, okc, ngc int64,
) error {

	f := excelize.NewFile()

	// Summary
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

	// OK / NG
	writeList := func(sheet string, list []Sample) {
		f.NewSheet(sheet)

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

		for i, s := range list {
			row := i + 2
			col = 1

			cell, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellValue(sheet, cell, i+1)
			col++

			for _, k := range order {
				cell, _ := excelize.CoordinatesToCellName(col, row)
				f.SetCellValue(sheet, cell, s.Values[k]) // xlsx は元単位で保存
				col++
			}
			cell, _ = excelize.CoordinatesToCellName(col, row)
			f.SetCellValue(sheet, cell, s.Y)
		}
	}

	writeList("OK", okList)
	writeList("NG", ngList)

	return f.SaveAs(filename)
}

// list を TSV で保存する（order の列順で出力）
// ※TSV は「触らない」方針なら、ここは呼び出し側のままでOK
func SaveListToTSV(filename string, order []string, list []Sample) error {
	if filename == "" {
		return nil
	}

	fp, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer fp.Close()

	w := csv.NewWriter(fp)
	w.Comma = '\t'

	header := append([]string{}, order...)
	header = append(header, "y")
	if err := w.Write(header); err != nil {
		return err
	}

	for _, s := range list {
		row := make([]string, 0, len(order)+1)
		for _, k := range order {
			row = append(row, fmtParam(k, s.Values[k])) // ← 今の挙動維持
		}
		row = append(row, fmt4(s.Y))
		if err := w.Write(row); err != nil {
			return err
		}
	}

	w.Flush()
	return w.Error()
}
