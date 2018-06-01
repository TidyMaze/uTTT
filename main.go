package main

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"time"
)

const (
	unassigned = iota
	player1    = iota
	player2    = iota
	draw       = iota
)

type coord struct {
	x, y int
}

type state struct {
	grid       [9][9]int
	nextPlayer int
	bigGrid    [3][3]int
	winner     int
}

type actionScored struct {
	action coord
	score  float32
}

func log(a interface{}) {
	fmt.Fprintln(os.Stderr, a)
}

func getCoordInBigGrid(x, y int) (bigX, bigY int) {
	return x % 3, y % 3
}

func getActionsInSubGrid(s *state, bigX, bigY int, actions *[]coord){
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			y := bigY*3 + i
			x := bigX*3 + j
			if s.grid[y][x] == unassigned {
				*actions = append(*actions, coord{x,y})
			}
		}
	}
}

func getActionsAllGrid(s *state, actions *[]coord) {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if s.bigGrid[i][j] == unassigned {
				getActionsInSubGrid(s, j, i, actions)
			}
		}
	}
}

func getActions(s *state, oppAction coord) (actions []coord) {
	res := make([]coord, 0, 9*9)
	if oppAction.x != -1 && oppAction.y != -1 {
		bigX, bigY := getCoordInBigGrid(oppAction.x, oppAction.y)
		if s.bigGrid[bigY][bigX] == unassigned {
			getActionsInSubGrid(s, bigX, bigY, &res)
		} else {
			getActionsAllGrid(s, &res)
		}
	} else {
		getActionsAllGrid(s, &res)
	}
	return res
}

func playUntilEnd(s state, oppAction coord) (winner int) {

	currentState := s
	lastAction := oppAction
	for currentState.winner == unassigned {
		actions := getActions(&currentState, lastAction)

		if len(actions) == 0 {

			countPlayer1 := 0
			countPlayer2 := 0
			for bigY := 0; bigY < 3; bigY++ {
				for bigX := 0; bigX < 3; bigX++ {
					if currentState.bigGrid[bigY][bigX] == player1 {
						countPlayer1++
					} else if currentState.bigGrid[bigY][bigX] == player2 {
						countPlayer2++
					} else if currentState.bigGrid[bigY][bigX] == draw {
						// nothing
					} else {
						log(currentState.grid)
						log(currentState.bigGrid)
						log(actions)
						panic(fmt.Sprintf("No action found but no winner for big Grid %d %d", bigX, bigY))
					}
				}
			}

			if countPlayer1 > countPlayer2 {
				return player1
			} else if countPlayer2 > countPlayer1 {
				return player2
			} else {
				return draw
			}
		}

		index := rand.Intn(len(actions))
		picked := actions[index]
		playMutable(&currentState, picked)
		lastAction = picked
	}

	return currentState.winner
}

func keepSim(turn int) bool {
	maxTimeMs := 100 * 90 / 100
	if turn < 2 {
		maxTimeMs = 1000 * 90 / 100
	}
	return time.Since(start).Nanoseconds()/int64(1e6) < int64(maxTimeMs)
}

func sortByScore(scored []actionScored) {
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})
}

func mc(turn int, s state, oppAction coord) (bestAction coord) {
	log("starting MC")

	actions := getActions(&s, oppAction)

	var (
		games     = make([]int, len(actions))
		victories = make([]int, len(actions))
		draws     = make([]int, len(actions))
		scores    = make([]actionScored, len(actions))
	)

	log("Start games")

	game := 0
	for ; game % 500 != 0 || keepSim(turn); game++ {
		index := game % len(actions)
		picked := actions[index]

		resultFirstTurn := playImmutable(s, picked)

		winner := playUntilEnd(resultFirstTurn, picked)

		games[index]++
		if winner == s.nextPlayer {
			victories[index]++
		} else if winner == draw {
			draws[index]++
		}
	}

	log(fmt.Sprintf("End game %d", game))

	for i := 0; i < len(actions); i++ {
		scores[i] = actionScored{
			actions[i],
			float32(victories[i]) / float32(games[i])*1000 + float32(draws[i]) / float32(games[i]),
		}
	}
	sortByScore(scores)
	return scores[0].action
}

func getNextPlayer(cur int) (next int) {
	switch cur {
	case player1:
		return player2
	case player2:
		return player1
	default:
		panic("Unknown player" + string(cur))
	}
}

func playImmutable(s state, c coord) (res state) {
	s.grid[c.y][c.x] = s.nextPlayer

	s.nextPlayer = getNextPlayer(s.nextPlayer)

	bigX := c.x / 3
	bigY := c.y / 3

	winner := checkWinSubGrid(&s, bigX, bigY)

	if winner == unassigned {
		someUnassigned := false
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				if s.grid[bigY*3+i][bigX*3+j] == unassigned {
					someUnassigned = true
				}
			}
		}

		if !someUnassigned {
			s.bigGrid[bigY][bigX] = draw
		}

	} else {
		s.bigGrid[bigY][bigX] = winner
		s.winner = checkWinBigGrid(&s)
	}

	return s
}

