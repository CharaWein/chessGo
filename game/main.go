package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"log"
	"sync"
	"time"

	"chessGo/bots"

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
	bots         map[string]bots.ChessBot
	currentBot   bots.ChessBot
	botMutex     sync.Mutex
}

type ChessBot interface {
	BestMove(game *chess.Game) *chess.Move
	Name() string
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
		pieces:       make(map[chess.Piece]*ebiten.Image), // Инициализируем map
		bots:         make(map[string]bots.ChessBot),
		boardOffsetX: (screenWidth - boardWidth) / 2,
		boardOffsetY: (screenHeight - boardHeight) / 2,
	}

	// Инициализация ботов
	g.bots["Newborn"] = bots.NewNewbornBot()
	g.bots["minimax3"] = bots.NewMinimaxBot(3, 5*time.Second)
	g.bots["minimax5"] = bots.NewMinimaxBot(5, 10*time.Second)

	// Устанавливаем бота по умолчанию
	g.currentBot = g.bots["Newborn"]

	g.loadPieceImages()
	return g
}

func (g *Game) loadPieceImages() {
	if g.pieces == nil {
		g.pieces = make(map[chess.Piece]*ebiten.Image)
	}

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

	// Создаем fallback-изображения для отсутствующих фигур
	createFallback := func(clr color.Color, symbol rune) *ebiten.Image {
		img := ebiten.NewImage(squareSize, squareSize)
		img.Fill(clr)
		// Можно добавить символ фигуры
		return img
	}

	// Проверяем, все ли фигуры загружены
	requiredPieces := []chess.Piece{
		chess.WhiteKing, chess.WhiteQueen, // ... все остальные фигуры ...
	}

	for _, piece := range requiredPieces {
		if _, exists := g.pieces[piece]; !exists {
			if piece.Color() == chess.White {
				g.pieces[piece] = createFallback(color.White, 'K')
			} else {
				g.pieces[piece] = createFallback(color.Black, 'K')
			}
		}
	}
}

func (g *Game) Update() error {
	if g == nil {
		return nil
	}

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
					go g.makeBotMove() // Запускаем ход бота
				}
			}
		}
		g.selected = 0
		g.dragging = nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		g.botMutex.Lock()
		defer g.botMutex.Unlock()

		// Простой цикл переключения между ботами
		var names []string
		for name := range g.bots {
			names = append(names, name)
		}

		for i, name := range names {
			if g.currentBot.Name() == g.bots[name].Name() {
				next := (i + 1) % len(names)
				g.currentBot = g.bots[names[next]]
				break
			}
		}
	}

	if !g.botThinking && g.gameStarted && g.chessGame.Position().Turn() != g.playerColor {
		g.botThinking = true
		go func() {
			time.Sleep(300 * time.Millisecond) // Небольшая задержка
			g.makeBotMove()
		}()
	}

	return nil
}

func (g *Game) startGame() {
	g.chessGame = chess.NewGame()
	g.gameStarted = true
	if g.playerColor == chess.Black {
		g.botThinking = true
		go func() {
			time.Sleep(500 * time.Millisecond) // Небольшая задержка для плавности
			g.makeBotMove()
		}()
	}
}

func (g *Game) makeBotMove() {
	g.botMutex.Lock()
	defer g.botMutex.Unlock()

	if g.currentBot == nil || g.chessGame == nil {
		g.botThinking = false
		return
	}

	// Проверяем, должен ли бот ходить
	if g.chessGame.Position().Turn() != g.playerColor && g.chessGame.Outcome() == chess.NoOutcome {
		move := g.currentBot.BestMove(g.chessGame)
		if move != nil {
			if err := g.chessGame.Move(move); err != nil {
				log.Printf("Bot move error: %v", err)
			}
		}
	}
	g.botThinking = false
}

func findMove(game *chess.Game, from, to chess.Square) *chess.Move {
	if game == nil {
		return nil
	}
	for _, m := range game.ValidMoves() {
		if m.S1() == from && m.S2() == to {
			return m
		}
	}
	return nil
}

func createBots() map[string]bots.ChessBot {
	return map[string]bots.ChessBot{
		"Newborn":  bots.NewNewbornBot(),
		"minimax3": bots.NewMinimaxBot(3, 5*time.Second),
		"minimax5": bots.NewMinimaxBot(5, 10*time.Second),
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g == nil || screen == nil {
		return
	}

	// Отображение текущего бота
	botInfo := fmt.Sprintf("Бот: %s", g.currentBot.Name())
	ebitenutil.DebugPrintAt(screen, botInfo, screenWidth-200, 20)

	// Кнопка смены бота (при клике)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if x > screenWidth-200 && x < screenWidth-20 && y > 20 && y < 50 {
			// Циклически меняем ботов
			botsList := []string{"Newborn", "minimax3", "minimax5"}
			current := ""
			for i, name := range botsList {
				if g.currentBot.Name() == g.bots[name].Name() {
					current = botsList[(i+1)%len(botsList)]
					break
				}
			}
			if current == "" {
				current = botsList[0]
			}
			g.currentBot = g.bots[current]
		}
	}

	g.botMutex.Lock()
	botName := "No bot selected"
	if g.currentBot != nil {
		botName = g.currentBot.Name()
	}
	g.botMutex.Unlock()

	ebitenutil.DebugPrintAt(screen, "Current bot: "+botName, 20, screenHeight-40)

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

	if g.botThinking {
		ebitenutil.DebugPrintAt(screen, "Бот думает...", screenWidth/2-50, screenHeight-30)
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
