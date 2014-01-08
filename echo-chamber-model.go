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
}

// returns the culture as a string
func (a *EchoChamberAgent) Culture() string {
	return fmt.Sprintf("%v", a.Features)
}

// required for the simulation interface, called everytime when the agent is activated
func (a *EchoChamberAgent) Act() {

	//TODO new logic
	/*
		// (ii) (a) selects a neighbor for cultural interaction
		other := a.Agent.GetRandomNeighbor().(*AxelrodAgent)
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
	*/

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

func (ml *MultilevelLandscape) Dump() []byte {
	return ml.Base.Dump()
}


type Feature []int

// implementation of the model
type EchoChamberModel struct {
	PhysicalCultures int
	VirtualCultures  int

	Landscape *MultilevelLandscape

	//parameters
	Traits           int     `goabm:"hide"` // don't show these in the stats'
	Features         int     `goabm:"hide"`
	FollowedBlogs    int     `goabm:"hide"`
	POnline          float64 `goabm:"hide"`
	PLookingForBlogs float64 `goabm:"hide"`
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

	var FollowedBlogs = flag.Int("blogs", 4, "number of blogs to follow in the virtual world")
	var POnline = flag.Float64("p-online", 0.5, "probability of being online")
	var PLooking = flag.Float64("p-looking", 0.2, "probability of looking for new blogs (more similar and ditching old ones)")

	var runs = flag.Int("runs", 200, "number of simulation runs")
	var size = flag.Int("size", 100, "number of agents")
	flag.Parse()

	model := &EchoChamberModel{Traits: *traits, Features: *features, FollowedBlogs: *FollowedBlogs, POnline: *POnline, PLookingForBlogs: *PLooking}
	physicalWorld := &goabm.FixedLandscapeWithMovement{Size: *size}
	virtualWorld := &goabm.NetworkLandscape{}

	combinedLandscape := &MultilevelLandscape{Base: physicalWorld, Overlay: virtualWorld}

	sim := &goabm.Simulation{Landscape: combinedLandscape, Model: model, Log: goabm.Logger{StdOut: true}}
	sim.Init()
	fmt.Println("ABM simulation")
	for i := 0; i < *runs; i++ {

		if model.PhysicalCultures == 1 || model.VirtualCultures == 1 {
			return
		}
		sim.Step()

	}
	//fmt.Printf("%v\n",sim.Landscape.GetAgents())

}
