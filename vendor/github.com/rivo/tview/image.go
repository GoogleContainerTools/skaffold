package tview

import (
	"image"
	"math"

	"github.com/gdamore/tcell/v2"
)

// Types of dithering applied to images.
const (
	DitheringNone           = iota // No dithering.
	DitheringFloydSteinberg        // Floyd-Steinberg dithering (the default).
)

// The number of colors supported by true color terminals (R*G*B = 256*256*256).
const TrueColor = 16777216

// This map describes what each block element looks like. A 1 bit represents a
// pixel that is drawn, a 0 bit represents a pixel that is not drawn. The least
// significant bit is the top left pixel, the most significant bit is the bottom
// right pixel, moving row by row from left to right, top to bottom.
var blockElements = map[rune]uint64{
	BlockLowerOneEighthBlock:            0b1111111100000000000000000000000000000000000000000000000000000000,
	BlockLowerOneQuarterBlock:           0b1111111111111111000000000000000000000000000000000000000000000000,
	BlockLowerThreeEighthsBlock:         0b1111111111111111111111110000000000000000000000000000000000000000,
	BlockLowerHalfBlock:                 0b1111111111111111111111111111111100000000000000000000000000000000,
	BlockLowerFiveEighthsBlock:          0b1111111111111111111111111111111111111111000000000000000000000000,
	BlockLowerThreeQuartersBlock:        0b1111111111111111111111111111111111111111111111110000000000000000,
	BlockLowerSevenEighthsBlock:         0b1111111111111111111111111111111111111111111111111111111100000000,
	BlockLeftSevenEighthsBlock:          0b0111111101111111011111110111111101111111011111110111111101111111,
	BlockLeftThreeQuartersBlock:         0b0011111100111111001111110011111100111111001111110011111100111111,
	BlockLeftFiveEighthsBlock:           0b0001111100011111000111110001111100011111000111110001111100011111,
	BlockLeftHalfBlock:                  0b0000111100001111000011110000111100001111000011110000111100001111,
	BlockLeftThreeEighthsBlock:          0b0000011100000111000001110000011100000111000001110000011100000111,
	BlockLeftOneQuarterBlock:            0b0000001100000011000000110000001100000011000000110000001100000011,
	BlockLeftOneEighthBlock:             0b0000000100000001000000010000000100000001000000010000000100000001,
	BlockQuadrantLowerLeft:              0b0000111100001111000011110000111100000000000000000000000000000000,
	BlockQuadrantLowerRight:             0b1111000011110000111100001111000000000000000000000000000000000000,
	BlockQuadrantUpperLeft:              0b0000000000000000000000000000000000001111000011110000111100001111,
	BlockQuadrantUpperRight:             0b0000000000000000000000000000000011110000111100001111000011110000,
	BlockQuadrantUpperLeftAndLowerRight: 0b1111000011110000111100001111000000001111000011110000111100001111,
}

// pixel represents a character on screen used to draw part of an image.
type pixel struct {
	style   tcell.Style
	element rune // The block element.
}

