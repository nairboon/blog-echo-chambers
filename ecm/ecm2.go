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

import "os"

import "sort"
import "math"


import "runtime/pprof"
import "log"
import "flag"
import . "flache/ecm/model"

import "code.google.com/p/probab/dst"

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
	 PLooking, PStartBlogging, PRespondBlogPost float64,
	RSubscribedBlogs IntRange, RSimilarityConfortLevel FloatRange,
	ret chan SimRes, rules goabm.Ruleset,
	pfUnderstanding BPFP,
	pfOnline, pfRead, pfRespond  NPFP) {

	model := &EchoChamberModel{
		NTraits:                 traits,
		NFeatures:               features,
		PVeloc:                  probveloc,
		Steplength:              steplength,
		PStartBlogging:          PStartBlogging,
		RSubscribedBlogs:        RSubscribedBlogs,
		RSimilarityConfortLevel: RSimilarityConfortLevel,
		PFOnline: pfOnline.Pf,//dst.Beta(pfOnline.α.Var, pfOnline.β.Var),
		PFConsumptive: pfRead.Pf,
		PFExpressive: pfRespond.Pf,
		
		//PFAI: dst.Beta(pfActiveInteraction.α.Var, pfActiveInteraction.β.Var),
		PFU: dst.Beta(pfUnderstanding.α.Var, pfUnderstanding.β.Var)}

	model.Ruleset = rules
	//fmt.Printf("rule: %v", rules)

	sim := &goabm.Simulation{Landscape: &goabm.FixedLandscapeWithMovement{Size: size, NAgents: numAgents, Sight: sight},
		Model: model, Log: goabm.Logger{StdOut: false}}
	sim.Init()

	nvar := 70
	r := make([]float64, runs)
	//last := 9.0

        variance := 0.0
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
			variance = stats.StatsSampleVariance(slidingWindow)
			//fmt.Printf("%f\n",variance)

			// if we have such a smal variance the model is considered
			// stable and we abort computation here to save cpu resources
			if variance < 0.00005 {
				//fmt.Printf("stop it at %d %f\n",i, variance)
				break
			}
		}
		if (i+1) >= runs {
		//fmt.Printf("after %d variance did not converge.. %f\n",runs, variance)
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




type Parameters struct {
	Probabilities []BPFP
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
	
	//pfOnline := p.Probabilities[0]
	//pfActiveInteraction := p.Probabilities[1]
	
	/*pfOnline := BPFP{α: DiscreteVarWithLimit{Var:1.8}, 
        β: DiscreteVarWithLimit{ Var:2.1}}
        
        pfRead := BPFP{α: DiscreteVarWithLimit{Var:1.8}, 
        β: DiscreteVarWithLimit{ Var:2.1}}
        
        pfRespond := BPFP{α: DiscreteVarWithLimit{Var:1.8}, 
        β: DiscreteVarWithLimit{ Var:2.1}}*/
        
        pfOnline := NPFP{μ: 2.7, σ: 2.0, Base: Range{Min:0, Max: 10}}
        pfOnline.Init()
        pfRead := NPFP{μ: 7.69, σ: 2.26 , Base: Range{Min:0, Max: 10}}
        pfRead.Init()
        pfRespond := NPFP{μ: 10.35, σ:5.68, Base: Range{Min:0, Max: 10} }
        pfRespond.Init()
        
	pfUnderstanding := p.Probabilities[0]
	
	MinConfort := 0.4
	PRespondBlogPost := 0.2

	PStartBlogging := 0.1


	features := 30
	traits := 30

	size := 200
	numAgents := 150

	runs := 400

	probveloc := 0.15
	steplength := 1.5
	sight := 1.0

	PLooking := 0.2

	RSubscribedBlogs := IntRange{1, 10}

	RSimilarityConfortLevel := FloatRange{MinConfort, 1}

	NCPU := 2
	innerRuns :=  2 // multiple of NCPU!
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
				probveloc, steplength, sight, PLooking,
				PStartBlogging, PRespondBlogPost, RSubscribedBlogs,
				RSimilarityConfortLevel,
				resc, p.Rules, pfUnderstanding, pfOnline, pfRead, pfRespond)
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

type NPFP struct {
        μ  float64
        σ float64
        Base Range
        f func()float64
}
func (n*NPFP) Init() {
        n.f = dst.Normal(n.μ, n.σ)
}

func (n*NPFP) Pf() float64 {
        // sample from f until in range Base
        s := -9999.0
               // fmt.Printf("s:%f > %f < %f",s,n.Base.Min, n.Base.Max )
               c:=0
        for (s < n.Base.Min || s > n.Base.Max) {
         s = n.f()
         c++
        //fmt.Printf("jajaj %f",s)
        }
        //fmt.Printf("c:%d\n",c)
        // normalize [0,1]
        r := (s- n.Base.Min)/(n.Base.Max- n.Base.Min)
        //fmt.Printf("pf()=%f\t",r)
        return r
}

type BPFP struct {
        α DiscreteVarWithLimit
        β DiscreteVarWithLimit
}

func sampleBPF(n int,start BPFP) []BPFP {
sizeα := start.α.Max - start.α.Min
stepα := sizeα / (float64(n)/2)
sizeβ := start.β.Max - start.β.Min
stepβ := sizeβ / (float64(n)/2)
 
 r:=  make([]BPFP, n)
 
 c:=0
for i := start.α.Min; i <= start.α.Max; i += stepα {
        for j := start.β.Min; j <= start.β.Max; j += stepβ {
          r[c].α.Var = i
          r[c].β.Var = j
          c++
        }
}
return r	

}

func randomBPF(start BPFP) BPFP {
 r := BPFP{}
 r.α.Var = goabm.Random(start.α.Min, start.α.Max)
  r.β.Var = goabm.Random(start.β.Min, start.β.Max)
  return r
}

func MCsampleBPF(n int,start BPFP) []BPFP {
//sizeα := start.α.Max - start.α.Min
//stepα := sizeα / (float64(n)/2)
//sizeβ := start.β.Max - start.β.Min
//stepβ := sizeβ / (float64(n)/2)
 
 r:=  make([]BPFP, n)
 
 c:=0
for i := 0; i < n; i++ {
          r[c] = randomBPF(start)
          c++
}
return r	
}

func samplePS(initial Parameters, samples int) []Parameters {
	/* samples per probability */
	//step := samples / float64(len(initial.Probabilities ))

        d1 := initial.Probabilities[0].α.Max - initial.Probabilities[0].α.Min
        d2 := initial.Probabilities[0].β.Max - initial.Probabilities[0].β.Min
        
        comb := math.Log(float64(samples))+3
	step1 := d1 / comb
	step2 := d2 / (comb)
		
	res := make([]Parameters, int(samples)+1000)
	
	for i := 0; i < len(initial.Probabilities ); i++ {
	        
	}
	//fmt.Printf("n: %f l: %d", n, len(res))
	c := 0
	
	for i := initial.Probabilities[0].α.Min; i < initial.Probabilities[0].α.Max; i += step1 {
		for j := initial.Probabilities[0].β.Min; j < initial.Probabilities[0].β.Max; j += step2 {
		res[c].Probabilities = []BPFP{BPFP{α: DiscreteVarWithLimit{Var:i},β:DiscreteVarWithLimit{Var:j}}}
		res[c].Ranges = initial.Ranges
				res[c].Rules = initial.Rules
				res[c].Discrete = initial.Discrete
		c++
		}
	
	}
	fmt.Printf("%d, %f samples\n",c,comb)
	/*for i := initial.Probabilities[0]; i < 1.0; i += resolution {
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
	}*/
	return res[:c]
}
        
func MCsamplePS(initial Parameters, samples int) []Parameters {

/*
 draw approximately n samples for each PDF
*/

//fmt.Printf("mc sample:%d\n",n)
res := make([]Parameters, int(samples))
c:=0
for i := 0; i < samples; i++ {
	        

          
		                // use prefedined ranges
		                res[c].Probabilities = []BPFP{
		                randomBPF(initial.Probabilities[0]),
		                randomBPF(initial.Probabilities[1]),
		                randomBPF(initial.Probabilities[2])}	                
		                
				res[c].Ranges = initial.Ranges
				res[c].Rules = initial.Rules
				res[c].Discrete = initial.Discrete
				c++
	        
	}

return res[:c]

}

type Results struct {
  Best []SimRunRes
}

func (r *Results) Init(n int) { 
 r.Best = make([]SimRunRes,n)

 for i:= range r.Best {
  r.Best[i].Score = 9999.9
 }
  //fmt.Printf("b: %v , len %d\n",r, len(r.Best))
}
func (r *Results) Check(nr SimRunRes) { 
//fmt.Printf("best: %v\n",r.Best)
if nr.Score < r.Best[len(r.Best)-1].Score {
// better score than worst
r.Best[len(r.Best)-1] = nr
sort.Sort(ByScore(r.Best))
}

}

type SimRunRes struct {
 Parameters
 Score float64
}

type ByScore []SimRunRes

func (a ByScore) Len() int           { return len(a) }
func (a ByScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByScore) Less(i, j int) bool { return a[i].Score < a[j].Score }

func main() {

//initialize the goabm library (logs & flags)
	goabm.Init()

var memprofile = flag.String("memprofile", "", "write memory profile to this file")
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	
//f := dst.Beta(2.0,2.0)

	

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

        /*
        pf(Online) = α, β
        pf(ActiveInteraction) = α, β
        pf(Understanding) = α, β
        */
        
        
        /*pfOnline := BPFP{α: DiscreteVarWithLimit{Min:2.5, Max: 12, Var:1.5}, 
        β: DiscreteVarWithLimit{Min:0.1, Max: 7, Var:2}}
        
        pfActiveInteraction := BPFP{α: DiscreteVarWithLimit{Min:0.5, Max: 7, Var:2} , 
        β: DiscreteVarWithLimit{Min:0.5, Max: 9, Var:2}}*/
        
        pfUnderstanding := BPFP{α: DiscreteVarWithLimit{Min:0.5, Max: 100, Var:1.8}, 
        β: DiscreteVarWithLimit{Min:0.5, Max: 100, Var:2.1}}
        
        /*
        search space = 6 dimensions
        */

	p := Parameters{Probabilities: []BPFP{/*pfOnline,pfActiveInteraction,*/pfUnderstanding,pfUnderstanding,pfUnderstanding}, Rules: rules}
	
	// parameter search
	samples := 20

	pars := samplePS(p, samples)
	fmt.Printf("size of ps: %d", len(pars))
	mt := MyTarget{}
	// run model for each parameter
	fmt.Printf("run, score, pfOnline, pfActiveInteraction, pfUnderstanding\n")
	
	// keep n best
	keep:= 15
	
	best := Results{}
	best.Init(keep)
	
	
	
	fon,_ := os.Create("pfOnline.csv")
	fai,_ := os.Create("pfAI.csv")
	fu,_ := os.Create("pfU.csv")
	
	 fmt.Fprintf(fon,"score, α, β\n")
	 fmt.Fprintf(fai,"score, α, β\n")
	 fmt.Fprintf(fu,"score, α, β\n")
            
	for i, p := range pars {
		/*if 0 == p.Probabilities[0] {
		continue
		}*/
		r := mt.Run(p)

/*
		fmt.Printf("%d,\t %f,\t (α: %.2f,β: %.2f),\t (α: %.2f,β: %.2f),\t (α: %.2f,β: %.2f)\n", i, r, 
		p.Probabilities[0].α.Var, p.Probabilities[0].β.Var,
		p.Probabilities[1].α.Var, p.Probabilities[1].β.Var,
		p.Probabilities[2].α.Var, p.Probabilities[2].β.Var,)*/
		
		fmt.Printf("%d,\t %f,\t (α: %.2f,β: %.2f),\t \n", i, r, 
		p.Probabilities[0].α.Var, p.Probabilities[0].β.Var)
		
		srr := SimRunRes{Parameters: p, Score: r}
		best.Check(srr)
                
                //fmt.Fprintf(fon,"%f, %f, %f\n",r, p.Probabilities[0].α.Var, p.Probabilities[0].β.Var)
               // fmt.Fprintf(fai,"%f, %f, %f\n",r, p.Probabilities[1].α.Var, p.Probabilities[2].β.Var)
                fmt.Fprintf(fu,"%f, %f, %f\n",r, p.Probabilities[0].α.Var, p.Probabilities[0].β.Var)
                
	}
	
	/*ioutil.WriteFile("pfOnline.csv",[]byte(logpfOnline),0777)
		ioutil.WriteFile("pfAI.csv",[]byte(logpfActiveInteraction),0777)
			ioutil.WriteFile("pfU.csv",[]byte(logpfUnderstanding),0777)/*/
			
	fmt.Printf("Best res:\n")
	
	asA1:=0.0
	asB1:=0.0
	/*	asA2:=0.0
	asB2:=0.0
		asA3:=0.0
	asB3:=0.0*/
	
	for i:=0;i<keep;i++ {
	if best.Best[i].Score > 10 {
	continue
	}
	/* fmt.Printf("#%d, score: %f\t (α: %.2f,β: %.2f),\t (α: %.2f,β: %.2f),\t (α: %.2f,β: %.2f)\n",i, best.Best[i].Score,
	 best.Best[i].Probabilities[0].α.Var, best.Best[i].Probabilities[0].β.Var,
		best.Best[i].Probabilities[1].α.Var, best.Best[i].Probabilities[1].β.Var,
		best.Best[i].Probabilities[2].α.Var, best.Best[i].Probabilities[2].β.Var,)*/
		
	fmt.Printf("#%d, score: %f\t (α: %.2f,β: %.2f),\n",i, best.Best[i].Score,
	 best.Best[i].Probabilities[0].α.Var, best.Best[i].Probabilities[0].β.Var)
		
	asA1 += best.Best[i].Probabilities[0].α.Var
	//	asA2 += best.Best[i].Probabilities[1].α.Var
	//		asA3 += best.Best[i].Probabilities[2].α.Var
			
	asB1 += best.Best[i].Probabilities[0].β.Var
	//	asB2 += best.Best[i].Probabilities[1].β.Var
	//		asB3 += best.Best[i].Probabilities[2].β.Var
	}
	
	avgA1 := asA1 / float64(keep)
	//avgA2 := asA2 / float64(keep)
	//avgA3 := asA3 / float64(keep)
	avgB1 := asB1 / float64(keep)
	//avgB2 := asB2 / float64(keep)
	//avgB3 := asB3 / float64(keep)
	

	/*fmt.Printf("avg:\t (α: %.2f,β: %.2f),\t (α: %.2f,β: %.2f),\t (α: %.2f,β: %.2f)\n",
	avgA1,avgB1, avgA2, avgB2, avgA3, avgB3)*/
	fmt.Printf("avg:\t (α: %.2f,β: %.2f),\t \n",
	avgA1,avgB1)
	
	
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
		return
	}
}
