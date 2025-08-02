package bots

import (
	"math"

	"github.com/notnil/chess"
)

type DefaultEvaluator struct{}

func (e DefaultEvaluator) Evaluate(game *chess.Game) float64 {
	if outcome := game.Outcome(); outcome != chess.NoOutcome {
		switch outcome {
		case chess.WhiteWon:
			return math.MaxFloat64
		case chess.BlackWon:
			return -math.MaxFloat64
		default:
			return 0
		}
	}

	score := e.materialScore(game) * 10
	score += e.pawnStructure(game) // Переименованный метод
	score += e.pieceActivity(game) // Переименованный метод
	score += e.kingSafety(game)    // Используем существующий метод

	if game.Position().Turn() == chess.Black {
		score = -score
	}

	return score
}

func (e DefaultEvaluator) pawnStructure(game *chess.Game) float64 {
	var score float64
	board := game.Position().Board()

	// Бонус за связанные пешки
	for sq := chess.A1; sq <= chess.H8; sq++ {
		piece := board.Piece(sq)
		if piece == chess.WhitePawn || piece == chess.BlackPawn {
			// Проверяем наличие соседних пешек того же цвета
			file := sq.File()
			rank := sq.Rank()
			for _, delta := range []int{-1, 1} {
				if f := int(file) + delta; f >= 0 && f < 8 {
					neighbor := chess.NewSquare(chess.File(f), rank)
					if board.Piece(neighbor) == piece {
						score += 0.1
						if piece.Color() == chess.Black {
							score -= 0.2 // Учитываем цвет
						}
					}
				}
			}
		}
	}

	return score
}

func (e DefaultEvaluator) pieceActivity(game *chess.Game) float64 {
	var score float64
	validMoves := game.ValidMoves()

	// Бонус за количество доступных ходов
	score += float64(len(validMoves)) * 0.01

	// Штраф за слонов одного цвета
	// ... дополнительная реализация ...

	return score
}

func (e DefaultEvaluator) materialScore(game *chess.Game) float64 {
	// Веса фигур
	values := map[chess.PieceType]float64{
		chess.Pawn:   1,
		chess.Knight: 3,
		chess.Bishop: 3.25,
		chess.Rook:   5,
		chess.Queen:  9,
		chess.King:   100,
	}

	var score float64
	board := game.Position().Board()
	for square := chess.A1; square <= chess.H8; square++ {
		piece := board.Piece(square)
		if piece != chess.NoPiece {
			value := values[piece.Type()]
			if piece.Color() == game.Position().Turn() {
				score += value
			} else {
				score -= value
			}
		}
	}
	return score
}

func (e DefaultEvaluator) positionalScore(game *chess.Game) float64 {
	// Центральный контроль, активность фигур и т.д.
	var score float64

	// Пример: бонус за центральные клетки
	centerSquares := []chess.Square{chess.E4, chess.D4, chess.E5, chess.D5}
	for _, square := range centerSquares {
		piece := game.Position().Board().Piece(square)
		if piece != chess.NoPiece {
			if piece.Color() == game.Position().Turn() {
				score += 0.1
			} else {
				score -= 0.1
			}
		}
	}

	return score
}

func (e DefaultEvaluator) kingSafety(game *chess.Game) float64 {
	// Оценка безопасности короля
	var score float64
	// ... реализация ...
	return score
}
