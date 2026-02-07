// config.go
package main

import (
	"math"
	"time"
)

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
	F          func(x map[string]float64) float64
}

// ============================================================
// ユーザー設定（ここから）
// ============================================================

// 各変数がとりうる範囲をここで決める。
// Linearにすると範囲に対して一様な乱数をもとに値を選ぶ
// Logにすると範囲に対して対数的に一様な乱数をもとに値を選ぶ
// つまり，たとえば範囲が10^3から10^5なら，3から5の範囲に対して一様な乱数を発生し
// たとえば4.2になったら10^4.2という値を選ぶことになる。
// これに対して同じ範囲10^3=1000から10^5=100000に対して一様な乱数を用いると，
// 1000-10000から値が選ばれる確率は，10000-100000から値が選ばれる確率の1/10になってしまう。
// そこで広い範囲をもつ変数の探索を行う場合は対数的に一様な分布をもつ
// 乱数を選択した方が偏りなく探索を行うことができる。
func DefaultConfig() Config {
	params := []ParamSpec{
		{Name: "k", Min: 0.01, Max: 0.02, Scale: Linear},
		{Name: "f", Min: 10_000, Max: 100_000, Scale: Log}, // Hz
		{Name: "R1", Min: 1.0, Max: 1.0, Scale: Log},       // Ω
		{Name: "R2", Min: 10.0, Max: 10.0, Scale: Log},     // Ω
		{Name: "L1", Min: 140e-6, Max: 140e-6, Scale: Log}, // H
		{Name: "L2", Min: 80e-6, Max: 80e-6, Scale: Log},   // H
		{Name: "C1", Min: 1e-9, Max: 100e-9, Scale: Log},   // F
		{Name: "C2", Min: 1e-9, Max: 100e-9, Scale: Log},   // F
	}

	// 関数の値の範囲。計算結果がこの範囲に入っていれば正解，入っていなければ不正解
	yRange := Range{Min: 0.4, Max: 1.0}

	// 繰り返し回数（10_000_000 で数秒）
	maxIters := int64(10_000_000)

	// 保存する正解・不正解の数（多くするとファイルサイズ増）
	maxOKSave := 10
	maxNGSave := 10

	// 進行状況表示の更新間隔（多すぎると遅くなる）
	printEvery := int64(200_000)

	// 乱数 seed（実行時刻ベース）
	// seedを自分で決めてもよい。
	// 実行すると乱数発生に使用されたseedが表示されるので，そのときと同じ数字を使うと
	// 同じ乱数が発生するので，結果も同じになる。
	seed := time.Now().UnixNano()

	// xlsx 出力（空文字なら保存しない）
	// "" にすると保存はせず表示だけ
	xlsxFile := "result.xlsx"

	// 関数（例：WPT SS の PN）
	// 自分が考えている問題に合わせて変更する。
	// ここの変数を変えた場合は params := []ParamSpec{ の下も同時に修正する。
	// ここの変数の"k"などと，params := []ParamSpec{ の下の "k"は一致している必要がある。
	f := func(x map[string]float64) float64 {
		k := x["k"]
		fHz := x["f"]
		R1 := x["R1"]
		R2 := x["R2"]
		L1 := x["L1"]
		L2 := x["L2"]
		C1 := x["C1"]
		C2 := x["C2"]

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
		F:          f,
	}
}