// Image implements a widget that displays one image. The original image
// (specified with [Image.SetImage]) is resized according to the specified size
// (see [Image.SetSize]), using the specified number of colors (see
// [Image.SetColors]), while applying dithering if necessary (see
// [Image.SetDithering]).
//
// Images are approximated by graphical characters in the terminal. The
// resolution is therefore limited by the number and type of characters that can
// be drawn in the terminal and the colors available in the terminal. The
// quality of the final image also depends on the terminal's font and spacing
// settings, none of which are under the control of this package. Results may
// vary.
type Image struct {
	*Box

	// The image to be displayed. If nil, the widget will be empty.
	image image.Image

	// The size of the image. If a value is 0, the corresponding size is chosen
	// automatically based on the other size while preserving the image's aspect
	// ratio. If both are 0, the image uses as much space as possible. A
	// negative value represents a percentage, e.g. -50 means 50% of the
	// available space.
	width, height int

	// The number of colors to use. If 0, the number of colors is chosen based
	// on the terminal's capabilities.
	colors int

	// The dithering algorithm to use, one of the constants starting with
	// "ImageDithering".
	dithering int

	// The width of a terminal's cell divided by its height.
	aspectRatio float64

	// Horizontal and vertical alignment, one of the "Align" constants.
	alignHorizontal, alignVertical int

	// The text to be displayed before the image.
	label string

	// The label style.
	labelStyle tcell.Style

	// The screen width of the label area. A value of 0 means use the width of
	// the label text.
	labelWidth int

	// The actual image size (in cells) when it was drawn the last time.
	lastWidth, lastHeight int

	// The actual image (in cells) when it was drawn the last time. The size of
	// this slice is lastWidth * lastHeight, indexed by y*lastWidth + x.
	pixels []pixel

	// A callback function set by the Form class and called when the user leaves
	// this form item.
	finished func(tcell.Key)
}

// NewImage returns a new image widget with an empty image (use [Image.SetImage]
// to specify the image to be displayed). The image will use the widget's entire
// available space. The dithering algorithm is set to Floyd-Steinberg dithering.
// The terminal's cell aspect ratio defaults to 0.5.
func NewImage() *Image {
	return &Image{
		Box:             NewBox(),
		dithering:       DitheringFloydSteinberg,
		aspectRatio:     0.5,
		alignHorizontal: AlignCenter,
		alignVertical:   AlignCenter,
	}
}

// SetImage sets the image to be displayed. If nil, the widget will be empty.
func (i *Image) SetImage(image image.Image) *Image {
	i.image = image
	i.lastWidth, i.lastHeight = 0, 0
	return i
}

// SetSize sets the size of the image. Positive values refer to cells in the
// terminal. Negative values refer to a percentage of the available space (e.g.
// -50 means 50%). A value of 0 means that the corresponding size is chosen
// automatically based on the other size while preserving the image's aspect
// ratio. If both are 0, the image uses as much space as possible while still
// preserving the aspect ratio.
func (i *Image) SetSize(rows, columns int) *Image {
	i.width = columns
	i.height = rows
	return i
}

// SetColors sets the number of colors to use. This should be the number of
// colors supported by the terminal. If 0, the number of colors is chosen based
// on the TERM environment variable (which may or may not be reliable).
//
// Only the values 0, 2, 8, 256, and 16777216 ([TrueColor]) are supported. Other
// values will be rounded up to the next supported value, to a maximum of
// 16777216.
//
// The effect of using more colors than supported by the terminal is undefined.
func (i *Image) SetColors(colors int) *Image {
	i.colors = colors
	i.lastWidth, i.lastHeight = 0, 0
	return i
}

// GetColors returns the number of colors that will be used while drawing the
// image. This is one of the values listed in [Image.SetColors], except 0 which
// will be replaced by the actual number of colors used.
func (i *Image) GetColors() int {
	switch {
	case i.colors == 0:
		return availableColors
	case i.colors <= 2:
		return 2
	case i.colors <= 8:
		return 8
	case i.colors <= 256:
		return 256
	}
	return TrueColor
}

// SetDithering sets the dithering algorithm to use, one of the constants
// starting with "Dithering", for example [DitheringFloydSteinberg] (the
// default). Dithering is not applied when rendering in true-color.
func (i *Image) SetDithering(dithering int) *Image {
	i.dithering = dithering
	i.lastWidth, i.lastHeight = 0, 0
	return i
}

// SetAspectRatio sets the width of a terminal's cell divided by its height.
// You may change the default of 0.5 if your terminal / font has a different
// aspect ratio. This is used to calculate the size of the image if the
// specified width or height is 0. The function will panic if the aspect ratio
// is 0 or less.
func (i *Image) SetAspectRatio(aspectRatio float64) *Image {
	if aspectRatio <= 0 {
		panic("aspect ratio must be greater than 0")
	}
	i.aspectRatio = aspectRatio
	i.lastWidth, i.lastHeight = 0, 0
	return i
}

