package bots

import (
	"math"

	"github.com/notnil/chess"
)

type PositionEvaluator interface {
	Evaluate(*chess.Game) float64
}

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

	score := e.materialScore(game)*10 +
		e.pawnStructure(game)*3 +
		e.pieceActivity(game)*2 +
		e.kingSafety(game)*5 +
		e.centerControl(game)*1.5

	if game.Position().Turn() == chess.Black {
		score = -score
	}

	return score
}

func (e DefaultEvaluator) centerControl(game *chess.Game) float64 {
	var score float64
	center := []chess.Square{chess.D4, chess.E4, chess.D5, chess.E5}
	board := game.Position().Board()

	for _, sq := range center {
		piece := board.Piece(sq)
		if piece != chess.NoPiece {
			if piece.Color() == game.Position().Turn() {
				score += 0.5
			} else {
				score -= 0.5
			}
		}
	}
	return score
}

func (e DefaultEvaluator) pawnStructure(game *chess.Game) float64 {
	var score float64
	board := game.Position().Board()

	for sq := chess.A1; sq <= chess.H8; sq++ {
		piece := board.Piece(sq)
		if piece == chess.WhitePawn || piece == chess.BlackPawn {
			file := sq.File()
			rank := sq.Rank()
			for _, delta := range []int{-1, 1} {
				if f := int(file) + delta; f >= 0 && f < 8 {
					neighbor := chess.NewSquare(chess.File(f), rank)
					if board.Piece(neighbor) == piece {
						score += 0.1
						if piece.Color() == chess.Black {
							score -= 0.2
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

	score += float64(len(validMoves)) * 0.01

	return score
}

func (e DefaultEvaluator) materialScore(game *chess.Game) float64 {
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

func (e DefaultEvaluator) kingSafety(game *chess.Game) float64 {
	var score float64
	return score
}
