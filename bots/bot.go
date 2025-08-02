package bots

import "github.com/notnil/chess"

// ChessBot интерфейс для всех ботов
type ChessBot interface {
	BestMove(game *chess.Game) *chess.Move
	Name() string
}
