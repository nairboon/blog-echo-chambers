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
*/

package main

import "goabm"
import "fmt"
import "math/rand"

type Feature []int

// Implementation of the Agent, cultural Traits are stored in features
type EchoChamberAgent struct {
	Features Feature
	*goabm.FLWMAgent  `json:"Agent"`
	
	Model *EchoChamberModel `json:"-"`
	FreeNode bool `json:"FreeNode"`
	
	OfflineChangeCounter uint
	OnlineChangeCounter uint
	
	NInteractionF float64
	NDaysSinceLastInteractionF float64
	NNeedToInteractF float64
	SimilarityF float64
	NMaxRelations uint
}

type EchoChamberModel struct {
	PhysicalCultures int
	VirtualCultures  int

	Landscape goabm.Landscaper

	//parameters
	NTraits           int     `goabm:"hide"` // don't show these in the stats'
	NFeatures         int     `goabm:"hide"`

	POnline          float64 `goabm:"hide"`
	Steplength float64 `goabm:"hide"`
	PVeloc float64 `goabm:"hide"`
}


func (e *EchoChamberModel) Init(l interface{}) {
	e.Landscape = l.(goabm.Landscaper)
}


func (a *EchoChamberModel) CreateAgent(agenter interface{}) goabm.Agenter {

	agent := &EchoChamberAgent{FLWMAgent: agenter.(*goabm.FLWMAgent)}

	f := make(Feature, a.NFeatures)
	for i := range f {
		f[i] = rand.Intn(a.NTraits)
	}
	agent.Features = f

	agent.Model = a
	return agent
}

func (a *EchoChamberModel) LandscapeAction() {
	//a.Cultures = a.CountCultures()
}

func (a *EchoChamberModel) CountCultures() int {
	/*cultures := make(map[string]int)
	for _, b := range *a.Landscape.GetAgents() {
		a := b.(*EchoChamberAgent)
		cul := a.Culture()
		if _, ok := cultures[cul]; ok {
			cultures[cul] = 1
		} else {
			cultures[cul] = cultures[cul] + 1
		}
	}
	return len(cultures)*/
	return 1
}

type SimRes struct
{
 AvgOnline float64
 AvgOffline float64
 CultureDiff int
 OnlineCultures int
 OfflineCultures int
}


func simRun(traits, features, size, numAgents, runs, FollowedBlogs int, probveloc, steplength, sight,POnline,PLooking float64) SimRes {

model := &EchoChamberModel{NTraits: traits, NFeatures: features, PVeloc: probveloc, Steplength: steplength, POnline: POnline}
	 
sim := &goabm.Simulation{Landscape: &goabm.FixedLandscapeWithMovement{Size: size, NAgents: numAgents,Sight:sight},
 Model: model , Log: goabm.Logger{StdOut: true}}
	sim.Init()
	for i := 0; i < runs; i++ {
		//fmt.Printf("Step #%d, Events:%d, Cultures:%d\n", i, sim.Stats.Events, model.Cultures)
		/*if model.Cultures == 1 {
		sim.Stop()
	fmt.Printf("Stimulation prematurely done\n")
			break
		}*/
		sim.Step()

	}
		sim.Stop()
		
		
return SimRes{}

}


func main() {
       //initialize the goabm library (logs & flags)
	goabm.Init()
	        

 features := 15
        size:= 10
        numAgents := 30
        runs := 200
        FollowedBlogs := 4
        probveloc := 0.15
        steplength := 0.2 
        sight := 1.0
        POnline := 0.5
        PLooking := 0.2
        traits := 10
	goabm.Init()
	//fmt.Printf("#Parameter search...\n");
	
	
	r := simRun(traits, features, size, numAgents, runs, FollowedBlogs, probveloc, steplength, sight,POnline,PLooking)
	
	fmt.Printf("%d, %d, %d, %d, %d, %f, %f\n",traits,features, r.CultureDiff, r.OnlineCultures, r.OfflineCultures,r.AvgOnline,r.AvgOffline)
	/*
	Min1 := 10; 
	Max1 := 10;
	
	Min2 := 10; 
	Max2 := 10;
	
	fmt.Printf("traits,f, cdiff,onc,offc,avgon,avgoff\n")
	for i:= Min1; i< Max1; i++ {
	for j:= Min2; j< Max2; j++ {
	
	r := simRun(i, features, size, numAgents, runs, FollowedBlogs, probveloc, steplength, sight,POnline,PLooking)
	
	fmt.Printf("%d, %d, %d, %d, %d, %f, %f\n",i,j, r.CultureDiff, r.OnlineCultures, r.OfflineCultures,r.AvgOnline,r.AvgOffline)
	} }*/
}

