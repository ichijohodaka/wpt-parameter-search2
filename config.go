// config.go
package main

import (
	"math"
	"time"
)

// Get: ユーザー関数でキー打ち間違いしたら即気づけるようにする
func Get(x map[string]float64, key string) float64 {
	v, ok := x[key]
	if !ok {
		panic("missing key in x: " + key)
	}
	return v
}

// Config は「ユーザー設定」をまとめたもの
type Config struct {
	Params     []ParamSpec
	YRange     Range
	MaxIters   int64
	MaxOKSave  int
	MaxNGSave  int
	PrintEvery int64
	Seed       int64
	XLSXFile   string // "" なら保存しない
	OKTSVFile  string // "" なら保存しない
	NGTSVFile  string // "" なら保存しない
	MaxPrint   int    // コンソールに表示する最大件数（0なら制限なし）
	F          func(x map[string]float64) float64
}

// ============================================================
// ユーザー設定（ここから）
// ============================================================

func DefaultConfig() Config {
	// params に表示メタ（Label / DisplayScale）も持たせる。
	// これにより output.go は params を走査するだけで列・単位変換が決まる（switch不要）。
	params := []ParamSpec{
		{Key: "k", Label: "k", Min: 0.01, Max: 1.0, Scale: Linear, DisplayScale: 1.0},

		// 周波数：元は Hz だが表示は kHz にしたい → DisplayScale = 1e-3
		{Key: "f", Label: "f [kHz]", Min: 10_000, Max: 100_000, Scale: Log, DisplayScale: 1e-3},

		{Key: "R1", Label: "R1 [Ω]", Min: 1.0, Max: 1.0, Scale: Log, DisplayScale: 1.0},
		{Key: "R2", Label: "R2 [Ω]", Min: 10.0, Max: 10.0, Scale: Log, DisplayScale: 1.0},

		// インダクタ：元は H、表示は µH → *1e6
		{Key: "L1", Label: "L1 [µH]", Min: 140e-6, Max: 140e-6, Scale: Log, DisplayScale: 1e6},
		{Key: "L2", Label: "L2 [µH]", Min: 80e-6, Max: 80e-6, Scale: Log, DisplayScale: 1e6},

		// キャパシタ：元は F、表示は nF → *1e9
		{Key: "C1", Label: "C1 [nF]", Min: 47e-9, Max: 47e-9, Scale: Log, DisplayScale: 1e9},
		{Key: "C2", Label: "C2 [nF]", Min: 47e-9, Max: 47e-9, Scale: Log, DisplayScale: 1e9},
	}

	// 関数の値の範囲。計算結果がこの範囲に入っていれば正解，入っていなければ不正解
	yRange := Range{Min: 0.1, Max: 0.5}

	// 繰り返し回数（10_000_000 で数秒）
	maxIters := int64(10_000_000)

	// 保存する正解・不正解の数（多くするとファイルサイズ増）
	maxOKSave := 30110
	maxNGSave := 10

	maxPrint := 100

	// 進行状況表示の更新間隔（多すぎると遅くなる）
	printEvery := int64(200_000)

	// 乱数 seed（実行時刻ベース）
	seed := time.Now().UnixNano()
	seed = 1771046723902691400

	// xlsx 出力（空文字なら保存しない）
	xlsxFile := "result.xlsx"
	xlsxFile = ""

	// tsv 出力（"" なら保存しない）
	okTSVFile := "ok.tsv"
	// okTSVFile = ""
	ngTSVFile := "ng.tsv"
	ngTSVFile = ""

	// 関数（例：WPT SS の 正規化電力 PN）
	// params の Key と一致している必要がある（Get を使うとミスは即発覚する）。
	f := func(x map[string]float64) float64 {
		k := Get(x, "k")
		fHz := Get(x, "f")
		R1 := Get(x, "R1")
		R2 := Get(x, "R2")
		L1 := Get(x, "L1")
		L2 := Get(x, "L2")
		C1 := Get(x, "C1")
		C2 := Get(x, "C2")

		w := 2 * math.Pi * fHz

		term1 := w*L1 - 1.0/(w*C1)
		term2 := w*L2 - 1.0/(w*C2)

		A := (R1 * R2) + (term1 * term2) - (w * w * k * k * L1 * L2)
		B := (R1 * term2) - (R2 * term1)

		num := 4.0 * k * k * R1 * R2 * L1 * L2 * w * w
		den := (A * A) + (B * B) + num

		if den == 0 {
			return math.NaN()
		}
		return num / den
	}

	// ============================================================
	// ユーザー設定（ここまで）
	// ============================================================

	return Config{
		Params:     params,
		YRange:     yRange,
		MaxIters:   maxIters,
		MaxOKSave:  maxOKSave,
		MaxNGSave:  maxNGSave,
		PrintEvery: printEvery,
		Seed:       seed,
		XLSXFile:   xlsxFile,
		OKTSVFile:  okTSVFile,
		NGTSVFile:  ngTSVFile,
		MaxPrint:   maxPrint,
		F:          f,
	}
}
