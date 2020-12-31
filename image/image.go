// Package image is a go library that creates images from board positions
package image

import (
	"fmt"
	"image/color"
	"io"
	"strings"

	svg "github.com/ajstarks/svgo"
	"github.com/notnil/chess"
	"github.com/notnil/chess/image/internal"
)

type Format interface {
	Init() error
	DrawSquare(xIdx, yIdx int, col color.Color) error
	DrawPiece(xIdx, yIdx int, piece chess.Piece) error
}

type svgFormat struct {
	canvas            *svg.SVG
	sqWidth, sqHeight int
	width, height     int
}

func (s *svgFormat) Init() error {
	s.canvas.Start(s.width, s.height)
	s.canvas.Rect(0, 0, s.width, s.height)
}

func (s *svgFormat) DrawSquare(xIdx, yIdx int, col color.Color) error {
	canvas.Rect(x, y, sqWidth, sqHeight, "fill: "+colorToHex(c))
}

func (s *svgFormat) DrawPiece(xIdx, yIdx int, piece chess.Piece) error {

}

type Option func(*config, *encoder)

// SVG writes the board SVG representation into the writer.
// An error is returned if there is there is an error writing data.
// SVG also takes options which can customize the image output.
func SVG(w io.Writer, b *chess.Board, opts ...Option) error {
	e := new(opts)
	boardWidth, boardHeight := e.boardSize()
	return e.Encode(b, &svgFormat{
		canvas:   svg.New(w),
		sqWidth:  e.cfg.sqWidth,
		sqHeight: e.cfg.sqHeight,
		width:    boardWidth,
		height:   boardHeight,
	})
}

// Flip rotates the board 180 degrees, placing the player in black at the
// bottom.
func Flip() Option {
	return func(cfg *config, _ *encoder) {
		cfg.flip = true
	}
}

// SquareColors is designed to be used as an optional argument
// to the SVG function.  It changes the default light and
// dark square colors to the colors given.
func SquareColors(light, dark color.Color) Option {
	return func(cfg *config, _ *encoder) {
		cfg.light = light
		cfg.dark = dark
	}
}

// MarkSquares is designed to be used as an optional argument
// to the SVG function.  It marks the given squares with the
// color.  A possible usage includes marking squares of the
// previous move.
func MarkSquares(c color.Color, sqs ...chess.Square) Option {
	return func(_ *config, e *encoder) {
		for _, sq := range sqs {
			e.marks[sq] = c
		}
	}
}

// config encompasses static parameters about how the board should be rendered.
type config struct {
	sqWidth  int
	sqHeight int
	flip     bool
	light    color.Color
	dark     color.Color
}

// encoder encodes chess boards into images.
type encoder struct {
	marks map[chess.Square]color.Color
	cfg   *config
}

func (e *encoder) boardSize() (int, int) {
	return e.cfg.sqWidth * 8, e.cfg.sqHeight * 8
}

// New returns an encoder that writes to the given writer.
// New also takes options which can customize the image
// output.
func new(options []Option) *encoder {
	cfg := &config{
		sqWidth:  45,
		sqHeight: 45,
		flip:     false,
		light:    color.RGBA{235, 209, 166, 1},
		dark:     color.RGBA{165, 117, 81, 1},
	}

	for _, op := range options {
		op(cfg, &encoder{})
	}

	e := &encoder{
		marks: map[chess.Square]color.Color{},
	}

	for _, op := range options {
		op(cfg, e)
	}
	return e
}

var (
	orderOfRanks = []chess.Rank{chess.Rank8, chess.Rank7, chess.Rank6, chess.Rank5, chess.Rank4, chess.Rank3, chess.Rank2, chess.Rank1}
	orderOfFiles = []chess.File{chess.FileA, chess.FileB, chess.FileC, chess.FileD, chess.FileE, chess.FileF, chess.FileG, chess.FileH}
)

// EncodeSVG writes the board SVG representation into
// the Encoder's writer.  An error is returned if there
// is there is an error writing data.
func (e *encoder) Encode(b *chess.Board, f Format) error {
	boardWidth, boardHeight := e.boardSize()

	boardMap := b.SquareMap()

	if err := f.Init(); err != nil {
		fmt.Errorf("failed to init output formatter: %w", err)
	}

	for i := 0; i < 64; i++ {
		sq := chess.Square(i)
		x, y := xyForSquare(sq)
		// draw square
		c := e.colorForSquare(sq)
		f.DrawSquare(x, y, c)
		markColor, ok := e.marks[sq]
		if ok {
			canvas.Rect(x, y, sqWidth, sqHeight, "fill-opacity:0.2;fill: "+colorToHex(markColor))
		}
		// draw piece
		p := boardMap[sq]
		if p != chess.NoPiece {
			xml := pieceXML(x, y, p)
			if _, err := io.WriteString(canvas.Writer, xml); err != nil {
				return err
			}
		}
		// draw rank text on file A
		txtColor := e.colorForText(sq)
		if sq.File() == chess.FileA {
			style := "font-size:11px;fill: " + colorToHex(txtColor)
			canvas.Text(x+(sqWidth*1/20), y+(sqHeight*5/20), sq.Rank().String(), style)
		}
		// draw file text on rank 1
		if sq.Rank() == chess.Rank1 {
			style := "text-anchor:end;font-size:11px;fill: " + colorToHex(txtColor)
			canvas.Text(x+(sqWidth*19/20), y+sqHeight-(sqHeight*1/15), sq.File().String(), style)
		}
	}
	canvas.End()
	return nil
}

func (e *encoder) colorForSquare(sq chess.Square) color.Color {
	sqSum := int(sq.File()) + int(sq.Rank())
	if sqSum%2 == 0 {
		return e.dark
	}
	return e.light
}

func (e *encoder) colorForText(sq chess.Square) color.Color {
	sqSum := int(sq.File()) + int(sq.Rank())
	if sqSum%2 == 0 {
		return e.light
	}
	return e.dark
}

func xyForSquare(sq chess.Square) (x, y int) {
	fileIndex := int(sq.File())
	rankIndex := 7 - int(sq.Rank())
	return fileIndex, rankIndex
}

func colorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", uint8(float64(r)+0.5), uint8(float64(g)*1.0+0.5), uint8(float64(b)*1.0+0.5))
}

func pieceXML(x, y int, p chess.Piece) string {
	fileName := fmt.Sprintf("pieces/%s%s.svg", p.Color().String(), pieceTypeMap[p.Type()])
	svgStr := string(internal.MustAsset(fileName))
	old := `<svg xmlns="http://www.w3.org/2000/svg" version="1.1" width="45" height="45">`
	new := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" version="1.1" width="360" height="360" viewBox="%d %d 360 360">`, (-1 * x), (-1 * y))
	return strings.Replace(svgStr, old, new, 1)
}

var (
	pieceTypeMap = map[chess.PieceType]string{
		chess.King:   "K",
		chess.Queen:  "Q",
		chess.Rook:   "R",
		chess.Bishop: "B",
		chess.Knight: "N",
		chess.Pawn:   "P",
	}
)
