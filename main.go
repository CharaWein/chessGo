package main

import (
	"bytes"
	"embed"
	_ "embed"
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/notnil/chess"
)

// Размеры фигур (160x160)
const pieceSize = 160

var (
	screenWidth  int
	screenHeight int
	squareSize   int
)

//go:embed assets/*
var assets embed.FS

type Game struct {
	chessGame    *chess.Game
	pieces       map[chess.Piece]*ebiten.Image
	selected     chess.Square
	dragging     *chess.Piece
	dragX, dragY int
	playerColor  chess.Color
	gameStarted  bool
	botThinking  bool
	boardOffsetX int
	boardOffsetY int
}

func NewGame() *Game {
	// Получаем размеры экрана
	screenWidth, screenHeight = ebiten.ScreenSizeInFullscreen()

	// Вычисляем размер клетки (оставляем место для информации сверху)
	boardHeight := screenHeight - 80
	squareSize = boardHeight / 8
	if screenWidth/8 < squareSize {
		squareSize = screenWidth / 8
	}

	// Центрируем доску
	boardWidth := squareSize * 8
	g := &Game{
		pieces:       make(map[chess.Piece]*ebiten.Image),
		boardOffsetX: (screenWidth - boardWidth) / 2,
		boardOffsetY: (screenHeight - boardHeight) / 2,
	}
	g.loadPieceImages()
	return g
}

func (g *Game) loadPieceImages() {
	pieceAssets := map[chess.Piece]string{
		chess.WhiteKing:   "white_king.png",
		chess.WhiteQueen:  "white_queen.png",
		chess.WhiteRook:   "white_rook.png",
		chess.WhiteBishop: "white_bishop.png",
		chess.WhiteKnight: "white_knight.png",
		chess.WhitePawn:   "white_pawn.png",
		chess.BlackKing:   "black_king.png",
		chess.BlackQueen:  "black_queen.png",
		chess.BlackRook:   "black_rook.png",
		chess.BlackBishop: "black_bishop.png",
		chess.BlackKnight: "black_knight.png",
		chess.BlackPawn:   "black_pawn.png",
	}

	for piece, filename := range pieceAssets {
		data, err := assets.ReadFile("assets/" + filename)
		if err != nil {
			log.Fatalf("Failed to load image %s: %v", filename, err)
		}

		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			log.Fatalf("Failed to decode image %s: %v", filename, err)
		}

		// Масштабируем изображение под размер клетки
		scaledImg := ebiten.NewImage(squareSize, squareSize)
		op := &ebiten.DrawImageOptions{}
		scale := float64(squareSize) / float64(pieceSize)
		op.GeoM.Scale(scale, scale)
		scaledImg.DrawImage(ebiten.NewImageFromImage(img), op)
		g.pieces[piece] = scaledImg
	}
}

func (g *Game) Update() error {
	if !g.gameStarted {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			btnWidth := 200
			btnHeight := 60
			btnY := screenHeight/2 + 100

			if y > btnY && y < btnY+btnHeight {
				if x > screenWidth/2-btnWidth-20 && x < screenWidth/2-btnWidth-20+btnWidth {
					g.playerColor = chess.White
					g.startGame()
				} else if x > screenWidth/2+20 && x < screenWidth/2+20+btnWidth {
					g.playerColor = chess.Black
					g.startGame()
				}
			}
		}
		return nil
	}

	if g.chessGame.Position().Turn() == g.playerColor && !g.botThinking {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			x -= g.boardOffsetX
			y -= g.boardOffsetY
			if x >= 0 && x < squareSize*8 && y >= 0 && y < squareSize*8 {
				file := x / squareSize
				rank := 7 - y/squareSize
				sq := chess.Square(file + rank*8)
				piece := g.chessGame.Position().Board().Piece(sq)
				if piece != chess.NoPiece && piece.Color() == g.playerColor {
					g.selected = sq
					g.dragging = &piece
					g.dragX, g.dragY = x+g.boardOffsetX, y+g.boardOffsetY
				}
			}
		}

		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && g.dragging != nil {
			x, y := ebiten.CursorPosition()
			x -= g.boardOffsetX
			y -= g.boardOffsetY
			if x >= 0 && x < squareSize*8 && y >= 0 && y < squareSize*8 {
				file := x / squareSize
				rank := 7 - y/squareSize
				target := chess.Square(file + rank*8)
				move := findMove(g.chessGame, g.selected, target)
				if move != nil {
					g.chessGame.Move(move)
					g.botThinking = true
					go g.makeBotMove()
				}
			}
			g.selected = 0
			g.dragging = nil
		}
	}

	return nil
}

