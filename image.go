package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"strings"

	"github.com/chai2010/webp"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/nfnt/resize"
)

// 图片处理函数，获取主题色调等
func handleImageRequest(imageData []byte, params string) (json.RawMessage, string) {

	if strings.Contains(params, "get=theme") {
		theme, err := getThemeColor(imageData)
		if err != nil {
			log.Printf("获取图片主题色调时出错：%v\n", err)
			errorJSON := fmt.Sprintf(`{"error": "主题色调获取失败", "hitokoto": "%s"}`, hitokoto())
			return nil, errorJSON
		}
		return json.RawMessage(fmt.Sprintf(`{"theme":"%s"}`, theme)), "application/json"
	}

	if strings.Contains(params, "get=size") {
		img, _, err := image.Decode(bytes.NewReader(imageData))
		if err != nil {
			log.Printf("获取图片尺寸时出错：%v\n", err)
			errorJSON := fmt.Sprintf(`{"error": "图片尺寸获取失败", "hitokoto": "%s"}`, hitokoto())
			return nil, errorJSON
		}
		bounds := img.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()
		return json.RawMessage(fmt.Sprintf(`{"width":%d,"height":%d}`, width, height)), "application/json"
	}

	return nil, ""
}

// img2color
func getThemeColor(imageData []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", err
	}

	img = resize.Resize(50, 0, img, resize.Lanczos3)

	bounds := img.Bounds()
	var r, g, b uint32
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			r0, g0, b0, _ := c.RGBA()
			r += r0
			g += g0
			b += b0
		}
	}

	totalPixels := uint32(bounds.Dx() * bounds.Dy())
	averageR := r / totalPixels
	averageG := g / totalPixels
	averageB := b / totalPixels

	mainColor := colorful.Color{R: float64(averageR) / 0xFFFF, G: float64(averageG) / 0xFFFF, B: float64(averageB) / 0xFFFF}

	colorHex := mainColor.Hex()

	return colorHex, nil
}

func convertWebpToPng(webpData []byte) ([]byte, error) {
	img, err := webp.Decode(bytes.NewReader(webpData))
	if err != nil {
		return nil, fmt.Errorf("解码webp图像失败：%v", err)
	}

	buf := new(bytes.Buffer)

	err = png.Encode(buf, img)
	if err != nil {
		return nil, fmt.Errorf("转换图片格式失败：%v", err)
	}

	return buf.Bytes(), nil
}

func compressImage(imageData []byte) ([]byte, error) {

	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("解码图像失败：%v", err)
	}

	buf := new(bytes.Buffer)

	err = jpeg.Encode(buf, img, &jpeg.Options{Quality: 80})
	if err != nil {
		return nil, fmt.Errorf("压缩图片失败：%v", err)
	}

	return buf.Bytes(), nil
}

func convertImageToWebp(imageData []byte) ([]byte, error) {

	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("解码图像失败：%v", err)
	}

	buf := new(bytes.Buffer)

	err = webp.Encode(buf, img, &webp.Options{Lossless: false, Quality: 80})
	if err != nil {
		return nil, fmt.Errorf("转换图片格式失败：%v", err)
	}

	return buf.Bytes(), nil
}
