package main

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	"log"
	"math/rand"
	"sync"
	"time"

	"chessGo/bots"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/notnil/chess"
)

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
	bots         map[string]bots.ChessBot
	currentBot   bots.ChessBot
	botMutex     sync.RWMutex
}

func NewGame() *Game {
	screenWidth, screenHeight = ebiten.ScreenSizeInFullscreen()
	boardHeight := screenHeight - 80
	squareSize = boardHeight / 8
	if screenWidth/8 < squareSize {
		squareSize = screenWidth / 8
	}

	boardWidth := squareSize * 8
	g := &Game{
		pieces:       make(map[chess.Piece]*ebiten.Image),
		bots:         createBots(),
		boardOffsetX: (screenWidth - boardWidth) / 2,
		boardOffsetY: (screenHeight - boardHeight) / 2,
	}
	g.currentBot = g.bots["Newborn"]
	g.loadPieceImages()
	return g
}

func createBots() map[string]bots.ChessBot {
	return map[string]bots.ChessBot{
		"Newborn":      bots.NewNewbornBot(),
		"Beginner":     bots.NewMinimaxBot(2, 2*time.Second, "Beginner"),
		"Intermediate": bots.NewMinimaxBot(3, 5*time.Second, "Intermediate"),
		"Advanced":     bots.NewMinimaxBot(4, 10*time.Second, "Advanced"),
		"Expert":       bots.NewMinimaxBot(5, 15*time.Second, "Expert"),
	}
}

type NamedMinimaxBot struct {
	*bots.MinimaxBot
	name string
}

func NewNamedMinimaxBot(depth int, timeLimit time.Duration, name string) *NamedMinimaxBot {
	return &NamedMinimaxBot{
		MinimaxBot: bots.NewMinimaxBot(depth, timeLimit, name), // Добавлен третий аргумент name
		name:       name,
	}
}

func (b *NamedMinimaxBot) Name() string {
	return b.name
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
			log.Printf("Warning: failed to load image %s: %v", filename, err)
			continue
		}

		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			log.Printf("Warning: failed to decode image %s: %v", filename, err)
			continue
		}

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

	// Обработка смены бота по клавише B
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		g.switchBot()
	}

	// Обработка хода игрока
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
				if err := g.chessGame.Move(move); err == nil {
					g.botThinking = true
					go g.makeBotMove()
				}
			}
		}
		g.selected = 0
		g.dragging = nil
	}

	if !g.botThinking && g.gameStarted && g.chessGame.Position().Turn() != g.playerColor {
		g.botThinking = true
		go func() {
			time.Sleep(300 * time.Millisecond)
			g.makeBotMove()
		}()
	}

	return nil
}

func (g *Game) switchBot() {
	g.botMutex.Lock()
	defer g.botMutex.Unlock()

	// Получаем список имен ботов
	var botNames []string
	for name := range g.bots {
		botNames = append(botNames, name)
	}

	// Находим текущего бота
	currentIndex := -1
	for i, name := range botNames {
		if g.bots[name] == g.currentBot {
			currentIndex = i
			break
		}
	}

	// Выбираем следующего бота
	if currentIndex >= 0 {
		nextIndex := (currentIndex + 1) % len(botNames)
		g.currentBot = g.bots[botNames[nextIndex]]
	} else if len(botNames) > 0 {
		g.currentBot = g.bots[botNames[0]]
	}
}

func (g *Game) startGame() {
	g.chessGame = chess.NewGame()
	g.gameStarted = true
	if g.playerColor == chess.Black {
		g.botThinking = true
		go func() {
			time.Sleep(500 * time.Millisecond)
			g.makeBotMove()
		}()
	}
}

func (g *Game) makeBotMove() {
	g.botMutex.RLock()
	defer g.botMutex.RUnlock()
	defer func() { g.botThinking = false }()

	if g.currentBot == nil || g.chessGame == nil || g.chessGame.Outcome() != chess.NoOutcome {
		return
	}

	// Даем боту больше времени на размышления в зависимости от сложности
	timeLimit := time.Second
	if minimaxBot, ok := g.currentBot.(*bots.MinimaxBot); ok {
		timeLimit = time.Duration(minimaxBot.Depth) * time.Second
	}

	resultChan := make(chan *chess.Move, 1)

	go func() {
		resultChan <- g.currentBot.BestMove(g.chessGame)
	}()

	select {
	case move := <-resultChan:
		if move != nil {
			g.chessGame.Move(move)
		}
	case <-time.After(timeLimit):
		// Если время вышло, делаем случайный ход
		moves := g.chessGame.ValidMoves()
		if len(moves) > 0 {
			g.chessGame.Move(moves[rand.Intn(len(moves))])
		}
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
	if g == nil || screen == nil {
		return
	}

	// Получаем имя текущего бота
	g.botMutex.RLock()
	botName := "No bot selected"
	if g.currentBot != nil {
		botName = g.currentBot.Name()
	}
	g.botMutex.RUnlock()

	// Рисуем кнопку смены бота
	botBtn := ebiten.NewImage(200, 40)
	botBtn.Fill(color.RGBA{70, 70, 70, 255})
	ebitenutil.DebugPrintAt(botBtn, "Bot: "+botName, 10, 10)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(screenWidth-220), 20)
	screen.DrawImage(botBtn, op)

	// Обработка клика по кнопке смены бота
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if x > screenWidth-220 && x < screenWidth-20 && y > 20 && y < 60 {
			g.switchBot()
		}
	}

	if !g.gameStarted {
		ebitenutil.DebugPrintAt(screen, "Шахматы на Go", screenWidth/2-70, screenHeight/2-50)
		ebitenutil.DebugPrintAt(screen, "Выберите цвет фигур:", screenWidth/2-100, screenHeight/2)

		whiteBtn := ebiten.NewImage(200, 60)
		whiteBtn.Fill(color.RGBA{200, 200, 200, 255})
		ebitenutil.DebugPrintAt(whiteBtn, "Играть белыми", 50, 20)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(screenWidth/2-200-20), float64(screenHeight/2+100))
		screen.DrawImage(whiteBtn, op)

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
			clr := color.RGBA{240, 217, 181, 255}
			if (x+y)%2 == 1 {
				clr = color.RGBA{181, 136, 99, 255}
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
	ebiten.SetWindowTitle("Шахматы на Go")
	ebiten.SetWindowResizable(true)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
