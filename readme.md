# WPT Paramter Search 2

## 実行方法

- `git`をインストールしておく
- `go`言語をインストールしておく
- コマンドプロンプトやpowershellで
```bash
git clone https://github.com/ichijohodaka/wpt-parameter-search2.git
```
- `wpt-parameter-search2`フォルダに移動して実行する
```bash
cd wpt-parameter-search2
go run .
```

## カスタマイズ

- `config.go`の「ユーザー設定（ここから）」から「ユーザー設定（ここまで）」の間のコードを適宜書き換える。他は修正の必要はない。
- たとえば変数L3を追加したい場合は，
```go
		{Key: "L3", Label: "L3 [µH]", Min: 80e-6, Max: 80e-6, Scale: Log, DisplayScale: 1e6},
```
の行を追加する。そして
```go
	f := func(x map[string]float64) float64 {
```
の下の関数の定義式を修正するとよい。
- ユーザーが関数を変更して使うことを想定しているので、buildせずにコードを直接修正して実行
- GOには非常に厳しい文法チェックがあり，pythonのようにはいかない。
- `Visual Studio Code`エディターにGO拡張を追加しておくと，文法チェックが直ちに入り，間違いがあれば指摘される。
- 動かない場合はコードをchatGPTに貼り付けて，状況を伝えると大抵は修正できる。GOのコードは記述の自由度が少ない分，chatGPTも修正をかけやすいのだろう。

## 出力（コンソール表示）（`output.go`）

- 保存した正解リスト
- 保存した不正解リスト
- Ctrl-Cで終了した場合はその時点における保存した正解リスト，不正解リスト（思ったより時間がかかってしまった場合や，とりあえず繰り返し回数を多くしておいてその時点までの結果を知りたいとき）
- エクセルファイル（ファイル名を指定した場合）
- tsv形式のファイル（ファイル名を指定した場合）

## 終了条件

- 繰り返し回数に到達
- Ctrl-C

## アルゴリズム

以下を終了条件を満たすまで繰り返す。進行状況確認用に正解数，不正解数カウンターを表示

- 線形分点指定の引数については，範囲中から線形的に一様乱数によって値を選ぶ
- 対数分点指定の引数については，範囲中から対数的に一様乱数によって値を選ぶ
- 選んだ引数の値で関数の値を計算
- 関数の値が範囲に入っていれば正解リストに追加。
- 関数の値が範囲に入っていなければ不正解リストに追加

## 使用例（実験装置の製作時）

- 電力がそこそこ出るような実験装置を作りたい。
- 電源は決まっている。
- コイルもある。

このような場合は、R1, R2, L1, L2のMinとMaxを決まっている値に一致させ、探索範囲を狭くしたほうが探索が成功しやすい。

## 電力が指定した範囲に入るようなパラメータの領域を視覚化

- R1, R2, L1, L2, C1, C2は決まった値を用いる。
- 正規化電力が0.1以上，0.5以下になるようなkとfの領域をみたい。
- はじめはいろいろ試したいので，下記のようにしてファイル保存しない。

この条件で実行したところ、次の結果を得た。

```bash
iter=    10000000 (100.00%)  OK_hits=      442626  NG_hits=     9557374

seed=1771046723902691400
yRange=[       0.1,        0.5]
iters=10000000  OK_hits=442626  NG_hits=9557374
OK_ratio=   0.04426  NG_ratio=    0.9557
```

OK_hitsの数がそこそこ大きいので領域をうまく表せそうである。そこでそのデータを保存する。データ数が大きくなりそうなので、むやみに保存すべきでない。ng.tsvのほうはいらないだろう。さきほどのseedを用いると同じ結果になる。OKの数もわかっているので、保存する数を一致させる。

保存した結果を視覚化してみよう。ここでは`gnuplot`を用いた例を示す。次を`regionOK.gp`のように拡張子`gp`を付けて`ok.tsv`と同じフォルダに保存し，`.\regionOK.gp`で実行すると，`scatter.png`ができる。

```gnuplot
set datafile separator "\t"

set key autotitle columnhead

set terminal pngcairo size 900,600 enhanced font "Arial,20"
set output "scatter.png"

set xlabel "f [kHz]"
set ylabel "k"

set logscale x 10
set mxtics 10

set grid lc rgb "#5a5a5a" lw 3
set border lw 3

set xrange [40:100]

plot "ok.tsv" using "f [kHz]":"k" with points pt 7 ps 0.6 notitle
```

![scatter plot](scatter.png)



