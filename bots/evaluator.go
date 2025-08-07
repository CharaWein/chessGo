package bots

import (
	"fmt"
	"math"

	"github.com/notnil/chess"
)

type DefaultEvaluator struct{}

const (
	MaterialWeight = 2000 // Главный приоритет
	ThreatWeight   = 1500 // Угрозы/защиты почти равны материалу
	// Все позиционные факторы минимальны
	MobilityWeight      = 1
	PawnStructWeight    = 1
	KingSafetyWeight    = 5
	CenterWeight        = 1
	PieceActivityWeight = 1
)

func (e DefaultEvaluator) pieceValue(p chess.PieceType) float64 {
	switch p {
	case chess.Pawn:
		return 1
	case chess.Knight:
		return 3.05
	case chess.Bishop:
		return 3.33
	case chess.Rook:
		return 5.63
	case chess.Queen:
		return 9.5
	case chess.King:
		return 100
	default:
		return 0
	}
}

func (e DefaultEvaluator) Evaluate(game *chess.Game) float64 {
	if outcome := game.Outcome(); outcome != chess.NoOutcome {
		switch outcome {
		case chess.WhiteWon:
			return math.MaxFloat64 / 2
		case chess.BlackWon:
			return -math.MaxFloat64 / 2
		default:
			return 0
		}
	}

	material := e.materialScore(game)
	threats := e.threatsScore(game)

	// Основная оценка (больше влияния)
	score := material*MaterialWeight + threats*ThreatWeight

	// Второстепенные факторы (меньше влияния)
	minorFactors := e.mobilityScore(game)*MobilityWeight +
		e.pawnStructure(game)*PawnStructWeight +
		e.kingSafety(game)*KingSafetyWeight +
		e.centerControl(game)*CenterWeight +
		e.pieceActivity(game)*PieceActivityWeight

	score += minorFactors * 0.02

	if game.Position().Turn() == chess.Black {
		score = -score
	}

	fmt.Printf("Оценка: %.2f (Материал: %.2f Угрозы: %.2f Факторы: %.2f)\n",
		score, material, threats, minorFactors)

	return score
}

func (e DefaultEvaluator) materialScore(game *chess.Game) float64 {
	var score float64
	board := game.Position().Board()

	for sq := chess.A1; sq <= chess.H8; sq++ {
		piece := board.Piece(sq)
		if piece != chess.NoPiece {
			value := e.pieceValue(piece.Type())
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
	turn := game.Position().Turn()
	opponent := turn.Other()

	// 1. Жесткий штраф за каждую атакованную фигуру
	for sq := chess.A1; sq <= chess.H8; sq++ {
		piece := board.Piece(sq)
		if piece != chess.NoPiece && piece.Color() == turn {
			if e.isSquareAttacked(sq, opponent, game) {
				pieceVal := e.pieceValue(piece.Type())

				if !e.isSquareDefended(sq, turn, game) {
					// Критический штраф за незащищенную фигуру под боем
					score -= pieceVal * 3.0 // В 3 раза больше ценности фигуры!

					// Дополнительный штраф если это не пешка
					if piece.Type() != chess.Pawn {
						score -= 2.0
					}
				} else {
					// Штраф даже за защищенную фигуру
					score -= pieceVal * 0.5
				}
			}
		}
	}

	// 2. Супер-бонусы за взятия
	for _, move := range game.ValidMoves() {
		if move.HasTag(chess.Capture) {
			captured := board.Piece(move.S2())
			capturer := board.Piece(move.S1())
			capturedVal := e.pieceValue(captured.Type())

			// Базовый бонус
			attackBonus := capturedVal * 1.2

			if !e.isSquareDefended(move.S2(), opponent, game) {
				// Огромный бонус за взятие незащищенной фигуры
				attackBonus += 5.0
			} else if e.pieceValue(capturer.Type()) < capturedVal {
				// Бонус за выгодный размен
				attackBonus += (capturedVal - e.pieceValue(capturer.Type())) * 0.8
			}

			score += attackBonus
		}
	}

	return score
}

func (e DefaultEvaluator) isSquareDefended(sq chess.Square, byColor chess.Color, game *chess.Game) bool {
	for _, move := range game.ValidMoves() {
		if move.S2() == sq && game.Position().Board().Piece(move.S1()).Color() == byColor {
			// Не учитываем короля как защитника
			if game.Position().Board().Piece(move.S1()).Type() != chess.King {
				return true
			}
		}
	}
	return false
}

func (e DefaultEvaluator) isSquareAttacked(sq chess.Square, byColor chess.Color, game *chess.Game) bool {
	for _, move := range game.ValidMoves() {
		if move.S2() == sq && game.Position().Board().Piece(move.S1()).Color() == byColor {
			return true
		}
	}
	return false
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
