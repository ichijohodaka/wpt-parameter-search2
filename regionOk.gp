set datafile separator "\t"

set key autotitle columnhead

set terminal pngcairo size 900,600 enhanced font "Arial,12"
set output "scatter.png"

set xlabel "f"
set ylabel "k"

set logscale x 10
set format x "10^{%L}"
set mxtics 10

set grid

plot "ok.tsv" using "f":"k" with points pt 7 ps 0.6 notitle