func playMutable(s *state, c coord) {
	s.grid[c.y][c.x] = s.nextPlayer

	s.nextPlayer = getNextPlayer(s.nextPlayer)

	bigX := c.x / 3
	bigY := c.y / 3

	winner := checkWinSubGrid(s, bigX, bigY)

	if winner == unassigned {
		someUnassigned := false
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				if s.grid[bigY*3+i][bigX*3+j] == unassigned {
					someUnassigned = true
				}
			}
		}

		if !someUnassigned {
			s.bigGrid[bigY][bigX] = draw
		}

	} else {
		s.bigGrid[bigY][bigX] = winner
		s.winner = checkWinBigGrid(s)
	}
}

func checkWinBigGrid(s *state) (winner int) {
	g := s.bigGrid

	// check rows
	for i := 0; i < 3; i++ {
		if g[i][0] != unassigned && g[i][0] == g[i][1] && g[i][1] == g[i][2] {
			return g[i][0]
		}
	}

	// check columns
	for j := 0; j < 3; j++ {
		if g[0][j] != unassigned && g[0][j] == g[1][j] && g[1][j] == g[2][j] {
			return g[0][j]
		}
	}

	// check diag
	if g[0][0] != unassigned && g[0][0] == g[1][1] && g[1][1] == g[2][2] {
		return g[0][0]
	}

	// check anti diag
	if g[0][2] != unassigned && g[0][2] == g[1][1] && g[1][1] == g[2][0] {
		return g[0][2]
	}

	return unassigned
}

func checkWinSubGrid(s *state, bigX int, bigY int) (winner int) {
	bigXO := bigX * 3
	bigYO := bigY * 3

	g := s.grid

	// check rows
	for i := 0; i < 3; i++ {
		if g[bigYO+i][bigXO+0] != unassigned && g[bigYO+i][bigXO+0] == g[bigYO+i][bigXO+1] && g[bigYO+i][bigXO+1] == g[bigYO+i][bigXO+2] {
			return g[bigYO+i][bigXO+0]
		}
	}

	// check columns
	for j := 0; j < 3; j++ {
		if g[bigYO+0][bigXO+j] != unassigned && g[bigYO+0][bigXO+j] == g[bigYO+1][bigXO+j] && g[bigYO+1][bigXO+j] == g[bigYO+2][bigXO+j] {
			return g[bigYO+0][bigXO+j]
		}
	}

	// check diag
	if g[bigYO+0][bigXO+0] != unassigned && g[bigYO+0][bigXO+0] == g[bigYO+1][bigXO+1] && g[bigYO+1][bigXO+1] == g[bigYO+2][bigXO+2] {
		return g[bigYO+0][bigXO+0]
	}

	// check anti diag
	if g[bigYO+0][bigXO+2] != unassigned && g[bigYO+0][bigXO+2] == g[bigYO+1][bigXO+1] && g[bigYO+1][bigXO+1] == g[bigYO+2][bigXO+0] {
		return g[bigYO+0][bigXO+2]
	}

	return unassigned
}

var start = time.Now()

func initTurnPlayer(turn, player, opponentRow, opponentCol int) (playerRes, turnRes int) {
	playerRes = player
	turnRes = turn

	if turn == -1 {
		if opponentRow == -1 && opponentCol == -1 {
			playerRes = player1
			turnRes = 0
		} else {
			playerRes = player2
			turnRes = 1
		}
	}
	return playerRes, turnRes
}

func main() {

	myPlayer := unassigned

	s := state{nextPlayer: unassigned}

	for turn := -1; ; turn+=2 {
		var opponentRow, opponentCol int
		fmt.Scan(&opponentRow, &opponentCol)

		last := coord{-1, -1}

		myPlayer, turn = initTurnPlayer(turn, myPlayer, opponentRow, opponentCol)

		log(myPlayer)

		s.nextPlayer = getNextPlayer(myPlayer)

		last.x = opponentCol
		last.y = opponentRow

		if last.x != -1 && last.y != -1 {
			s = playImmutable(s, last)
		}

		s.nextPlayer = myPlayer

		var validActionCount int
		fmt.Scan(&validActionCount)

		actions := make([]coord, validActionCount)
		for i := 0; i < validActionCount; i++ {
			var row, col int
			fmt.Scan(&row, &col)
			actions = append(actions, coord{col, row})
		}
		start = time.Now()

		picked := mc(turn, s, last)

		s = playImmutable(s, picked)

		log(fmt.Sprintf("Spent %d ms", time.Since(start).Nanoseconds()/int64(1e6)))
		fmt.Println(fmt.Sprintf("%d %d", picked.y, picked.x)) // Write action to stdout
	}
}
