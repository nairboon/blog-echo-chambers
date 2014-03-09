/*
A Agent model to simulate opinion dynamics in echo chambers using the goabm library
Copyright 2014 by Remo Hertig <remo.hertig@bluewin.ch>

v2

model:

agent has relation to other agents r
a relation consists of an series of interactions

agent can have at most N relation

the agent is attracted to other with p(similarity)
sort agents according to score function -> nInter * nDaysSince * Similarity
however he will also interact with long known "friends": p( nInteractions^f=1 * nDaysSinceLastI^-f=1)

c1: condition on which he will seach a new interaction partner
case 1: nearby new neighbor
        if there is unknown neighbor -> try
                otherwise internet
case 2: "internet"

for every feature we keep track of interactions & potential partners
if a certain feature's count falls below a treshold, the agent starts to search for potential partners
 f[needtointeract] = e^(x/f=1) -> triggers search for this feature

 on each interactions: update similarity (overall) -> especially for feature list
                        interact (change feature)

6 factor


/// v2

agent loop
        * is there physical ip?
                *interact normally
        * no: blog loop
                * evaluate blogs
                * search new blog
                * read postings
                * comment

*/

package main

import "goabm"
import "fmt"

import "math"

import . "flache/ecm/model"

//import "time"
import "runtime"
import "github.com/GaryBoone/GoStats/stats"

type SimRes struct {
	Cultures           int
	OnlineInteraction  int
	OfflineInteraction int
	TotalEchoChambers  int
	EchoChamberRatio   float64
	Events             int
}

func simRun(traits, features, size, numAgents, runs int,
	probveloc, steplength, sight,
	POnline, PLooking, PStartBlogging, PWriteBlogPost, PRespondBlogPost float64,
	RSubscribedBlogs IntRange, RSimilarityConfortLevel FloatRange,
	ret chan SimRes, rules goabm.Ruleset) {

	model := &EchoChamberModel{
		NTraits:                 traits,
		NFeatures:               features,
		PVeloc:                  probveloc,
		Steplength:              steplength,
		PStartBlogging:          PStartBlogging,
		PWriteBlogPost:          PWriteBlogPost,
		PRespondBlogPost:        PRespondBlogPost,
		RSubscribedBlogs:        RSubscribedBlogs,
		RSimilarityConfortLevel: RSimilarityConfortLevel,
		POnline:                 POnline}

	model.Ruleset = rules
	//fmt.Printf("rule: %v", rules)

	sim := &goabm.Simulation{Landscape: &goabm.FixedLandscapeWithMovement{Size: size, NAgents: numAgents, Sight: sight},
		Model: model, Log: goabm.Logger{StdOut: false}}
	sim.Init()

	nvar := 100
	r := make([]float64, runs)
	//last := 9.0

	for i := 0; i < runs; i++ {
		//fmt.Printf("Step #%d, Events:%d, Cultures:%d\n", i, sim.Stats.Events, model.Cultures)
		/*if model.Cultures == 1 {
			sim.Stop()
		fmt.Printf("Stimulation prematurely done\n")
				break
			}*/
		sim.Step()
		t := model.EchoChamberRatio

		//last = t
		r[i] = t
		//fmt.Printf("%d %f %d\n",c,t,runs)

		// check for variance in last N steps
		if i > nvar {
			slidingWindow := r[i-nvar : i]
			variance := stats.StatsSampleVariance(slidingWindow)
			//fmt.Printf("%f\n",variance)

			// if we have such a smal variance the model is considered
			// stable and we abort computation here to save cpu resources
			if variance < 0.000001 {
				//fmt.Printf("stop it at %d %f\n",i, variance)
				break
			}
		}
	}
	sim.Stop()

	ret <- SimRes{Cultures: model.Cultures,
		OnlineInteraction:  model.OnlineInteraction,
		OfflineInteraction: model.OfflineInteraction,
		TotalEchoChambers:  model.TotalEchoChambers,
		EchoChamberRatio:   model.EchoChamberRatio,
		Events:             sim.Stats.Events}
}

type DiscreteVarWithLimit struct {
	Var float64
	Min float64
	Max float64
}

type Range struct {
	Var FloatRange
	Min float64
	Max float64
}

type Parameters struct {
	Probabilities []float64
	Discrete      []DiscreteVarWithLimit
	Ranges        []Range
	Rules         goabm.Ruleset
}

type TargetFunction interface {
	Run(Parameters) float64
}

type MyTarget struct {
}