// SetAlign sets the vertical and horizontal alignment of the image within the
// widget's space. The possible values are [AlignTop], [AlignCenter], and
// [AlignBottom] for vertical alignment and [AlignLeft], [AlignCenter], and
// [AlignRight] for horizontal alignment. The default is [AlignCenter] for both
// (or [AlignTop] and [AlignLeft] if the image is part of a [Form]).
func (i *Image) SetAlign(vertical, horizontal int) *Image {
	i.alignHorizontal = horizontal
	i.alignVertical = vertical
	return i
}

// SetLabel sets the text to be displayed before the image.
func (i *Image) SetLabel(label string) *Image {
	i.label = label
	return i
}

// GetLabel returns the text to be displayed before the image.
func (i *Image) GetLabel() string {
	return i.label
}

// SetLabelWidth sets the screen width of the label. A value of 0 will cause the
// primitive to use the width of the label string.
func (i *Image) SetLabelWidth(width int) *Image {
	i.labelWidth = width
	return i
}

// GetFieldWidth returns this primitive's field width. This is the image's width
// or, if the width is 0 or less, the proportional width of the image based on
// its height as returned by [Image.GetFieldHeight]. If there is no image, 0 is
// returned.
func (i *Image) GetFieldWidth() int {
	if i.width <= 0 {
		if i.image == nil {
			return 0
		}
		bounds := i.image.Bounds()
		height := i.GetFieldHeight()
		return bounds.Dx() * height / bounds.Dy()
	}
	return i.width
}

// GetFieldHeight returns this primitive's field height. This is the image's
// height or 8 if the height is 0 or less.
func (i *Image) GetFieldHeight() int {
	if i.height <= 0 {
		return 8
	}
	return i.height
}

// SetDisabled sets whether or not the item is disabled / read-only.
func (i *Image) SetDisabled(disabled bool) FormItem {
	return i // Images are always read-only.
}

// SetFormAttributes sets attributes shared by all form items.
func (i *Image) SetFormAttributes(labelWidth int, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) FormItem {
	i.labelWidth = labelWidth
	i.backgroundColor = bgColor
	i.SetLabelStyle(tcell.StyleDefault.Foreground(labelColor).Background(bgColor))
	i.lastWidth, i.lastHeight = 0, 0
	return i
}

// SetLabelStyle sets the style of the label.
func (i *Image) SetLabelStyle(style tcell.Style) *Image {
	i.labelStyle = style
	return i
}

// GetLabelStyle returns the style of the label.
func (i *Image) GetLabelStyle() tcell.Style {
	return i.labelStyle
}

// SetFinishedFunc sets a callback invoked when the user leaves this form item.
func (i *Image) SetFinishedFunc(handler func(key tcell.Key)) FormItem {
	i.finished = handler
	return i
}

// Focus is called when this primitive receives focus.
func (i *Image) Focus(delegate func(p Primitive)) {
	// If we're part of a form, there's nothing the user can do here so we're
	// finished.
	if i.finished != nil {
		i.finished(-1)
		return
	}

	i.Box.Focus(delegate)
}

