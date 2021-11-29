// Copyright 2021 Brad Fitzpatrick. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The candysim command simulates games of Candylane to give me an
// idea of how long this will go on.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"sort"
	"time"
)

var (
	verbose  = flag.Bool("verbose", false, "verbose/debug")
	players  = flag.Int("players", 1, "number of players")
	N        = flag.Int("n", 10000, "number of games to simulate")
	backJump = flag.Bool("allow-back", true, "allow backwards candy jumps")
)

type color uint8

const (
	red color = iota + 1
	orange
	yellow
	green
	blue
	purple
)

var colors = [...]string{
	red:    "red",
	orange: "orange",
	yellow: "yellow",
	green:  "green",
	blue:   "blue",
	purple: "purple",
}

func (c color) String() string {
	return colors[c]
}

var (
	r = square{color: red}
	o = square{color: orange}
	y = square{color: yellow}
	g = square{color: green}
	b = square{color: blue}
	p = square{color: purple}
)

type square struct {
	color     color  // color or candy is set
	candy     string // color or candy is set
	roadStart string
	roadEnd   string
	pit       bool
	warpTo    int // if roadStart
}

func roadStart(s square, roadName string) square {
	s.roadStart = roadName
	return s
}

func roadEnd(s square, roadName string) square {
	s.roadEnd = roadName
	return s
}

func pit(s square) square {
	s.pit = true
	return s
}

func candy(name string) square { return square{candy: name} }

var board = []square{
	p, y, b, o,
	roadStart(g, "rainbow"),
	r, p,
	candy("heart"),
	y, b, o, g, r, p, y,
	candy("cane"),
	b, o, g, r, p, y, b, o, g, r, p, y,
	candy("man"),
	b, o, g, r,
	roadStart(p, "mountain"),
	y, b, o, g, r, p, y, b,
	candy("drop"),
	o, g, r,
	roadEnd(p, "mountain"),
	y,
	pit(b),
	o, g, r, p, y, b, o, g, r, p, y,
	roadEnd(b, "rainbow"),
	o, g, r, p, y, b, o, g, r, p, y, b, o, g,
	candy("brittle"),
	r, p, y, b, o, g, r, p, y, b, o, g,
	pit(r),
	p, y, b, o, g, r, p, y, b,
	candy("pop"),
	o, g, r, p, y, b, o,
	candy("float"),
	g, r, p, y, b, o, g, r, p, y, b, o, g, r, p, y,
	pit(b),
	o, g, r, p, y, b, o, g, r, p, y, b,
}

var candyPos = map[string]int{}

func init() {
	for i := range board {
		s := &board[i]
		if s.candy != "" {
			if _, ok := candyPos[s.candy]; ok {
				panic("dup " + s.candy)
			} else {
				candyPos[s.candy] = i
			}
		}
		if s.roadStart != "" {
			s.warpTo = findRoadEnd(s.roadStart)
		}
	}
}

func findRoadEnd(name string) int {
	for i, s := range board {
		if s.roadEnd == name {
			return i
		}
	}
	panic("road not found")
}

type card struct {
	candy    string
	color    color
	double   bool
	cardType int // unique for (candy, color, double)
}

func (c card) String() string {
	if c.candy != "" {
		return c.candy
	}
	if c.double {
		return fmt.Sprintf("double %v", c.color)
	}
	return c.color.String()
}

var (
	deck     []card
	shuffled []card
)

func deal() *card {
	if len(shuffled) == 0 {
		shuffled = deck
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(deck), func(i, j int) {
			deck[i], deck[j] = deck[j], deck[i]
		})
	}
	c := &shuffled[0]
	shuffled = shuffled[1:]
	return c
}

var uniqCardTypes int

// addCard adds n cards of type t (1=single, 2=double) of the
// provided color.
func addCard(n, t int, color color) {
	uniqCardTypes++
	c := card{
		color:    color,
		double:   t == 2,
		cardType: uniqCardTypes - 1,
	}
	if t != 1 && t != 2 {
		panic("bad")
	}
	for i := 0; i < n; i++ {
		deck = append(deck, c)
	}
}

func init() {
	for _, candy := range []string{"float", "drop", "pop", "man", "heart", "brittle", "cane"} {
		deck = append(deck, card{candy: candy})
	}
	addCard(2, 2, r.color)
	addCard(8, 1, r.color)
	addCard(2, 2, o.color)
	addCard(7, 1, o.color)
	addCard(2, 2, y.color)
	addCard(7, 1, y.color)
	addCard(2, 2, g.color)
	addCard(8, 1, g.color)
	addCard(3, 2, b.color)
	addCard(7, 1, b.color)
	addCard(2, 2, p.color)
	addCard(7, 1, p.color)
}

type game struct {
	players []player
	moves   int
	winner  *player
}

type player struct {
	pos            int
	moves          int
	stucks         int
	candyJumps     int
	candyJumpsBack int
	roads          int
}

func (p *player) move(c *card) (won bool) {
	if c.candy != "" {
		pos, ok := candyPos[c.candy]
		if !ok {
			panic("bad data")
		}
		if pos < p.pos {
			if !*backJump {
				return
			}
			p.candyJumpsBack++
		}
		p.candyJumps++
		p.pos = pos
		return false
	}
	if p.pos >= 0 {
		curs := &board[p.pos]
		if curs.pit && c.color != curs.color {
			p.stucks++
			return false
		}
	}
	n := 1
	if c.double {
		n = 2
	}
	var s *square
	for i := 0; i < n; i++ {
		for {
			p.pos++
			if p.pos >= len(board) {
				return true
			}
			s = &board[p.pos]
			if s.color == c.color {
				break
			}
		}
	}
	if s.roadStart != "" {
		p.pos = s.warpTo
		p.roads++
	}
	return false
}

func newGame(players int) *game {
	g := new(game)
	for i := 0; i < players; i++ {
		g.players = append(g.players, player{pos: -1})
	}
	return g
}

func main() {
	flag.Parse()
	g := newGame(*players)

	var moves []int
	for i := 0; i < *N; i++ {
		g.reset()
		g.run()
		moves = append(moves, g.moves)
		if *verbose {
			fmt.Printf("moves: %v\n", g.moves)
			for _, p := range g.players {
				fmt.Printf("  player: %+v\n", p)
			}
			return
		}
	}
	sort.Ints(moves)
	fmt.Println("min", moves[0])
	fmt.Println("med", moves[*N/2])
	fmt.Println("90p", moves[*N*9/10])
	fmt.Println("max", moves[len(moves)-1])

}

func (g *game) reset() {
	g.moves = 0
	g.winner = nil
	for i := range g.players {
		g.players[i] = player{pos: -1}
	}
}

func (g *game) run() {
	turn := -1
	for {
		turn++
		if turn == len(g.players) {
			turn = 0
		}
		p := &g.players[turn]
		g.moves++
		p.moves++

		was := p.pos

		c := deal()
		if p.move(c) {
			if *verbose {
				fmt.Printf("%s\t%d => %d WIN\n", c, was, p.pos)
			}
			g.winner = p
			return
		}

		if *verbose {
			fmt.Printf("%s\t%d => %d\n", c, was, p.pos)
		}
	}
}
