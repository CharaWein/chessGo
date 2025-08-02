package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/notnil/chess"
)

type ChessBot struct {
	depth int
}

func NewChessBot(depth int) *ChessBot {
	return &ChessBot{depth: depth}
}

func (bot *ChessBot) BestMove(game *chess.Game) *chess.Move {
	validMoves := game.ValidMoves()
	if len(validMoves) == 0 {
		return nil
	}
	return validMoves[0] // Простой бот - просто берет первый доступный ход
}

func printBoard(game *chess.Game) {
	fmt.Println("\n   a b c d e f g h")
	fmt.Println("  +-----------------+")

	board := game.Position().Board()
	for rank := 7; rank >= 0; rank-- {
		fmt.Printf("%d | ", rank+1)
		for file := 0; file < 8; file++ {
			square := chess.Square(file + rank*8)
			piece := board.Piece(square)
			if piece == chess.NoPiece {
				fmt.Print(". ")
			} else {
				fmt.Printf("%s ", piece.String())
			}
		}
		fmt.Printf("| %d\n", rank+1)
	}

	fmt.Println("  +-----------------+")
	fmt.Println("   a b c d e f g h\n")
}

func main() {
	game := chess.NewGame()
	bot := NewChessBot(3)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Шахматный бот на Go")
	fmt.Println("Введите ход (например: e4, Nf3, Bxc4, O-O) или 'quit' для выхода")

	for game.Outcome() == chess.NoOutcome {
		printBoard(game)

		if game.Position().Turn() == chess.White {
			fmt.Print("Ваш ход (белые): ")
			scanner.Scan()
			input := strings.TrimSpace(scanner.Text())

			if input == "quit" {
				return
			}

			// Пробуем разные варианты нотации
			var move *chess.Move
			var err error

			// 1. Сначала пробуем стандартную нотацию (Bxf7)
			move, err = chess.AlgebraicNotation{}.Decode(game.Position(), input)

			// 2. Если не получилось, пробуем без символа взятия (Bf7)
			if err != nil {
				cleanInput := strings.ReplaceAll(input, "x", "")
				move, err = chess.AlgebraicNotation{}.Decode(game.Position(), cleanInput)
			}

			// 3. Если все еще не получилось, пробуем координатный формат (e6f7)
			if err != nil && len(input) == 4 {
				move = findMoveByCoordinates(game, input[0:2], input[2:4])
			}

			if err != nil || move == nil {
				fmt.Println("Неверный ход. Попробуйте еще раз.")
				fmt.Println("Для взятия используйте: Bxf7 или Bf7")
				continue
			}

			if err := game.Move(move); err != nil {
				fmt.Println("Недопустимый ход:", err)
				continue
			}
		} else {
			fmt.Println("Бот думает...")
			move := bot.BestMove(game)
			if move == nil {
				fmt.Println("Бот не может сделать ход!")
				break
			}

			fmt.Printf("Бот сделал ход: %s\n", move.String())
			game.Move(move)
		}
	}

	printBoard(game)
	fmt.Println("Игра завершена. Результат:", game.Outcome())
	fmt.Println("PGN:", game.String())
}

// Функция для поиска хода по координатам (формат "e2e4")
func findMoveByCoordinates(game *chess.Game, fromStr, toStr string) *chess.Move {
	from, errFrom := parseSquare(fromStr)
	to, errTo := parseSquare(toStr)

	if errFrom != nil || errTo != nil {
		return nil
	}

	for _, move := range game.ValidMoves() {
		if move.S1() == from && move.S2() == to {
			return move
		}
	}

	return nil
}

// Парсинг координат шахматной клетки (например, "e4")
func parseSquare(s string) (chess.Square, error) {
	if len(s) != 2 {
		return 0, fmt.Errorf("неверный формат клетки")
	}

	file := strings.ToLower(s[0:1])
	if file < "a" || file > "h" {
		return 0, fmt.Errorf("неверная буква клетки")
	}

	rank := s[1:2]
	if rank < "1" || rank > "8" {
		return 0, fmt.Errorf("неверная цифра клетки")
	}

	fileIndex := int(file[0] - 'a')
	rankIndex := int(rank[0] - '1')

	return chess.Square(fileIndex + rankIndex*8), nil
}
