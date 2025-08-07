package bots

import (
	"fmt"
	"math"

	"github.com/notnil/chess"
)

type DefaultEvaluator struct{}

const (
	MaterialWeight      = 100
	MobilityWeight      = 1
	PawnStructWeight    = 30
	KingSafetyWeight    = 50
	CenterWeight        = 20
	PieceActivityWeight = 15
)

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

	score := e.materialScore(game)*MaterialWeight +
		e.mobilityScore(game)*MobilityWeight +
		e.pawnStructure(game)*PawnStructWeight +
		e.kingSafety(game)*KingSafetyWeight +
		e.centerControl(game)*CenterWeight +
		e.pieceActivity(game)*PieceActivityWeight

	if game.Position().Turn() == chess.Black {
		score = -score
	}

	// Логирование (можно убрать после отладки)
	fmt.Printf("Оценка позиции: %.2f (Материал: %.2f, Угрозы: %.2f)\n",
		score,
		e.materialScore(game)*100,
		e.threatsScore(game)*50)

	if game.Position().Turn() == chess.Black {
		score = -score
	}
	return score
}

func (e DefaultEvaluator) materialScore(game *chess.Game) float64 {
	values := map[chess.PieceType]float64{
		chess.Pawn:   1,
		chess.Knight: 3,
		chess.Bishop: 3,
		chess.Rook:   5,
		chess.Queen:  9,
		chess.King:   1000,
	}

	var score float64
	board := game.Position().Board()

	// Бонус за сохранение фигур
	pieceCount := 0
	for sq := chess.A1; sq <= chess.H8; sq++ {
		piece := board.Piece(sq)
		if piece != chess.NoPiece && piece.Type() != chess.Pawn && piece.Type() != chess.King {
			pieceCount++
		}
	}
	score += float64(pieceCount) * 0.05 // Бонус за каждую сохраненную фигуру

	// Основная оценка материала
	for sq := chess.A1; sq <= chess.H8; sq++ {
		piece := board.Piece(sq)
		if piece != chess.NoPiece {
			value := values[piece.Type()]
			if piece.Color() == chess.White {
				score += value
			} else {
				score -= value
			}
		}
	}
	return score
}

func (e DefaultEvaluator) threatsScore(game *chess.Game) float64 {
	var score float64
	board := game.Position().Board()

	// Проверяем все возможные взятия
	for _, move := range game.ValidMoves() {
		if move.HasTag(chess.Capture) {
			captured := board.Piece(move.S2())
			attacker := board.Piece(move.S1())

			// Бонус за взятие более ценной фигуры
			if e.pieceValue(captured.Type()) > e.pieceValue(attacker.Type()) {
				score += 0.5
			}
			// Штраф за отдачу более ценной фигуры
			if e.pieceValue(attacker.Type()) > e.pieceValue(captured.Type()) {
				score -= 1.0
			}
		}
	}

	return score
}

func (e DefaultEvaluator) pieceValue(piece chess.PieceType) float64 {
	switch piece {
	case chess.Pawn:
		return 1
	case chess.Knight:
		return 3
	case chess.Bishop:
		return 3
	case chess.Rook:
		return 5
	case chess.Queen:
		return 9
	default:
		return 0
	}
}

func (e DefaultEvaluator) centerControl(game *chess.Game) float64 {
	var score float64
	center := []chess.Square{chess.D4, chess.E4, chess.D5, chess.E5}
	extendedCenter := []chess.Square{
		chess.C3, chess.D3, chess.E3, chess.F3,
		chess.C4, chess.F4, chess.C5, chess.F5,
		chess.C6, chess.D6, chess.E6, chess.F6,
	}

	board := game.Position().Board()

	for _, sq := range center {
		score += e.squareControl(sq, board, game)
	}

	for _, sq := range extendedCenter {
		score += e.squareControl(sq, board, game) * 0.5
	}

	return score
}

func (e DefaultEvaluator) squareControl(sq chess.Square, board *chess.Board, game *chess.Game) float64 {
	var score float64

	if board.Piece(sq) == chess.NoPiece || board.Piece(sq).Color() == chess.Black {
		if e.isSquareAttacked(sq, chess.White, game) {
			score += 0.2
		}
	}

	if board.Piece(sq) == chess.NoPiece || board.Piece(sq).Color() == chess.White {
		if e.isSquareAttacked(sq, chess.Black, game) {
			score -= 0.2
		}
	}

	return score
}