func (g *Game) startGame() {
	g.chessGame = chess.NewGame()
	g.gameStarted = true
	if g.playerColor == chess.Black {
		g.botThinking = true
		go g.makeBotMove()
	}
}

func (g *Game) makeBotMove() {
	moves := g.chessGame.ValidMoves()
	if len(moves) > 0 {
		g.chessGame.Move(moves[0])
	}
	g.botThinking = false
}

func findMove(game *chess.Game, from, to chess.Square) *chess.Move {
	for _, m := range game.ValidMoves() {
		if m.S1() == from && m.S2() == to {
			return m
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if !g.gameStarted {
		// Экран выбора цвета
		ebitenutil.DebugPrintAt(screen, "Шахматы на Go", screenWidth/2-70, screenHeight/2-50)
		ebitenutil.DebugPrintAt(screen, "Выберите цвет фигур:", screenWidth/2-100, screenHeight/2)

		// Кнопка "Белые"
		whiteBtn := ebiten.NewImage(200, 60)
		whiteBtn.Fill(color.RGBA{200, 200, 200, 255})
		ebitenutil.DebugPrintAt(whiteBtn, "Играть белыми", 50, 20)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(screenWidth/2-200-20), float64(screenHeight/2+100))
		screen.DrawImage(whiteBtn, op)

		// Кнопка "Черные"
		blackBtn := ebiten.NewImage(200, 60)
		blackBtn.Fill(color.RGBA{50, 50, 50, 255})
		ebitenutil.DebugPrintAt(blackBtn, "Играть черными", 50, 20)
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(screenWidth/2+20), float64(screenHeight/2+100))
		screen.DrawImage(blackBtn, op)
		return
	}

	// Рисуем доску
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			clr := color.RGBA{240, 217, 181, 255} // светлые клетки
			if (x+y)%2 == 1 {
				clr = color.RGBA{181, 136, 99, 255} // темные клетки
			}
			rect := ebiten.NewImage(squareSize, squareSize)
			rect.Fill(clr)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x*squareSize+g.boardOffsetX), float64(y*squareSize+g.boardOffsetY))
			screen.DrawImage(rect, op)
		}
	}

	// Рисуем фигуры
	board := g.chessGame.Position().Board()
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			sq := chess.Square(x + (7-y)*8)
			piece := board.Piece(sq)
			if piece != chess.NoPiece && (g.dragging == nil || sq != g.selected) {
				img := g.pieces[piece]
				if img != nil {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(
						float64(x*squareSize+g.boardOffsetX),
						float64(y*squareSize+g.boardOffsetY),
					)
					screen.DrawImage(img, op)
				}
			}
		}
	}

	// Рисуем перетаскиваемую фигуру
	if g.dragging != nil {
		img := g.pieces[*g.dragging]
		if img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(
				float64(g.dragX)-float64(squareSize)/2,
				float64(g.dragY)-float64(squareSize)/2,
			)
			screen.DrawImage(img, op)
		}
	}

	// Статус игры
	status := "Ваш ход"
	if g.botThinking {
		status = "Бот думает..."
	} else if g.chessGame.Position().Turn() != g.playerColor {
		status = "Ход бота"
	}
	ebitenutil.DebugPrintAt(screen, status, 20, 20)

	// Информация о игре
	outcome := g.chessGame.Outcome().String()
	if outcome != "*" {
		ebitenutil.DebugPrintAt(screen, "Результат: "+outcome, screenWidth/2-50, 20)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	game := NewGame()
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Шахматы на Go - Полноэкранный режим")
	ebiten.SetWindowResizable(true)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
