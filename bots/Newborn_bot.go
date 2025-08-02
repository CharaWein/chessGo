package bots

import "github.com/notnil/chess"

type NewbornBot struct{}

func NewNewbornBot() *NewbornBot {
	return &NewbornBot{}
}

func (b *NewbornBot) BestMove(game *chess.Game) *chess.Move {
	moves := game.ValidMoves()
	if len(moves) > 0 {
		return moves[0]
	}
	return nil
}

func (b *NewbornBot) Name() string {
	return "Newborn"
}
