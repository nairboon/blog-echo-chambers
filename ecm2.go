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
import "math/rand"

type Feature []int

type Comment struct {
    Message Feature
    Responses Feature
}

type Blog struct {
   Posts []Comment
   Followers []goabm.AgentID
}

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
	
	
	PStartBlogging float64 `goabm:"hide"`
	Steplength float64 `goabm:"hide"`
	PVeloc float64 `goabm:"hide"`
	
	MyBlog *Blog
}

func (a *EchoChamberAgent) ChangeFeatures(other *EchoChamberAgent) {
 for i := range a.Features {
				if a.Features[i] != other.Features[i] {
					//fmt.Printf("%d influenced %d\n", other.seqnr, a.seqnr)
					a.Features[i] = other.Features[i]
					/*if OtherIsOnline {
					a.OnlineChangeCounter++;
					} else {
					a.OfflineChangeCounter++;
					}*/
					return
				}
			}
}

func (a *EchoChamberAgent) PhysicalInteraction(other *EchoChamberAgent) {

                sim := a.Similarity(other)
                
		if sim >= 0.99 {
			// agents are already equal
			return
		}
		dice := rand.Float64()
		//interact with sim% chance
		if dice <= sim {
			a.ChangeFeatures(other)
		}
}

func (a *EchoChamberAgent) WriteBlog() {

}

func (a *EchoChamberAgent) ReadBlogs() {

}

func (a *EchoChamberAgent) VirtualInteraction() {

//are we blogging?
if a.MyBlog == nil {
// no we're not
 
//let's consider starting a blog
if goabm.RollDice(a.PStartBlogging) {
// setup blog
 a.MyBlog = a.Model.CreateBlog(a)
 // enough for today
 return
}

a.ReadBlogs()
} else {
// we blog

a.WriteBlog()

}
}

// required for the simulation interface, called everytime when the agent is activated
func (a *EchoChamberAgent) Act() {

	dicem := rand.Float64()
	// (i) agent decides to move according to the probability veloc
	if dicem <= a.PVeloc {
		a.MoveRandomly(a.Steplength)
		//fmt.Println("move...")
	}
	
	//l := a.Model.Landscape.(*goabm.FixedLandscapeWithMovement)
// check if we have agent around
other := a.GetRandomNeighbor()
if( other != nil) {
 a.PhysicalInteraction( other.(*EchoChamberAgent))
} else {
// go online otherwise
 a.VirtualInteraction()
}


/*

 var OtherIsOnline bool;
//fmt.Printf("agent (%d - %d)\n",a.PAgent.(*goabm.FLWMAgent).Seqnr,a.VAgent.ID)
var other *EchoChamberAgent
// step 1: decide in which world we interact
dicew:= rand.Float64()
if dicew <= a.Model.POnline {

// we are in the virtual world
// 2.v.1 in vw decide whether looking for blogs
// 2.v.2 select a blog & change traits
diceb:= rand.Float64()
if diceb <= a.Model.PLookingForBlogs {
// first ditch all existing connections
a.VAgent.ClearLinks()
// find at most n(FollowedBlogs) most matching blogs
// v1 exponentional exhaustive search, check every other agent
 similarities := make(map[float64]goabm.GenericAgent)
var keys[]float64
for _, pa := range *a.Model.Landscape.Overlay.GetAgents() {
 potentialAgent := pa.(*EchoChamberAgent)
 if potentialAgent.VAgent.ID != a.VAgent.ID { // no comparing with ourselve
 	sim := a.Similarity(potentialAgent)
 	similarities[sim] = potentialAgent.VAgent

 }
 //fmt.Println("nort: ", a.VAgent.ID, potentialAgent.VAgent.ID)
}//(*goabm.NetworkLandscape).

// sort the other agents according to similarities
for k := range similarities {
    keys = append(keys, k)
}
// highes P first
sort.Sort(sort.Reverse(sort.Float64Slice(keys)))

// pick FollowedBlogs and connect in the virtual space

for i:=0; i< a.Model.FollowedBlogs && i < len(keys);i++ {
k := keys[i]
 blog := similarities[k]
// fmt.Printf("%f\t",k)
 a.VAgent.ConnectTo(&blog)
}

}

// select a blog random blog
// chance to interact is its similarity
 randomLink := a.VAgent.GetRandomLink()
 if randomLink == nil {
  // there is no link at all might want to look for blogs next time??
  return
 }
 

 other = a.Model.Landscape.Overlay.GetAgentById(randomLink.ID + 0*goabm.AgentID( a.Model.Landscape.Base.(*goabm.FixedLandscapeWithMovement).NAgents)).(*EchoChamberAgent)
 		//sim := a.Similarity(other)
 		 //fmt.Printf("sim virtual: %f\n", sim)
 /*if other == nil {
 
 }*
OtherIsOnline = true;
	//other = a.VAgent.GetRandomNeighbor().(*EchoChamberAgent)
} else {
OtherIsOnline = false;
// offline world
// 2.p.1 in pw execute agent logic from axelrods culture model


	
	neighbor := a.PAgent.GetRandomNeighbor()

	if neighbor == nil {
                // there is no agent around to interace :( maybe move a bit?
                 return
         } else {
         	other = neighbor.(*EchoChamberAgent)
         }
}

		// (ii) (a) selects a neighbor for cultural interaction
		
	*/

}

// helper function to determine the similarity between to agents
func (a *EchoChamberAgent) Similarity(other *EchoChamberAgent) float64 {
	c := float64(0.0)
	// count equal traits, final score = shared traits/total traits
	for i := range a.Features {
		if a.Features[i] == other.Features[i] {
			c = c + 1
		}
	}
	//fmt.Printf("sim: %f/%d\n",c,len(a.features))
	return c / float64(len(a.Features))
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

        Blogger map[goabm.AgentID]*Blog
}

func (e *EchoChamberModel) CreateBlog(a EchoChamberAgent) *Blog{
        e.Blogger[a.ID()] = &Blog{}
        return e.Blogger[a.ID()]
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