// render re-populates the [Image.pixels] slice based on the current settings,
// if [Image.lastWidth] and [Image.lastHeight] don't match the current image's
// size. It also sets the new image size in these two variables.
func (i *Image) render() {
	// If there is no image, there are no pixels.
	if i.image == nil {
		i.pixels = nil
		return
	}

	// Calculate the new (terminal-space) image size.
	bounds := i.image.Bounds()
	imageWidth, imageHeight := bounds.Dx(), bounds.Dy()
	if i.aspectRatio != 1.0 {
		imageWidth = int(float64(imageWidth) / i.aspectRatio)
	}
	width, height := i.width, i.height
	_, _, innerWidth, innerHeight := i.GetInnerRect()
	if i.labelWidth > 0 {
		innerWidth -= i.labelWidth
	} else {
		innerWidth -= TaggedStringWidth(i.label)
	}
	if innerWidth <= 0 {
		i.pixels = nil
		return
	}
	if width == 0 && height == 0 {
		// Use all available space.
		width, height = innerWidth, innerHeight
		if adjustedWidth := imageWidth * height / imageHeight; adjustedWidth < width {
			width = adjustedWidth
		} else {
			height = imageHeight * width / imageWidth
		}
	} else {
		// Turn percentages into absolute values.
		if width < 0 {
			width = innerWidth * -width / 100
		}
		if height < 0 {
			height = innerHeight * -height / 100
		}
		if width == 0 {
			// Adjust the width.
			width = imageWidth * height / imageHeight
		} else if height == 0 {
			// Adjust the height.
			height = imageHeight * width / imageWidth
		}
	}
	if width <= 0 || height <= 0 {
		i.pixels = nil
		return
	}

	// If nothing has changed, we're done.
	if i.lastWidth == width && i.lastHeight == height {
		return
	}
	i.lastWidth, i.lastHeight = width, height // This could still be larger than the available space but that's ok for now.

	// Generate the initial pixels by resizing the image (8x8 per cell).
	pixels := i.resize()

	// Turn them into block elements with background/foreground colors.
	i.stamp(pixels)
}

// resize resizes the image to the current size and returns the result as a
// slice of pixels. It is assumed that [Image.lastWidth] (w) and
// [Image.lastHeight] (h) are positive, non-zero values, and the slice has a
// size of 64*w*h, with each pixel being represented by 3 float64 values in the
// range of 0-1. The factor of 64 is due to the fact that we calculate 8x8
// pixels per cell.
func (i *Image) resize() [][3]float64 {
	// Because most of the time, we will be downsizing the image, we don't even
	// attempt to do any fancy interpolation. For each target pixel, we
	// calculate a weighted average of the source pixels using their coverage
	// area.

	bounds := i.image.Bounds()
	srcWidth, srcHeight := bounds.Dx(), bounds.Dy()
	tgtWidth, tgtHeight := i.lastWidth*8, i.lastHeight*8
	coverageWidth, coverageHeight := float64(tgtWidth)/float64(srcWidth), float64(tgtHeight)/float64(srcHeight)
	pixels := make([][3]float64, tgtWidth*tgtHeight)
	weights := make([]float64, tgtWidth*tgtHeight)
	for srcY := bounds.Min.Y; srcY < bounds.Max.Y; srcY++ {
		for srcX := bounds.Min.X; srcX < bounds.Max.X; srcX++ {
			r32, g32, b32, _ := i.image.At(srcX, srcY).RGBA()
			r, g, b := float64(r32)/0xffff, float64(g32)/0xffff, float64(b32)/0xffff

			// Iterate over all target pixels. Outer loop is Y.
			startY := float64(srcY-bounds.Min.Y) * coverageHeight
			endY := startY + coverageHeight
			fromY, toY := int(startY), int(endY)
			for tgtY := fromY; tgtY <= toY && tgtY < tgtHeight; tgtY++ {
				coverageY := 1.0
				if tgtY == fromY {
					coverageY -= math.Mod(startY, 1.0)
				}
				if tgtY == toY {
					coverageY -= 1.0 - math.Mod(endY, 1.0)
				}

				// Inner loop is X.
				startX := float64(srcX-bounds.Min.X) * coverageWidth
				endX := startX + coverageWidth
				fromX, toX := int(startX), int(endX)
				for tgtX := fromX; tgtX <= toX && tgtX < tgtWidth; tgtX++ {
					coverageX := 1.0
					if tgtX == fromX {
						coverageX -= math.Mod(startX, 1.0)
					}
					if tgtX == toX {
						coverageX -= 1.0 - math.Mod(endX, 1.0)
					}

					// Add a weighted contribution to the target pixel.
					index := tgtY*tgtWidth + tgtX
					coverage := coverageX * coverageY
					pixels[index][0] += r * coverage
					pixels[index][1] += g * coverage
					pixels[index][2] += b * coverage
					weights[index] += coverage
				}
			}
		}
	}

	// Normalize the pixels.
	for index, weight := range weights {
		if weight > 0 {
			pixels[index][0] /= weight
			pixels[index][1] /= weight
			pixels[index][2] /= weight
		}
	}

	return pixels
}

