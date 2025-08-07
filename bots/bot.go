// bot.go
package bots

import "github.com/notnil/chess"

// ChessBot интерфейс для всех ботов
type ChessBot interface {
	BestMove(game *chess.Game) *chess.Move
	Name() string
}

// PositionEvaluator defines the interface for position evaluation
type PositionEvaluator interface {
	Evaluate(game *chess.Game) float64
	pieceValue(p chess.PieceType) float64
	isSquareAttacked(sq chess.Square, byColor chess.Color, game *chess.Game) bool
	isSquareDefended(sq chess.Square, byColor chess.Color, game *chess.Game) bool
}
