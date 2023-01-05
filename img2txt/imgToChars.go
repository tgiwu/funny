package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
)

var arrayLen = 0

var scale = 1

func main() {

	image.RegisterFormat("jpeg", "jpeg", jpeg.Decode, jpeg.DecodeConfig)

	printChan := make(chan []byte, 1024)

	handleChan := make(chan image.Image)

	finishChan := make(chan int8)

	go readFromFile("/Users/yangzhang/Pictures/joker.jpg", handleChan)

	go handle(handleChan, printChan)

	go printFile(printChan, finishChan)

	finish := <-finishChan

	if finish == 1 {
		println("finish")
	} else {
		println("going")
	}

}

func readFromFile(path string, handleChan chan image.Image) {

	exist, _ := checkFileExist(path)

	if !exist {
		panic(path + " not exist")
	}

	imgFile, err := os.Open(path)

	if err != nil {
		panic("读取失败")
	}

	img, _, err := image.Decode(imgFile)
	if err != nil {
		panic(err)
	}
	printInfo(img)
	handleChan <- img
}

func checkFileExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func printInfo(img image.Image) {
	println("=============================")

	bounds := img.Bounds()
	arrayLen = bounds.Max.X / scale
	fmt.Printf("len = %d, %d x %d ", img.Bounds().Size(), bounds.Max.X, bounds.Max.Y)
	println("=============================")

}

func printFile(printChan chan []byte, finishChan chan int8) {

	_, exist := os.Stat("./img.log")
	if exist == nil {
		exist := os.Remove("./img.log")
		fmt.Printf("%v", exist)
	}

	file, err := os.OpenFile("./img.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0744)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	for true {
		bytes := <-printChan
		//println(string(bytes))
		n, err := file.Write(bytes)
		if err != nil {
			panic(err)
		}
		fmt.Printf("n = %d\n", n)

		writeString, err := file.WriteString("\n")
		if err != nil {
			panic(err)
		}
		fmt.Printf("write = %d \n", writeString)

		if len(bytes) > arrayLen {
			finishChan <- 1
			break
		}
	}

}

func handle(handleChan chan image.Image, printChan chan []byte) {
	img := <-handleChan

	bounds := img.Bounds()

	for y := 0; y < bounds.Max.Y; y += scale {
		line := make([]byte, bounds.Max.X/scale)
		for x := 0; x < bounds.Max.X; x += scale {
			r, g, b, a := img.At(x, y).RGBA()

			asc := ((r+g+b+a)/4)%26 + 62

			line[x/scale] = byte(asc)

		}
		if y == bounds.Max.Y-scale {
			line = append(line, ' ')
		}
		printChan <- line
	}
}

func getWH(data []byte) (int, int) {
	var offset int
	imgByteLen := len(data)
	for i := 0; i < imgByteLen-1; i++ {
		if data[i] != 0xff {
			continue
		}
		if data[i+1] == 0xC0 || data[i+1] == 0xC1 || data[i+1] == 0xC2 {
			offset = i
			break
		}
	}

	offset += 5

	if offset >= imgByteLen {
		return 0, 0
	}

	h := int(data[offset])<<8 + int(data[offset+1])
	w := int(data[offset+2])<<8 + int(data[offset+3])

	return w, h
}
