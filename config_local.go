// config.goｍ接さわらずにここで差し替え

package main

import (
	"math"
	"time"
)

func init() {
	LocalOverride = func(cfg *Config) {

		// コメントアウトでデフォルト値が使われる。

		// 関数の値の範囲。計算結果がこの範囲に入っていれば正解，入っていなければ不正解
		cfg.YRange = Range{Min: 0.35, Max: 0.5}
		// 繰り返し回数（10_000_000 で数秒）
		cfg.MaxIters = int64(10_000_000)
		// 保存する正解・不正解の数（多くするとファイルサイズ増）
		cfg.MaxOKSave = 10
		cfg.MaxNGSave = 10
		// 結果表示を制限。ファイルには全部保存される。
		cfg.MaxPrint = 10
		// 進行状況表示の更新間隔（多すぎると遅くなる）
		cfg.PrintEvery = int64(200_000)
		// 乱数 seed（実行時刻ベース）
		cfg.Seed = time.Now().UnixNano()
		// xlsx 出力のファイル名（"" なら保存しない）
		cfg.XLSXFile = ""
		// tsv 出力のファイル名（"" なら保存しない）
		cfg.OKTSVFile = ""
		cfg.NGTSVFile = ""

		// --- 変数範囲（表示ラベルと表示スケールも含む） ---
		cfg.Params = []ParamSpec{
			{Key: "k", Label: "k", Min: 0.01, Max: 0.01, Scale: Linear, DisplayScale: 1.0},

			// 周波数：元は Hz だが表示は kHz → DisplayScale = 1e-3
			{Key: "f", Label: "f [kHz]", Min: 10_000, Max: 100_000, Scale: Log, DisplayScale: 1e-3},

			{Key: "R1", Label: "R1 [Ω]", Min: 1.0, Max: 1.0, Scale: Log, DisplayScale: 1.0},
			{Key: "R2", Label: "R2 [Ω]", Min: 10.0, Max: 10.0, Scale: Log, DisplayScale: 1.0},

			// インダクタ：元は H、表示は µH → *1e6
			{Key: "L1", Label: "L1 [µH]", Min: 100e-6, Max: 200e-6, Scale: Log, DisplayScale: 1e6},
			{Key: "L2", Label: "L2 [µH]", Min: 100e-6, Max: 200e-6, Scale: Log, DisplayScale: 1e6},

			// キャパシタ：元は F、表示は nF → *1e9
			{Key: "C1", Label: "C1 [nF]", Min: 1e-9, Max: 47e-9, Scale: Log, DisplayScale: 1e9},
			{Key: "C2", Label: "C2 [nF]", Min: 1e-9, Max: 47e-9, Scale: Log, DisplayScale: 1e9},
		}

		// --- 関数（WPT SS の PN） ---
		// cfg.Params の Key と一致している必要がある
		cfg.F = func(x map[string]float64) float64 {
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
	}
}
