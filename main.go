package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	// BMP, GIFなど各種画像形式対応
	_ "image/gif"

	_ "golang.org/x/image/bmp"

	"github.com/nfnt/resize"
	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata" // Inconsolataフォントを使用
	"golang.org/x/image/math/fixed"
)

// 対応拡張子
var supportedExt = []string{".jpg", ".jpeg", ".png", ".gif", ".bmp"}

// テキスト描画用設定（Inconsolataを使用）
var (
	textFont  font.Face = inconsolata.Regular8x16
	textColor           = color.Black
)

func main() {
	dir := flag.String("dir", "", "Input directory containing images")
	output := flag.String("out", "output.png", "Output file name (png or jpg)")
	nValue := flag.Int("n", 3, "Number of images per row/column (n×n collage)")
	tileSize := flag.Int("tile", 300, "Tile size (width/height in pixels for the cell)")
	flag.Parse()

	if *dir == "" {
		log.Fatal("Please specify a directory with -dir")
	}

	// 画像ファイル一覧取得
	images, err := getImageFiles(*dir)
	if err != nil {
		log.Fatal(err)
	}

	total := (*nValue) * (*nValue)
	if len(images) < total {
		log.Fatalf("Not enough images in the directory: need at least %d, got %d", total, len(images))
	}

	// ランダムシード設定
	rand.Seed(time.Now().UnixNano())

	// n×n枚ランダム選択
	selected := randomSelect(images, total)

	// ここでファイル名でソート
	sort.Strings(selected)

	// 画像読み込み
	imgList, names := loadImages(selected)

	// コラージュ画像生成（アスペクト比維持）
	collageImg := createCollageImage(imgList, names, *nValue, *tileSize)

	// 出力ファイルに書き込み
	if err := saveImage(*output, collageImg); err != nil {
		log.Fatalf("Failed to save image: %v", err)
	}
	fmt.Printf("Saved collage image to %s\n", *output)
}

// getImageFiles はディレクトリ内の画像ファイル一覧を取得
func getImageFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && isImageFile(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// isImageFile は対応拡張子か判定
func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, e := range supportedExt {
		if ext == e {
			return true
		}
	}
	return false
}

// randomSelect は与えられたスライスからランダムにn要素選ぶ
func randomSelect(files []string, n int) []string {
	perm := rand.Perm(len(files))
	selected := make([]string, 0, n)
	for i := 0; i < n; i++ {
		selected = append(selected, files[perm[i]])
	}
	return selected
}

// loadImages は画像を読み込む（リサイズは後で行うためここではそのまま）
func loadImages(paths []string) ([]image.Image, []string) {
	var imgList []image.Image
	var names []string
	for _, imgPath := range paths {
		img, err := loadImage(imgPath)
		if err != nil {
			log.Fatalf("Failed to load image %s: %v", imgPath, err)
		}
		imgList = append(imgList, img)
		names = append(names, filepath.Base(imgPath))
	}
	return imgList, names
}

// loadImage はファイルから画像を読み込む
func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// createCollageImage はアスペクト比維持でリサイズ・配置、文字描画
func createCollageImage(imgList []image.Image, names []string, n, tileSize int) image.Image {
	margin := 10
	textHeight := 20

	finalWidth := n*tileSize + (n+1)*margin
	finalHeight := n*(tileSize+textHeight) + (n+1)*margin

	outputImg := image.NewRGBA(image.Rect(0, 0, finalWidth, finalHeight))

	// 背景を白で塗りつぶし
	draw.Draw(outputImg, outputImg.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	for i, originalImg := range imgList {
		row := i / n
		col := i % n

		// タイルの左上座標 (この中に画像を納める)
		x := margin + col*(tileSize+margin)
		y := margin + row*(tileSize+textHeight+margin)

		// オリジナル画像サイズ
		ow := originalImg.Bounds().Dx()
		oh := originalImg.Bounds().Dy()

		// アスペクト比維持リサイズ計算
		var newW, newH uint
		if float64(ow)/float64(oh) > 1.0 {
			// 横長
			newW = uint(tileSize)
			newH = uint(float64(tileSize) * float64(oh) / float64(ow))
		} else {
			// 縦長または正方形
			newH = uint(tileSize)
			newW = uint(float64(tileSize) * float64(ow) / float64(oh))
		}

		// リサイズ処理
		resized := resize.Resize(newW, newH, originalImg, resize.Lanczos3)

		// 中央に配置
		offsetX := x + (tileSize-int(newW))/2
		offsetY := y + (tileSize-int(newH))/2
		imgRect := image.Rect(offsetX, offsetY, offsetX+int(newW), offsetY+int(newH))
		draw.Draw(outputImg, imgRect, resized, image.Point{}, draw.Over)

		// ファイル名テキスト描画
		drawText(outputImg, x, y+tileSize+5, names[i])
	}

	return outputImg
}

// drawText はイメージ上にテキストを描画する
func drawText(img draw.Image, x, y int, text string) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: textFont,
		Dot: fixed.Point26_6{
			X: fixed.I(x),
			Y: fixed.I(y + textFont.Metrics().Ascent.Ceil()),
		},
	}
	d.DrawString(text)
}

// saveImage は拡張子でPNG/JPEGを判定し保存する
func saveImage(filename string, img image.Image) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png":
		err = png.Encode(f, img)
	case ".jpg", ".jpeg":
		err = jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	default:
		return errors.New("unsupported output format")
	}
	return err
}
