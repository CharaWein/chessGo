package bots

import (
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/notnil/chess"
)

type MinimaxBot struct {
	Depth         int
	TimeLimit     time.Duration
	Evaluator     PositionEvaluator
	name          string
	transposition map[uint64]transpositionEntry
	transMutex    sync.RWMutex // Добавляем мьютекс для защиты транспозиционной таблицы
}

type transpositionEntry struct {
	depth int
	score float64
	flag  int // 0 - exact, 1 - lower bound, 2 - upper bound
	move  *chess.Move
}

func NewMinimaxBot(depth int, timeLimit time.Duration, name string) *MinimaxBot {
	return &MinimaxBot{
		Depth:         depth,
		TimeLimit:     timeLimit,
		Evaluator:     DefaultEvaluator{},
		name:          name,
		transposition: make(map[uint64]transpositionEntry),
	}
}

func (b *MinimaxBot) Name() string {
	return b.name
}

func (b *MinimaxBot) BestMove(game *chess.Game) *chess.Move {
	startTime := time.Now()
	endTime := startTime.Add(b.TimeLimit)

	var bestMove *chess.Move
	bestScore := -math.MaxFloat64
	alpha := -math.MaxFloat64
	beta := math.MaxFloat64

	validMoves := b.orderMoves(game.ValidMoves(), game)

	for _, move := range validMoves {
		if time.Now().After(endTime) {
			break
		}

		newGame := game.Clone()
		newGame.Move(move)

		score := -b.alphaBeta(newGame, b.Depth-1, -beta, -alpha, false, endTime)

		if score > bestScore {
			bestScore = score
			bestMove = move
		}

		if score > alpha {
			alpha = score
		}

		if alpha >= beta {
			break
		}
	}

	return bestMove
}

func (b *MinimaxBot) alphaBeta(game *chess.Game, depth int, alpha, beta float64, maximizing bool, endTime time.Time) float64 {
	if time.Now().After(endTime) {
		return 0
	}

	hashBytes := game.Position().Hash()
	hash := binary.LittleEndian.Uint64(hashBytes[:8])

	// Блокируем на чтение
	b.transMutex.RLock()
	entry, ok := b.transposition[hash]
	b.transMutex.RUnlock()

	if ok && entry.depth >= depth {
		switch entry.flag {
		case 0: // Exact
			return entry.score
		case 1: // Lower bound
			alpha = math.Max(alpha, entry.score)
		case 2: // Upper bound
			beta = math.Min(beta, entry.score)
		}
		if alpha >= beta {
			return entry.score
		}
	}

	if depth == 0 || game.Outcome() != chess.NoOutcome {
		return b.Evaluator.Evaluate(game)
	}

	validMoves := b.orderMoves(game.ValidMoves(), game)
	var bestMove *chess.Move
	var bestScore float64

	if maximizing {
		bestScore = -math.MaxFloat64
		for _, move := range validMoves {
			newGame := game.Clone()
			newGame.Move(move)

			score := b.alphaBeta(newGame, depth-1, alpha, beta, false, endTime)

			if score > bestScore {
				bestScore = score
				bestMove = move
			}
			alpha = math.Max(alpha, bestScore)
			if beta <= alpha {
				break
			}
		}
	} else {
		bestScore = math.MaxFloat64
		for _, move := range validMoves {
			newGame := game.Clone()
			newGame.Move(move)

			score := b.alphaBeta(newGame, depth-1, alpha, beta, true, endTime)

			if score < bestScore {
				bestScore = score
				bestMove = move
			}
			beta = math.Min(beta, bestScore)
			if beta <= alpha {
				break
			}
		}
	}

	var flag int
	if bestScore <= alpha {
		flag = 2 // Upper bound
	} else if bestScore >= beta {
		flag = 1 // Lower bound
	} else {
		flag = 0 // Exact
	}

	// Блокируем на запись
	b.transMutex.Lock()
	b.transposition[hash] = transpositionEntry{
		depth: depth,
		score: bestScore,
		flag:  flag,
		move:  bestMove,
	}
	b.transMutex.Unlock()

	return bestScore
}

func (b *MinimaxBot) orderMoves(moves []*chess.Move, game *chess.Game) []*chess.Move {
	var captures, checks, others []*chess.Move

	for _, move := range moves {
		if move.HasTag(chess.Capture) {
			captures = append(captures, move)
		} else if b.isCheckMove(move, game) {
			checks = append(checks, move)
		} else {
			others = append(others, move)
		}
	}

	result := append(captures, checks...)
	return append(result, others...)
}

func (b *MinimaxBot) isCheckMove(move *chess.Move, game *chess.Game) bool {
	newGame := game.Clone()
	newGame.Move(move)

	opponentMoves := newGame.ValidMoves()
	for _, m := range opponentMoves {
		if m.HasTag(chess.Check) {
			return true
		}
	}
	return false
}
