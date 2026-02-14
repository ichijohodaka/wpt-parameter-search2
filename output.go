// output.go
package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

func fmt4(x float64) string { return fmt.Sprintf("%10.4g", x) }

func fmtCell(x float64) string {
	if math.IsNaN(x) {
		return "       NaN"
	}
	if math.IsInf(x, 1) {
		return "      +Inf"
	}
	if math.IsInf(x, -1) {
		return "      -Inf"
	}
	return fmt4(x)
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

func PrintSampleTable(title string, params []ParamSpec, list []Sample, maxPrint int) {

	fmt.Println(title)
	if len(list) == 0 {
		fmt.Println("(none)")
		return
	}
	origLen := len(list)
	if maxPrint > 0 && len(list) > maxPrint {
		list = list[:maxPrint]
	}

	// ヘッダ（No + params + y）
	headers := make([]string, 0, len(params)+2)
	headers = append(headers, "No")
	for _, p := range params {
		headers = append(headers, p.Label)
	}
	headers = append(headers, "y")

	// 各セルの文字列を先に作る（表示用の単位変換は DisplayScale で行う）
	rows := make([][]string, len(list))
	for i, s := range list {
		row := make([]string, 0, len(headers))
		row = append(row, fmt.Sprintf("%d", i+1))
		for _, p := range params {
			v := s.Values[p.Key] * p.DisplayScale
			row = append(row, fmtCell(v))
		}
		row = append(row, fmtCell(s.Y))
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

	if maxPrint > 0 && origLen > maxPrint {
		fmt.Printf("(printed %d of %d; truncated for console)\n\n", maxPrint, origLen)
	}

}

func SaveToXLSX(
	filename string,
	params []ParamSpec,
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

	okRatio := 0.0
	ngRatio := 0.0
	if total > 0 {
		okRatio = float64(okc) / float64(total)
		ngRatio = float64(ngc) / float64(total)
	}

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

		// xlsx は「元単位で保存」する（見出しは Key にするのが無難）
		for _, p := range params {
			cell, _ := excelize.CoordinatesToCellName(col, 1)
			f.SetCellValue(sheet, cell, p.Key)
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

			for _, p := range params {
				cell, _ := excelize.CoordinatesToCellName(col, row)
				f.SetCellValue(sheet, cell, s.Values[p.Key]) // 元単位
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

// list を TSV で保存する（params の順で出力）
// TSV は「表示単位で保存」する（DisplayScale を適用）
func SaveListToTSV(filename string, params []ParamSpec, list []Sample) error {
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

	// ヘッダ：Label
	header := make([]string, 0, len(params)+1)
	for _, p := range params {
		header = append(header, p.Label)
	}
	header = append(header, "y")
	if err := w.Write(header); err != nil {
		return err
	}

	for _, s := range list {
		row := make([]string, 0, len(params)+1)
		for _, p := range params {
			v := s.Values[p.Key] * p.DisplayScale
			row = append(row, fmt.Sprintf("%.10g", v)) // TSV は桁少し多め（解析向け）
		}
		row = append(row, fmt.Sprintf("%.10g", s.Y))
		if err := w.Write(row); err != nil {
			return err
		}
	}

	w.Flush()
	return w.Error()
}