func (tf MyTarget) Run(p Parameters) float64 {

	/*if len(p.Probabilities) != 3 {
	  fmt.Printf("%v",p)
	  //return 100.0
	  panic("len is not 3")
	  }*/
	//PStartBlogging := p.Probabilities[0]
	MinConfort := p.Probabilities[0]
	PRespondBlogPost := p.Probabilities[1]

	POnline := p.Probabilities[1]

	PStartBlogging := 0.1
	PWriteBlogPost := 0.3

	features := 20
	traits := 50
	size := 30
	numAgents := 10
	runs := 1000

	probveloc := 0.15
	steplength := 1.5
	sight := 1.0

	PLooking := 0.2

	RSubscribedBlogs := IntRange{1, 5}

	RSimilarityConfortLevel := FloatRange{MinConfort, 1}

	NCPU := 1
	innerRuns := NCPU * 4 // multiple of NCPU!
	runtime.GOMAXPROCS(NCPU)
	scoreSum := 0.0
	l := make([]float64, innerRuns)

	tevents := 0
	//start := time.Now()

	resc := make(chan SimRes, NCPU)

	//dispatch workers
	i := 0
	for i < innerRuns {
		for j := 0; j < NCPU; j++ {
			i++
			go simRun(traits, features, size, numAgents, runs,
				probveloc, steplength, sight, POnline, PLooking,
				PStartBlogging, PWriteBlogPost, PRespondBlogPost, RSubscribedBlogs,
				RSimilarityConfortLevel,
				resc, p.Rules)
		}
	}

	for i := 0; i < innerRuns; i++ {

		// collect results
		r := <-resc

		ratio := r.EchoChamberRatio
		target := 0.64

		tevents += r.Events
		score := math.Abs(target - ratio)
		scoreSum += score
		l[i] = score
		//fmt.Printf("score: %f  ratio: %f\n", score, ratio)
	}
	/* usedTime := time.Since(start)
	   eps := float64(tevents) / usedTime.Seconds()
	   fmt.Printf("%f events/s\t",eps)*/
	return scoreSum / float64(innerRuns)
}

func samplePS(initial Parameters, resolution float64) []Parameters {
	/* PS_probabilities = 1/resolution^N */
	n := 1.0 / resolution

	res := make([]Parameters, int(math.Pow(n+1, 3)))
	//fmt.Printf("n: %f l: %d", n, len(res))
	c := 0
	for i := initial.Probabilities[0]; i < 1.0; i += resolution {
		for j := initial.Probabilities[1]; j < 1.0; j += resolution {
			for k := initial.Probabilities[2]; k < 1.0; k += resolution {
				//fmt.Printf("c: %d, i:%f j:%f, k:%f\n", c, i, j, k)
				res[c].Probabilities = []float64{i, j, k}

				res[c].Ranges = initial.Ranges
				res[c].Rules = initial.Rules
				res[c].Discrete = initial.Discrete
				c++
			}
		}
	}

	return res[:c]
}

func main() {
	//initialize the goabm library (logs & flags)
	goabm.Init()

	/*
	    combine MC with Sa?
	   inner loop is MC sampling?
	*/
	//fmt.Printf("#Parameter search...\n");

	//fmt.Printf("%d, %d, %d, %d, %d, %f, %f\n", traits, features, r.CultureDiff, r.OnlineCultures, r.OfflineCultures, r.AvgOnline, r.AvgOffline)
	/*
		PStartBlogging := 0.005
		PWriteBlogPost := 0.2
		PRespondBlogPost := 0.1
	*/
	/*
		// simulated annealing
		p := Parameters{
			Probabilities: []float64{PStartBlogging,
				PWriteBlogPost,
				PRespondBlogPost},
		}

		sa := SimulatedAnnealing{Temp: 100, CoolingRate: 0.01, KMax: 400}
		sa.Parameters = p
		sa.TF = mt
		r := sa.Run()
		fmt.Printf("best score %f for %v\n", r.Energy, r)*/
	rules := goabm.Ruleset{}
	rules.Init()
	rules.SetRule("movement", true)            // not implemented
	rules.SetRule("transmission_error", false) // not implemented
	rules.SetRule("only_stable_models", false) // not implemented

	p := Parameters{Probabilities: []float64{0.1, 0.1, 0.1}, Rules: rules}
	// parameter search
	resolution := 0.1

	pars := samplePS(p, resolution)
	fmt.Printf("size of ps: %d", len(pars))
	mt := MyTarget{}
	// run model for each parameter
	fmt.Printf("run, score, minc, respond, ponline\n")
	for i, p := range pars {

		/*if 0 == p.Probabilities[0] {
		continue
		}*/
		r := mt.Run(p)

		fmt.Printf("%d,\t %f,\t %f,\t %f,\t %f\n", i, r, p.Probabilities[0],
			p.Probabilities[1],
			p.Probabilities[2])

	}
}
