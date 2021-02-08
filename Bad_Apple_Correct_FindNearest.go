package main

import (
	"os"
	"time"
	"math"
	"github.com/nsf/termbox-go"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep"
    "fmt"
    "image"
    "image/color"
    "image/draw"
    "image/gif"
    "io"
)

const FrameDelayMultiply = 10

type (
	Color struct {
		r int
		g int
		b int
		colorAttribute termbox.Attribute
	}

	BadApple struct {
		Width int
		Height int
		Media *gif.GIF
		Audio beep.StreamSeekCloser
		Format beep.Format
		Colors []Color
		channel chan int
	}
)

func GetGifDimensions(gif *gif.GIF) (x, y int) {
    var lowestX int
    var lowestY int
    var highestX int
    var highestY int

    for _, img := range gif.Image {
        if img.Rect.Min.X < lowestX {
            lowestX = img.Rect.Min.X
        }
        if img.Rect.Min.Y < lowestY {
            lowestY = img.Rect.Min.Y
        }
        if img.Rect.Max.X > highestX {
            highestX = img.Rect.Max.X
        }
        if img.Rect.Max.Y > highestY {
            highestY = img.Rect.Max.Y
        }
    }

    return highestX - lowestX, highestY - lowestY
}

func Resize(img image.Image, width int, height int) image.Image {
    minX := img.Bounds().Min.X
    minY := img.Bounds().Min.Y
    maxX := img.Bounds().Max.X
    maxY := img.Bounds().Max.Y
    
    for (maxX-minX)%width != 0 {
        maxX--
    }

    for (maxY-minY)%height!= 0 {
        maxY--
    }

    scaleX := (maxX - minX) / width
    scaleY := (maxY - minY) / height

    imgRect := image.Rect(0, 0, width, height)
    resImg := image.NewRGBA(imgRect)
    draw.Draw(resImg, resImg.Bounds(), &image.Uniform{C: color.White}, image.ZP, draw.Src)

    for y := 0; y < height; y += 1 {
        for x := 0; x < width; x += 1 {
            averageColor := GetAverageColor(img, minX+x*scaleX, minX+(x+1)*scaleX, minY+y*scaleY, minY+(y+1)*scaleY)
            resImg.Set(x, y, averageColor)
        }
    }

    return resImg
}

