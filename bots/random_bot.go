package bots

import "github.com/notnil/chess"

type RandomBot struct{}

func NewRandomBot() *RandomBot {
	return &RandomBot{}
}

func (b *RandomBot) BestMove(game *chess.Game) *chess.Move {
	moves := game.ValidMoves()
	if len(moves) > 0 {
		return moves[0]
	}
	return nil
}

func (b *RandomBot) Name() string {
	return "Random Bot"
}
