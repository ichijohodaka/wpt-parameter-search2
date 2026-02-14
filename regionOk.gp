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
