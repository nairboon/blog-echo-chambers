/*
A Agent model to simulate opinion dynamics in echo chambers using the goabm library
Copyright 2014 by Remo Hertig <remo.hertig@bluewin.ch>
*/

package main

import "fmt"
import "math/rand"
import "goabm"
import "flag"

// Implementation of the Agent, cultural Traits are stored in features
type EchoChamberAgent struct {
	Features Feature
	PAgent   goabm.FLWMAgenter  //physical agent
	VAgent   goabm.GenericAgent //virtual agent
	
	Model *EchoChamberModel
}

// returns the culture as a string
func (a *EchoChamberAgent) Culture() string {
	return fmt.Sprintf("%v", a.Features)
}

// required for the simulation interface, called everytime when the agent is activated
func (a *EchoChamberAgent) Act() {

var other *EchoChamberAgent
// step 1: decide in which world we interact
dicew:= rand.Float64()
if dicew <= a.Model.POnline {

// we are in the virtual world
// 2.v.1 in vw decide whether looking for blogs
// 2.v.2 select a blog & change traits
diceb:= rand.Float64()
if diceb >= a.Model.PLookingForBlogs {
// find at most n(FollowedBlogs) most matching blogs
}

// select a blog random blog
// chance to interact is its similarity
 randomLink := a.VAgent.GetRandomLink()
 if randomLink == nil {
  // there is no link at all might want to look for blogs next time??
  return
 }
 other = a.Model.Landscape.GetAgentById(randomLink.ID).(*EchoChamberAgent)
	//other = a.VAgent.GetRandomNeighbor().(*EchoChamberAgent)
} else {
// offline world
// 2.p.1 in pw execute agent logic from axelrods culture model

	dicem := rand.Float64()
	// (i) agent decides to move according to the probability veloc
	if dicem <= a.Model.PVeloc {
		a.PAgent.MoveRandomly(a.Model.Steplength)
		//fmt.Println("move...")
	}
	
	neighbor := a.PAgent.GetRandomNeighbor()

	if neighbor == nil {
                // there is no agent around to interace :( maybe move a bit?
                 return
         } else {
         	other = neighbor.(*EchoChamberAgent)
         }
}

		// (ii) (a) selects a neighbor for cultural interaction

		sim := a.Similarity(other)
		if sim >= 0.99 {
			// agents are already equal
			return
		}
		dice := rand.Float32()
		//interact with sim% chance
		if dice <= sim {
			for i := range a.Features {
				if a.Features[i] != other.Features[i] {
					//fmt.Printf("%d influenced %d\n", other.seqnr, a.seqnr)
					a.Features[i] = other.Features[i]
					return
				}

			}
		}
	

}

// helper function to determine the similarity between to agents
func (a *EchoChamberAgent) Similarity(other *EchoChamberAgent) float32 {
	c := float32(0.0)
	// count equal traits, final score = shared traits/total traits
	for i := range a.Features {
		if a.Features[i] == other.Features[i] {
			c = c + 1
		}
	}
	//fmt.Printf("sim: %f/%d\n",c,len(a.features))
	return c / float32(len(a.Features))
}

type MultilevelLandscape struct {
	Base       goabm.Landscaper
	Overlay goabm.Landscaper
}

func (ml *MultilevelLandscape) Init(arg goabm.Modeler) {
	ml.Base.Init(arg)
}

func (ml *MultilevelLandscape) GetAgents() []goabm.Agenter {
	return ml.Base.GetAgents()
}

func (ml *MultilevelLandscape) GetAgentById(id goabm.AgentID) goabm.Agenter {
	return ml.Base.GetAgentById(id)
}

func (ml *MultilevelLandscape) Dump() []byte {
	return ml.Base.Dump()
}


type Feature []int

// implementation of the model
type EchoChamberModel struct {
	PhysicalCultures int
	VirtualCultures  int

	Landscape *MultilevelLandscape `goabm:"hide"` 

	//parameters
	Traits           int     `goabm:"hide"` // don't show these in the stats'
	Features         int     `goabm:"hide"`
	FollowedBlogs    int     `goabm:"hide"`
	POnline          float64 `goabm:"hide"`
	PLookingForBlogs float64 `goabm:"hide"`
	Steplength float64 `goabm:"hide"`
	PVeloc float64 `goabm:"hide"`
}

func (m *EchoChamberModel) Init(l interface{}) {
	m.Landscape = l.(*MultilevelLandscape)
}

func (m *EchoChamberModel) CreateAgent(agenter interface{}) goabm.Agenter {

	agent := &EchoChamberAgent{PAgent: agenter.(goabm.FLWMAgenter), VAgent: goabm.GenericAgent{}}

	f := make(Feature, m.Features)
	for i := range f {
		f[i] = rand.Intn(m.Traits)
	}
	agent.Features = f
	agent.Model = m
	return agent
}

func (m *EchoChamberModel) LandscapeAction() {
	m.PhysicalCultures = m.CountCultures(m.Landscape.Base)
	m.VirtualCultures = m.CountCultures(m.Landscape.Overlay)

}

func (m *EchoChamberModel) CountCultures(ls goabm.Landscaper) int {
	cultures := make(map[string]int)
	for _, b := range ls.GetAgents() {
		a := b.(*EchoChamberAgent)
		cul := a.Culture()
		if _, ok := cultures[cul]; ok {
			cultures[cul] = 1
		} else {
			cultures[cul] = cultures[cul] + 1
		}
	}
	return len(cultures)
}

func main() {
	goabm.Init()

	var traits = flag.Int("traits", 5, "number of cultural traits per feature")
	var features = flag.Int("features", 5, "number of cultural features")
		var size = flag.Int("size", 10, "size (width/height) of the landscape")
	
	var probveloc = flag.Float64("pveloc", 0.05, "probability that an agent moves")
	var steplength = flag.Float64("steplength", 0.1, "maximal distance a agent can travel per step")
	var sight = flag.Float64("sight", 1, "radius in which agent can interact")

	var FollowedBlogs = flag.Int("blogs", 4, "number of blogs to follow in the virtual world")
	var POnline = flag.Float64("p-online", 0.5, "probability of being online")
	var PLooking = flag.Float64("p-looking", 0.2, "probability of looking for new blogs (more similar and ditching old ones)")

	var runs = flag.Int("runs", 200, "number of simulation runs")
	var numAgents = flag.Int("agents", 100, "number of agents to simulate")
	flag.Parse()

 
	model := &EchoChamberModel{Traits: *traits, Features: *features, PVeloc: *probveloc, Steplength: *steplength,
	 FollowedBlogs: *FollowedBlogs, POnline: *POnline, PLookingForBlogs: *PLooking}
	physicalWorld := &goabm.FixedLandscapeWithMovement{Size: *size, NAgents: *numAgents,Sight:*sight}
	virtualWorld := &goabm.NetworkLandscape{}

	combinedLandscape := &MultilevelLandscape{Base: physicalWorld, Overlay: virtualWorld}

	sim := &goabm.Simulation{Landscape: combinedLandscape, Model: model, Log: goabm.Logger{StdOut: true}}
		fmt.Println("ABM simulation")
	sim.Init()

	for i := 0; i < *runs; i++ {

		if model.PhysicalCultures == 1 || model.VirtualCultures == 1 {
			return
		}
		sim.Step()

	}
	//fmt.Printf("%v\n",sim.Landscape.GetAgents())

}