// stamp takes the pixels generated by [Image.resize] and populates the
// [Image.pixels] slice accordingly.
func (i *Image) stamp(resized [][3]float64) {
	// For each 8x8 pixel block, we find the best block element to represent it,
	// given the available colors.
	i.pixels = make([]pixel, i.lastWidth*i.lastHeight)
	colors := i.GetColors()
	for row := 0; row < i.lastHeight; row++ {
		for col := 0; col < i.lastWidth; col++ {
			// Calculate an error for each potential block element + color. Keep
			// the one with the lowest error.

			// Note that the values in "resize" may lie outside [0, 1] due to
			// the error distribution during dithering.

			minMSE := math.MaxFloat64 // Mean squared error.
			var final [64][3]float64  // The final pixel values.
			for element, bits := range blockElements {
				// Calculate the average color for the pixels covered by the set
				// bits and unset bits.
				var (
					bg, fg  [3]float64
					setBits float64
					bit     uint64 = 1
				)
				for y := 0; y < 8; y++ {
					for x := 0; x < 8; x++ {
						index := (row*8+y)*i.lastWidth*8 + (col*8 + x)
						if bits&bit != 0 {
							fg[0] += resized[index][0]
							fg[1] += resized[index][1]
							fg[2] += resized[index][2]
							setBits++
						} else {
							bg[0] += resized[index][0]
							bg[1] += resized[index][1]
							bg[2] += resized[index][2]
						}
						bit <<= 1
					}
				}
				for ch := 0; ch < 3; ch++ {
					fg[ch] /= setBits
					if fg[ch] < 0 {
						fg[ch] = 0
					} else if fg[ch] > 1 {
						fg[ch] = 1
					}
					bg[ch] /= 64 - setBits
					if bg[ch] < 0 {
						bg[ch] = 0
					}
					if bg[ch] > 1 {
						bg[ch] = 1
					}
				}

				// Quantize to the nearest acceptable color.
				for _, color := range []*[3]float64{&fg, &bg} {
					if colors == 2 {
						// Monochrome. The following weights correspond better
						// to human perception than the arithmetic mean.
						gray := 0.299*color[0] + 0.587*color[1] + 0.114*color[2]
						if gray < 0.5 {
							*color = [3]float64{0, 0, 0}
						} else {
							*color = [3]float64{1, 1, 1}
						}
					} else {
						for index, ch := range color {
							switch {
							case colors == 8:
								// Colors vary wildly for each terminal. Expect
								// suboptimal results.
								if ch < 0.5 {
									color[index] = 0
								} else {
									color[index] = 1
								}
							case colors == 256:
								color[index] = math.Round(ch*6) / 6
							}
						}
					}
				}

				// Calculate the error (and the final pixel values).
				var (
					mse         float64
					values      [64][3]float64
					valuesIndex int
				)
				bit = 1
				for y := 0; y < 8; y++ {
					for x := 0; x < 8; x++ {
						if bits&bit != 0 {
							values[valuesIndex] = fg
						} else {
							values[valuesIndex] = bg
						}
						index := (row*8+y)*i.lastWidth*8 + (col*8 + x)
						for ch := 0; ch < 3; ch++ {
							err := resized[index][ch] - values[valuesIndex][ch]
							mse += err * err
						}
						bit <<= 1
						valuesIndex++
					}
				}

				// Do we have a better match?
				if mse < minMSE {
					// Yes. Save it.
					minMSE = mse
					final = values
					index := row*i.lastWidth + col
					i.pixels[index].element = element
					i.pixels[index].style = tcell.StyleDefault.
						Foreground(tcell.NewRGBColor(int32(math.Min(255, fg[0]*255)), int32(math.Min(255, fg[1]*255)), int32(math.Min(255, fg[2]*255)))).
						Background(tcell.NewRGBColor(int32(math.Min(255, bg[0]*255)), int32(math.Min(255, bg[1]*255)), int32(math.Min(255, bg[2]*255))))
				}
			}

			// Check if there is a shade block which results in a smaller error.

			// What's the overall average color?
			var avg [3]float64
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					index := (row*8+y)*i.lastWidth*8 + (col*8 + x)
					for ch := 0; ch < 3; ch++ {
						avg[ch] += resized[index][ch] / 64
					}
				}
			}
			for ch := 0; ch < 3; ch++ {
				if avg[ch] < 0 {
					avg[ch] = 0
				} else if avg[ch] > 1 {
					avg[ch] = 1
				}
			}

			// Quantize and choose shade element.
			element := BlockFullBlock
			var fg, bg tcell.Color
			shades := []rune{' ', BlockLightShade, BlockMediumShade, BlockDarkShade, BlockFullBlock}
			if colors == 2 {
				// Monochrome.
				gray := 0.299*avg[0] + 0.587*avg[1] + 0.114*avg[2] // See above for details.
				shade := int(math.Round(gray * 4))
				element = shades[shade]
				for ch := 0; ch < 3; ch++ {
					avg[ch] = float64(shade) / 4
				}
				bg = tcell.ColorBlack
				fg = tcell.ColorWhite
			} else if colors == TrueColor {
				// True color.
				fg = tcell.NewRGBColor(int32(math.Min(255, avg[0]*255)), int32(math.Min(255, avg[1]*255)), int32(math.Min(255, avg[2]*255)))
				bg = fg
			} else {
				// 8 or 256 colors.
				steps := 1.0
				if colors == 256 {
					steps = 6.0
				}
				var (
					lo, hi, pos [3]float64
					shade       float64
				)
				for ch := 0; ch < 3; ch++ {
					lo[ch] = math.Floor(avg[ch]*steps) / steps
					hi[ch] = math.Ceil(avg[ch]*steps) / steps
					if r := hi[ch] - lo[ch]; r > 0 {
						pos[ch] = (avg[ch] - lo[ch]) / r
						if math.Abs(pos[ch]-0.5) < math.Abs(shade-0.5) {
							shade = pos[ch]
						}
					}
				}
				shade = math.Round(shade * 4)
				element = shades[int(shade)]
				shade /= 4
				for ch := 0; ch < 3; ch++ { // Find the closest channel value.
					best := math.Abs(avg[ch] - (lo[ch] + (hi[ch]-lo[ch])*shade)) // Start shade from lo to hi.
					if value := math.Abs(avg[ch] - (hi[ch] - (hi[ch]-lo[ch])*shade)); value < best {
						best = value // Swap lo and hi.
						lo[ch], hi[ch] = hi[ch], lo[ch]
					}
					if value := math.Abs(avg[ch] - lo[ch]); value < best {
						best = value // Use lo.
						hi[ch] = lo[ch]
					}
					if value := math.Abs(avg[ch] - hi[ch]); value < best {
						lo[ch] = hi[ch] // Use hi.
					}
					avg[ch] = lo[ch] + (hi[ch]-lo[ch])*shade // Quantize.
				}
				bg = tcell.NewRGBColor(int32(math.Min(255, lo[0]*255)), int32(math.Min(255, lo[1]*255)), int32(math.Min(255, lo[2]*255)))
				fg = tcell.NewRGBColor(int32(math.Min(255, hi[0]*255)), int32(math.Min(255, hi[1]*255)), int32(math.Min(255, hi[2]*255)))
			}

			// Calculate the error (and the final pixel values).
			var (
				mse         float64
				values      [64][3]float64
				valuesIndex int
			)
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					index := (row*8+y)*i.lastWidth*8 + (col*8 + x)
					for ch := 0; ch < 3; ch++ {
						err := resized[index][ch] - avg[ch]
						mse += err * err
					}
					values[valuesIndex] = avg
					valuesIndex++
				}
			}

			// Is this shade element better than the block element?
			if mse < minMSE {
				// Yes. Save it.
				final = values
				index := row*i.lastWidth + col
				i.pixels[index].element = element
				i.pixels[index].style = tcell.StyleDefault.Foreground(fg).Background(bg)
			}

			// Apply dithering.
			if colors < TrueColor && i.dithering == DitheringFloydSteinberg {
				// The dithering mask determines how the error is distributed.
				// Each element has three values: dx, dy, and weight (in 16th).
				var mask = [4][3]int{
					{1, 0, 7},
					{-1, 1, 3},
					{0, 1, 5},
					{1, 1, 1},
				}

				// We dither the 8x8 block as a 2x2 block, transferring errors
				// to its 2x2 neighbors.
				for ch := 0; ch < 3; ch++ {
					for y := 0; y < 2; y++ {
						for x := 0; x < 2; x++ {
							// What's the error for this 4x4 block?
							var err float64
							for dy := 0; dy < 4; dy++ {
								for dx := 0; dx < 4; dx++ {
									err += (final[(y*4+dy)*8+(x*4+dx)][ch] - resized[(row*8+(y*4+dy))*i.lastWidth*8+(col*8+(x*4+dx))][ch]) / 16
								}
							}

							// Distribute it to the 2x2 neighbors.
							for _, dist := range mask {
								for dy := 0; dy < 4; dy++ {
									for dx := 0; dx < 4; dx++ {
										targetX, targetY := (x+dist[0])*4+dx, (y+dist[1])*4+dy
										if targetX < 0 || col*8+targetX >= i.lastWidth*8 || targetY < 0 || row*8+targetY >= i.lastHeight*8 {
											continue
										}
										resized[(row*8+targetY)*i.lastWidth*8+(col*8+targetX)][ch] -= err * float64(dist[2]) / 16
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// Draw draws this primitive onto the screen.
func (i *Image) Draw(screen tcell.Screen) {
	i.DrawForSubclass(screen, i)

	// Regenerate image if necessary.
	i.render()

	// Draw label.
	viewX, viewY, viewWidth, viewHeight := i.GetInnerRect()
	_, labelBg, _ := i.labelStyle.Decompose()
	if i.labelWidth > 0 {
		labelWidth := i.labelWidth
		if labelWidth > viewWidth {
			labelWidth = viewWidth
		}
		printWithStyle(screen, i.label, viewX, viewY, 0, labelWidth, AlignLeft, i.labelStyle, labelBg == tcell.ColorDefault)
		viewX += labelWidth
		viewWidth -= labelWidth
	} else {
		_, _, drawnWidth := printWithStyle(screen, i.label, viewX, viewY, 0, viewWidth, AlignLeft, i.labelStyle, labelBg == tcell.ColorDefault)
		viewX += drawnWidth
		viewWidth -= drawnWidth
	}

	// Determine image placement.
	x, y, width, height := viewX, viewY, i.lastWidth, i.lastHeight
	if i.alignHorizontal == AlignCenter {
		x += (viewWidth - width) / 2
	} else if i.alignHorizontal == AlignRight {
		x += viewWidth - width
	}
	if i.alignVertical == AlignCenter {
		y += (viewHeight - height) / 2
	} else if i.alignVertical == AlignBottom {
		y += viewHeight - height
	}

	// Draw the image.
	for row := 0; row < height; row++ {
		if y+row < viewY || y+row >= viewY+viewHeight {
			continue
		}
		for col := 0; col < width; col++ {
			if x+col < viewX || x+col >= viewX+viewWidth {
				continue
			}

			index := row*width + col
			screen.SetContent(x+col, y+row, i.pixels[index].element, nil, i.pixels[index].style)
		}
	}
}
