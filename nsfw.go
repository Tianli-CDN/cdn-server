package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
)

type NSFWResult struct {
	NSFW   bool   `json:"nsfw"`
	Reason string `json:"reason"`
}

type Setu struct {
	Path string       `json:"path"`
	NSFW NSFWResponse `json:"nsfw"`
}

func detectNSFW(imageData []byte, pathAll string) (NSFWResult, error) {
	imageType := http.DetectContentType(imageData)
	if imageType == "image/webp" {
		pngData, err := convertWebpToPng(imageData)
		if err != nil {
			return NSFWResult{}, fmt.Errorf("转换图片格式失败：%v", err)
		}
		imageData = pngData
	}

	for len(imageData) > 1024*1024 {
		compressedData, err := compressImage(imageData)
		if err != nil {
			return NSFWResult{}, fmt.Errorf("压缩图片失败：%v", err)
		}

		os.WriteFile("compressed.jpg", compressedData, 0644)
		if err != nil {
			return NSFWResult{}, fmt.Errorf("写入压缩图片失败：%v", err)
		}

		imageData = compressedData

		if len(imageData) <= 1024*1024 {
			break
		}
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	var part io.Writer
	var err error

	if imageType == "image/png" {
		part, err = writer.CreateFormFile("image", "image.png")
	} else {
		part, err = writer.CreateFormFile("image", "image.jpg")
	}

	if err != nil {
		return NSFWResult{}, fmt.Errorf("创建请求主体失败：%v", err)
	}

	part.Write(imageData)
	writer.Close()

	resp, err := http.Post("http://localhost:6012/classify", writer.FormDataContentType(), body)
	if err != nil {
		return NSFWResult{}, fmt.Errorf("请求图片NSFW检查失败：%v", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return NSFWResult{}, fmt.Errorf("读取图片NSFW检查响应失败：%v", err)
	}

	var nsfwResponse NSFWResponse
	if err := json.Unmarshal(responseBody, &nsfwResponse); err != nil {
		return NSFWResult{}, fmt.Errorf("解析图片NSFW检查响应失败：%v", err)
	}

	fmt.Printf("NSFW检查结果：%+v\n", nsfwResponse.Porn)

	Porn, err := strconv.ParseFloat(pornStr, 64)
	if err != nil {
		return NSFWResult{}, fmt.Errorf("解析Porn阈值失败：%v", err)
	}

	if nsfwResponse.Porn > Porn {
		setu := Setu{
			Path: pathAll,
			NSFW: nsfwResponse,
		}
		setuJSON, err := json.Marshal(setu)
		if err != nil {
			return NSFWResult{}, fmt.Errorf("序列化setu.json失败：%v", err)
		}

		if _, err := os.Stat("setu.json"); err == nil {
			existingJSON, err := os.ReadFile("setu.json")
			if err != nil {
				return NSFWResult{}, fmt.Errorf("读取setu.json失败：%v", err)
			}

			mergedJSON := append(existingJSON, setuJSON...)

			err = os.WriteFile("setu.json", mergedJSON, 0644)
			if err != nil {
				return NSFWResult{}, fmt.Errorf("写入setu.json失败：%v", err)
			}
		} else {
			err = os.WriteFile("setu.json", setuJSON, 0644)
			if err != nil {
				return NSFWResult{}, fmt.Errorf("写入setu.json失败：%v", err)
			}
		}

		return NSFWResult{
			NSFW:   true,
			Reason: fmt.Sprintf("好耶，事涩涩，但是不能哦！分数:%+v", nsfwResponse.Porn),
		}, nil
	}

	return NSFWResult{NSFW: false, Reason: ""}, nil
}