func GetAverageColor(img image.Image, minX int, maxX int, minY int, maxY int) color.Color {
    var averageRed float64
    var averageGreen float64
    var averageBlue float64
    var averageAlpha float64
    scale := 1.0 / float64((maxX-minX)*(maxY-minY))

    for i := minX; i < maxX; i++ {
        for k := minY; k < maxY; k++ {
            r, g, b, a := img.At(i, k).RGBA()
            averageRed += float64(r) * scale
            averageGreen += float64(g) * scale
            averageBlue += float64(b) * scale
            averageAlpha += float64(a) * scale
        }
    }

    averageRed = math.Sqrt(averageRed)
    averageGreen = math.Sqrt(averageGreen)
    averageBlue = math.Sqrt(averageBlue)
    averageAlpha = math.Sqrt(averageAlpha)

    averageColor := color.RGBA{
        R: uint8(averageRed),
        G: uint8(averageGreen),
        B: uint8(averageBlue),
        A: uint8(averageAlpha)}

    return averageColor
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func ColorDistance(r1, g1, b1, r2, g2, b2 int) int {
	return Abs(r1 - r2) + Abs(g1 - g2) + Abs(b1 - b2)
}

func (apple *BadApple) FindNearest(r, g, b int) termbox.Attribute {
	nearest := int(^uint(0) >> 1) 
	var best termbox.Attribute

	for i := 0; i < len(apple.Colors); i++ {
		distance := ColorDistance(r, g, b, apple.Colors[i].r, apple.Colors[i].g, apple.Colors[i].b)
		if distance < nearest {
			nearest = distance
			best = apple.Colors[i].colorAttribute
		}
	}

	return best
}

func (apple *BadApple) GenearteFrame(img image.Image) []termbox.Attribute {
	var frame []termbox.Attribute

    for x := 0; x < apple.Width; x++ {
		for y := 0; y < apple.Height; y++ {
			r, g, b, _ := img.At(x, y).RGBA()
			frame = append(frame, apple.FindNearest(int(r), int(g), int(b)))
		}
	}

	return frame
}

func (apple *BadApple) ReadGif(FileName string) {
	f, _ := os.Open(FileName)

	var reader io.Reader
	reader = f

	gif, err := gif.DecodeAll(reader)

    if err != nil {
		fmt.Println("Problem decoding gif!")
		os.Exit(1)
    }

	apple.Media = gif
}

func (apple *BadApple) ReadAudio(FileName string) {
	f, err := os.Open(FileName)

	if err != nil {
		fmt.Println("Problem reading audio file!")
		os.Exit(1)
	}

	apple.Audio, apple.Format, err = mp3.Decode(f)

	if err != nil {
		fmt.Println("Problem decoding mp3!")
		os.Exit(1)
	}
}

func (apple *BadApple) DrawFrame(overpaintImage draw.Image, srcImg image.Image)  {
    draw.Draw(overpaintImage, overpaintImage.Bounds(), srcImg, image.ZP, draw.Over)
    image := Resize(overpaintImage, apple.Width, apple.Height)
	frame := apple.GenearteFrame(image)

	counter := 0

	for x := 0; x < apple.Width; x++ {
		for y := 0; y < apple.Height; y++ {
			termbox.SetCell(x, y, ' ', termbox.ColorDefault, frame[counter])
			counter++
		}
	}

	termbox.Flush()
	apple.channel <- 0
}

func (apple *BadApple) PlaySound(delay int) {
	time.Sleep(time.Duration(delay) * time.Millisecond)
	speaker.Init(apple.Format.SampleRate, apple.Format.SampleRate.N(time.Second/10))
	speaker.Play(apple.Audio)
}

func (apple *BadApple) Start() {
	termbox.SetCursor(0, 0)
	termbox.HideCursor()

	imgWidth, imgHeight := GetGifDimensions(apple.Media)
	overpaintImage := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(overpaintImage, overpaintImage.Bounds(), apple.Media.Image[0], image.ZP, draw.Src)

	go apple.PlaySound(250)

    for i, srcImg := range apple.Media.Image {
		go apple.DrawFrame(overpaintImage, srcImg)
		time.Sleep(time.Duration(int(float64(apple.Media.Delay[i]) * FrameDelayMultiply)) * time.Millisecond)
		<- apple.channel //Wait for frame to draw in case if its take longer than delay
    }

	termbox.SetCursor(0, 0)
	fmt.Println("Stopped playing!")
	time.Sleep(3000 * time.Millisecond)
}


func main() {
	if err := termbox.Init(); err != nil {
		os.Exit(1)
	}

	defer termbox.Close()

	var apple BadApple
	apple.channel = make(chan int)
	apple.Width = 160
	apple.Height = 60

	apple.Colors = append(apple.Colors, Color{12, 12, 12, termbox.ColorBlack})
	//apple.Colors = append(apple.Colors, Color{197, 15, 31, termbox.ColorRed})
	//apple.Colors = append(apple.Colors, Color{19, 161, 14, termbox.ColorGreen})
	//apple.Colors = append(apple.Colors, Color{193, 156, 0, termbox.ColorYellow})
	//apple.Colors = append(apple.Colors, Color{0, 55, 218, termbox.ColorBlue})
	//apple.Colors = append(apple.Colors, Color{136, 23, 152, termbox.ColorMagenta})
	//apple.Colors = append(apple.Colors, Color{58, 150, 221, termbox.ColorCyan})
	apple.Colors = append(apple.Colors, Color{204, 204, 204, termbox.ColorWhite})

	apple.ReadGif("Bad_Apple.gif")
	apple.ReadAudio("Bad_Apple.mp3")
	apple.Start()
}
