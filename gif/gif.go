package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	"math"
	"math/rand"
	"os"
)

var palette = []color.Color{color.White, color.Black}

const (
	whiteIndex = 0
	blackIndex = 1
)

func main() {
	file, writer := getWriter()
	Lissajous(writer)

	defer file.Close()
}

func Lissajous(out io.Writer) {
	const (
		cycles  = 5
		res     = 0.001
		size    = 100
		nframes = 64
		delay   = 8
	)
	freq := rand.Float64() * 3.0
	anim := gif.GIF{LoopCount: nframes}
	phase := 0.0
	for i := 0; i < nframes; i++ {
		rect := image.Rect(0, 0, 2*size+1, 2*size+1)
		img := image.NewPaletted(rect, palette)
		for t := 0.0; t < cycles*2*math.Pi; t += res {
			x := math.Sin(t)
			y := math.Sin(t*freq + phase)
			img.SetColorIndex(size+int(x*size+0.5), size+int(y*size+0.5), blackIndex)
		}
		phase += 0.1
		anim.Delay = append(anim.Delay, delay)
		anim.Image = append(anim.Image, img)
	}
	err := gif.EncodeAll(out, &anim)
	if nil != err {
		fmt.Printf("err : %s", err)
	}
}

func getWriter() (*os.File, io.Writer) {

	_, exist := os.Stat("./gif.gif")
	if exist == nil {
		exist := os.Remove("./gif.gif")
		fmt.Printf("%v", exist)
	}

	file, err := os.OpenFile("./gif.gif", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0744)
	if err != nil {
		panic(err)
	}

	return file, bufio.NewWriter(file)

}
