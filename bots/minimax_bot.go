package bots

import (
	"encoding/binary"
	"math"
	"sort"
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
	transMutex    sync.RWMutex
	killerMoves   [2][64]*chess.Move // Киллер-ходы для улучшения сортировки
}

type transpositionEntry struct {
	depth int
	score float64
	flag  int
	move  *chess.Move
}

func NewMinimaxBot(depth int, timeLimit time.Duration, name string) *MinimaxBot {
	killerMoves := [2][64]*chess.Move{} // Initialize with nil moves
	return &MinimaxBot{
		Depth:         depth,
		TimeLimit:     timeLimit,
		Evaluator:     DefaultEvaluator{},
		name:          name,
		transposition: make(map[uint64]transpositionEntry),
		killerMoves:   killerMoves,
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

	// Итеративное углубление
	for currentDepth := 1; currentDepth <= b.Depth; currentDepth++ {
		if time.Now().After(endTime) {
			break
		}

		validMoves := b.orderMoves(game.ValidMoves(), game, currentDepth)

		for _, move := range validMoves {
			if time.Now().After(endTime) {
				break
			}

			newGame := game.Clone()
			newGame.Move(move)

			score := -b.alphaBeta(newGame, currentDepth-1, -beta, -alpha, false, endTime)

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
	}

	return bestMove
}

func (b *MinimaxBot) alphaBeta(game *chess.Game, depth int, alpha, beta float64, maximizing bool, endTime time.Time) float64 {
	if time.Now().After(endTime) {
		return 0
	}

	hashBytes := game.Position().Hash()
	hash := binary.LittleEndian.Uint64(hashBytes[:8])

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
		return b.quiescenceSearch(game, alpha, beta, endTime)
	}

	validMoves := b.orderMoves(game.ValidMoves(), game, depth)
	var bestMove *chess.Move
	var bestScore float64

	if maximizing {
		bestScore = -math.MaxFloat64
		for _, move := range validMoves {
			newGame := game.Clone()
			newGame.Move(move)

			score := -b.alphaBeta(newGame, depth-1, -beta, -alpha, false, endTime)

			if score > bestScore {
				bestScore = score
				bestMove = move
			}
			alpha = math.Max(alpha, bestScore)
			if beta <= alpha {
				b.storeKillerMove(move, depth)
				break
			}
		}
	} else {
		bestScore = math.MaxFloat64
		for _, move := range validMoves {
			newGame := game.Clone()
			newGame.Move(move)

			score := -b.alphaBeta(newGame, depth-1, alpha, beta, true, endTime)

			if score < bestScore {
				bestScore = score
				bestMove = move
			}
			beta = math.Min(beta, bestScore)
			if beta <= alpha {
				b.storeKillerMove(move, depth)
				break
			}
		}
	}

	var flag int
	if bestScore <= alpha {
		flag = 2
	} else if bestScore >= beta {
		flag = 1
	} else {
		flag = 0
	}

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

func (b *MinimaxBot) quiescenceSearch(game *chess.Game, alpha, beta float64, endTime time.Time) float64 {
	standPat := b.Evaluator.Evaluate(game)
	if standPat >= beta {
		return beta
	}
	if alpha < standPat {
		alpha = standPat
	}

	captures := b.getCaptures(game)
	for _, move := range captures {
		if time.Now().After(endTime) {
			return 0
		}

		newGame := game.Clone()
		newGame.Move(move)

		score := -b.quiescenceSearch(newGame, -beta, -alpha, endTime)

		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}

	return alpha
}

func (b *MinimaxBot) orderMoves(moves []*chess.Move, game *chess.Game, depth int) []*chess.Move {
	var captures, checks, killers, others []*chess.Move

	for _, move := range moves {
		if move.HasTag(chess.Capture) {
			captures = append(captures, move)
		} else if b.isCheckMove(move, game) {
			checks = append(checks, move)
		} else if b.isKillerMove(move, depth) {
			killers = append(killers, move)
		} else {
			others = append(others, move)
		}
	}

	// Сортируем взятия по материалу
	sort.Slice(captures, func(i, j int) bool {
		return b.see(game, captures[i]) > b.see(game, captures[j])
	})

	result := append(captures, checks...)
	result = append(result, killers...)
	return append(result, others...)
}

func (b *MinimaxBot) see(game *chess.Game, move *chess.Move) float64 {
	// Простая оценка SEE (Static Exchange Evaluation)
	captured := game.Position().Board().Piece(move.S2())
	switch captured.Type() {
	case chess.Pawn:
		return 1
	case chess.Knight:
		return 3
	case chess.Bishop:
		return 3.25
	case chess.Rook:
		return 5
	case chess.Queen:
		return 9
	default:
		return 0
	}
}

func (b *MinimaxBot) storeKillerMove(move *chess.Move, depth int) {
	if move == nil || depth >= len(b.killerMoves) {
		return
	}
	b.killerMoves[depth][1] = b.killerMoves[depth][0]
	b.killerMoves[depth][0] = move
}

func (b *MinimaxBot) isKillerMove(move *chess.Move, depth int) bool {
	if move == nil || depth >= len(b.killerMoves) {
		return false
	}
	if b.killerMoves[depth][0] != nil && move.String() == b.killerMoves[depth][0].String() {
		return true
	}
	if b.killerMoves[depth][1] != nil && move.String() == b.killerMoves[depth][1].String() {
		return true
	}
	return false
}

func (b *MinimaxBot) getCaptures(game *chess.Game) []*chess.Move {
	var captures []*chess.Move
	for _, move := range game.ValidMoves() {
		if move.HasTag(chess.Capture) {
			captures = append(captures, move)
		}
	}
	return captures
}

func (b *MinimaxBot) isCheckMove(move *chess.Move, game *chess.Game) bool {
	newGame := game.Clone()
	newGame.Move(move)

	// Get the opponent's color (whose turn it is now)
	opponentColor := newGame.Position().Turn()

	// Find the opponent's king square
	var kingSquare chess.Square
	board := newGame.Position().Board()
	for sq := chess.A1; sq <= chess.H8; sq++ {
		piece := board.Piece(sq)
		if piece.Type() == chess.King && piece.Color() == opponentColor {
			kingSquare = sq
			break
		}
	}

	// Check if any of our pieces attack the king
	ourColor := opponentColor.Other()
	for _, m := range newGame.ValidMoves() {
		if m.S2() == kingSquare {
			piece := board.Piece(m.S1())
			if piece.Color() == ourColor {
				return true
			}
		}
	}

	return false
}