func (e DefaultEvaluator) isSquareAttacked(sq chess.Square, byColor chess.Color, game *chess.Game) bool {
	for _, move := range game.ValidMoves() {
		if move.S2() == sq && game.Position().Board().Piece(move.S1()).Color() == byColor {
			return true
		}
	}
	return false
}

func (e DefaultEvaluator) mobilityScore(game *chess.Game) float64 {
	// Подсчет мобильности для текущего игрока
	currentMoves := len(game.ValidMoves())

	// Для подсчета мобильности противника создаем копию игры
	tmpGame := game.Clone()

	// Если нет возможных ходов, возвращаем 0
	if len(tmpGame.ValidMoves()) == 0 {
		return 0
	}

	// Делаем первый возможный ход (чтобы изменить сторону)
	move := tmpGame.ValidMoves()[0]
	if err := tmpGame.Move(move); err != nil {
		return 0
	}

	opponentMoves := len(tmpGame.ValidMoves())

	if game.Position().Turn() == chess.White {
		return float64(currentMoves - opponentMoves)
	}
	return float64(opponentMoves - currentMoves)
}

func (e DefaultEvaluator) pawnStructure(game *chess.Game) float64 {
	var score float64
	board := game.Position().Board()

	whitePawns := make(map[chess.File]int)
	blackPawns := make(map[chess.File]int)

	for sq := chess.A1; sq <= chess.H8; sq++ {
		piece := board.Piece(sq)
		if piece == chess.WhitePawn {
			whitePawns[sq.File()]++
		} else if piece == chess.BlackPawn {
			blackPawns[sq.File()]++
		}
	}

	for file, count := range whitePawns {
		if count > 1 {
			score -= 0.3 * float64(count-1)
		}
		prevFile := chess.File(int(file) - 1)
		nextFile := chess.File(int(file) + 1)
		if (prevFile < 0 || whitePawns[prevFile] == 0) &&
			(nextFile > 7 || whitePawns[nextFile] == 0) {
			score -= 0.5
		}
	}

	for file, count := range blackPawns {
		if count > 1 {
			score += 0.3 * float64(count-1)
		}
		prevFile := chess.File(int(file) - 1)
		nextFile := chess.File(int(file) + 1)
		if (prevFile < 0 || blackPawns[prevFile] == 0) &&
			(nextFile > 7 || blackPawns[nextFile] == 0) {
			score += 0.5
		}
	}

	return score
}

func (e DefaultEvaluator) kingSafety(game *chess.Game) float64 {
	var score float64
	board := game.Position().Board()

	var whiteKing, blackKing chess.Square
	for sq := chess.A1; sq <= chess.H8; sq++ {
		piece := board.Piece(sq)
		if piece == chess.WhiteKing {
			whiteKing = sq
		} else if piece == chess.BlackKing {
			blackKing = sq
		}
	}

	score += e.kingProtection(whiteKing, chess.White, board, game)
	score -= e.kingProtection(blackKing, chess.Black, board, game)

	return score
}

func (e DefaultEvaluator) kingProtection(kingSq chess.Square, color chess.Color, board *chess.Board, game *chess.Game) float64 {
	protection := 0.0
	danger := 0.0

	kingFile := int(kingSq.File())
	kingRank := int(kingSq.Rank())

	for x := -1; x <= 1; x++ {
		for y := -1; y <= 1; y++ {
			if x == 0 && y == 0 {
				continue
			}
			file := kingFile + x
			rank := kingRank + y

			if file >= 0 && file < 8 && rank >= 0 && rank < 8 {
				sq := chess.NewSquare(chess.File(file), chess.Rank(rank))
				piece := board.Piece(sq)

				if piece.Color() == color {
					protection += 0.2
				} else if piece.Color() == color.Other() {
					danger += 0.3
				}
			}
		}
	}

	return protection - danger
}

func (e DefaultEvaluator) pieceActivity(game *chess.Game) float64 {
	var score float64
	board := game.Position().Board()

	for sq := chess.A1; sq <= chess.H8; sq++ {
		piece := board.Piece(sq)
		if piece != chess.NoPiece && piece.Type() != chess.King {
			if (piece.Color() == chess.White && sq.Rank() >= 4) ||
				(piece.Color() == chess.Black && sq.Rank() <= 3) {
				score += 0.1
			}

			file := int(sq.File())
			rank := int(sq.Rank())
			if (file >= 2 && file <= 5) && (rank >= 2 && rank <= 5) {
				score += 0.15
			}
		}
	}

	if game.Position().Turn() == chess.Black {
		score = -score
	}

	return score
}
