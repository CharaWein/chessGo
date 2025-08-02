package bots

import (
	"fmt"
	"math"
	"time"

	"github.com/notnil/chess"
)

type MinimaxBot struct {
	Depth     int
	TimeLimit time.Duration
	Evaluator PositionEvaluator
	startTime time.Time
}

type PositionEvaluator interface {
	Evaluate(*chess.Game) float64
}

func NewMinimaxBot(depth int, timeLimit time.Duration) *MinimaxBot {
	return &MinimaxBot{
		Depth:     depth,
		TimeLimit: timeLimit,
		Evaluator: DefaultEvaluator{},
	}
}

func (b *MinimaxBot) Name() string {
	return fmt.Sprintf("Minimax Bot (depth %d)", b.Depth) // Используем fmt.Sprintf
}

func (b *MinimaxBot) BestMove(game *chess.Game) *chess.Move {
	if game == nil {
		return nil
	}

	b.startTime = time.Now()
	validMoves := game.ValidMoves()
	if len(validMoves) == 0 {
		return nil
	}

	// Простое падение глубины, если мало времени
	depth := b.Depth
	if time.Since(b.startTime) > b.TimeLimit/2 {
		depth = b.Depth - 1
	}

	result := b.minimax(game.Clone(), depth, -math.MaxFloat64, math.MaxFloat64, true)
	return result.move
}

type scoredMove struct {
	move  *chess.Move
	score float64
}

func (b *MinimaxBot) minimax(game *chess.Game, depth int, alpha, beta float64, maximizing bool) scoredMove {
	if depth == 0 || game.Outcome() != chess.NoOutcome || time.Since(b.startTime) > b.TimeLimit {
		return scoredMove{nil, b.Evaluator.Evaluate(game)}
	}

	validMoves := game.ValidMoves()
	if len(validMoves) == 0 {
		return scoredMove{nil, b.Evaluator.Evaluate(game)}
	}

	var bestMove scoredMove
	if maximizing {
		bestMove.score = -math.MaxFloat64
		for _, move := range validMoves {
			newGame := game.Clone()
			newGame.Move(move)
			current := b.minimax(newGame, depth-1, alpha, beta, false)
			if current.score > bestMove.score {
				bestMove = scoredMove{move, current.score}
			}
			alpha = math.Max(alpha, bestMove.score)
			if beta <= alpha {
				break
			}
		}
	} else {
		bestMove.score = math.MaxFloat64
		for _, move := range validMoves {
			newGame := game.Clone()
			newGame.Move(move)
			current := b.minimax(newGame, depth-1, alpha, beta, true)
			if current.score < bestMove.score {
				bestMove = scoredMove{move, current.score}
			}
			beta = math.Min(beta, bestMove.score)
			if beta <= alpha {
				break
			}
		}
	}

	return bestMove
}
