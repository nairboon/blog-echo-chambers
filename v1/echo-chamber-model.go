/*
A Agent model to simulate opinion dynamics in echo chambers using the goabm library
Copyright 2014 by Remo Hertig <remo.hertig@bluewin.ch>
*/

package flache

import "fmt"
import "math/rand"

import "goabm"
import "flag"
import "sort"
import "os"
import "log"
import "runtime/pprof"

// Implementation of the Agent, cultural Traits are stored in features
type EchoChamberAgent struct {
	goabm.GenericAgent
	Features Feature
	PAgent   goabm.FLWMAgenter  `json:"Agent"` //physical agent
	VAgent   goabm.GenericAgent `json:"-"`     //virtual agent

	Model    *EchoChamberModel `json:"-"`
	FreeNode bool              `json:"FreeNode"`

	OfflineChangeCounter uint
	OnlineChangeCounter  uint
}

// returns the culture as a string
func (a *EchoChamberAgent) Culture() string {
	return fmt.Sprintf("%v", a.Features)
}

// required for the simulation interface, called everytime when the agent is activated
func (a *EchoChamberAgent) Act() {

	var OtherIsOnline bool
	//fmt.Printf("agent (%d - %d)\n",a.PAgent.(*goabm.FLWMAgent).Seqnr,a.VAgent.ID)
	var other *EchoChamberAgent
	// step 1: decide in which world we interact
	dicew := rand.Float64()
	if dicew <= a.Model.POnline {

		// we are in the virtual world
		// 2.v.1 in vw decide whether looking for blogs
		// 2.v.2 select a blog & change traits
		diceb := rand.Float64()
		if diceb <= a.Model.PLookingForBlogs {
			// first ditch all existing connections
			a.VAgent.ClearLinks()
			// find at most n(FollowedBlogs) most matching blogs
			// v1 exponentional exhaustive search, check every other agent
			similarities := make(map[float64]goabm.GenericAgent)
			var keys []float64
			for _, pa := range *a.Model.Landscape.Overlay.GetAgents() {
				potentialAgent := pa.(*EchoChamberAgent)
				if potentialAgent.VAgent.ID != a.VAgent.ID { // no comparing with ourselve
					sim := a.Similarity(potentialAgent)
					similarities[sim] = potentialAgent.VAgent

				}
				//fmt.Println("nort: ", a.VAgent.ID, potentialAgent.VAgent.ID)
			} //(*goabm.NetworkLandscape).

			// sort the other agents according to similarities
			for k := range similarities {
				keys = append(keys, k)
			}
			// highes P first
			sort.Sort(sort.Reverse(sort.Float64Slice(keys)))

			// pick FollowedBlogs and connect in the virtual space

			for i := 0; i < a.Model.FollowedBlogs && i < len(keys); i++ {
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

		other = a.Model.Landscape.Overlay.GetAgentById(randomLink.ID + 0*goabm.AgentID(a.Model.Landscape.Base.(*goabm.FixedLandscapeWithMovement).NAgents)).(*EchoChamberAgent)
		//sim := a.Similarity(other)
		//fmt.Printf("sim virtual: %f\n", sim)
		/*if other == nil {

				 }*/
		OtherIsOnline = true
		//other = a.VAgent.GetRandomNeighbor().(*EchoChamberAgent)
	} else {
		OtherIsOnline = false
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
	dice := rand.Float64()
	//interact with sim% chance
	if dice <= sim {
		for i := range a.Features {
			if a.Features[i] != other.Features[i] {
				//fmt.Printf("%d influenced %d\n", other.seqnr, a.seqnr)
				a.Features[i] = other.Features[i]
				if OtherIsOnline {
					a.OnlineChangeCounter++
				} else {
					a.OfflineChangeCounter++
				}
				return
			}

		}
	}

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

type MultilevelLandscape struct {
	Base          goabm.Landscaper
	Overlay       goabm.Landscaper
	PhysToVirtual map[goabm.AgentID]goabm.AgentID
	VirtualToPhys map[goabm.AgentID]goabm.AgentID
}

func (ml *MultilevelLandscape) Init(arg goabm.Modeler) {
	/* ml.PhysToVirtual = make(map[goabm.AgentID]goabm.AgentID)
	   ml.VirtualToPhys = make(map[goabm.AgentID]goabm.AgentID)
	*/
	ml.Base.Init(arg)

	for i, agent := range *ml.Base.GetAgents() {
		a := agent.(*EchoChamberAgent)

		a.OfflineChangeCounter = 0
		a.OnlineChangeCounter = 0
		a.VAgent.ID = a.PAgent.(*goabm.FLWMAgent).Seqnr +
			goabm.AgentID(ml.Base.(*goabm.FixedLandscapeWithMovement).NAgents)
		(*ml.Overlay.GetAgents())[i] = a
		ml.Overlay.(*goabm.NetworkLandscape).SetAgent(i, a.VAgent)

		//agenter.(*goabm.FLWMAgent).Seqnr + goabm.AgentID(m.Landscape.Base.(*goabm.FixedLandscapeWithMovement).NAgents)
	}
	/*	// create a lookup table, so that every agent has a node in each world
		for i, agent := range ml.Base.GetAgents() {
		a := agent.(*EchoChamberAgent)
		 newId := a.Seqnr + ml.Base.NAgents
		 PhysToVirtual[a.Seqnr] = newID
		 VirtualToPhys[newID] = a       .Seqnr
		}*/
}

func (ml *MultilevelLandscape) GetAgents() *[]goabm.Agenter {
	return ml.Base.GetAgents()
}

func (ml *MultilevelLandscape) GetAgentById(id goabm.AgentID) goabm.Agenter {
	return ml.Base.GetAgentById(id)
}

func (ml *MultilevelLandscape) Dump() goabm.NetworkDump {

	b := ml.Base.Dump()
	o := ml.Overlay.Dump()

	//fmt.Printf("add %d to %d nodes", len(b.Nodes),len(o.Nodes))
	// we have to convert the overlay nodes and set freenode=true
	for _, n := range o.Nodes {
		n.(*EchoChamberAgent).FreeNode = true
		b.Nodes = append(b.Nodes, n)
	}
	//b.Nodes = append(b.Nodes, o.Nodes...)
	//fmt.Printf("total %d nodes", len(b.Nodes))
	//b = append(b)
	//return []byte(string(ml.Base.Dump()) + "\n" + string(ml.Overlay.Dump()))
	return b
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
	Steplength       float64 `goabm:"hide"`
	PVeloc           float64 `goabm:"hide"`
}

func (m *EchoChamberModel) Init(l interface{}) {
	m.Landscape = l.(*MultilevelLandscape)
}

func (m *EchoChamberModel) CreateAgent(agenter interface{}) goabm.Agenter {

	agent := &EchoChamberAgent{PAgent: agenter.(goabm.FLWMAgenter), VAgent: goabm.GenericAgent{}}

	ol := m.Landscape.Overlay.(*goabm.NetworkLandscape)
	ol.UserAgents = append(ol.UserAgents, agent)
	ol.Agents = append(ol.Agents, agent.VAgent)

	f := make(Feature, m.Features)
	for i := range f {
		f[i] = rand.Intn(m.Traits)
	}
	agent.Features = f
	agent.Model = m
	agent.FreeNode = false // physicalLandscape
	return agent
}

func (m *EchoChamberModel) LandscapeAction() {
	m.PhysicalCultures = m.CountCultures(m.Landscape.Base)
	m.VirtualCultures = m.CountCultures(m.Landscape.Overlay)
}

func (m *EchoChamberModel) CountCultures(ls goabm.Landscaper) int {
	cultures := make(map[string]int)
	for _, b := range *ls.GetAgents() {
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

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func omain() {
	goabm.Init()

	var traits = flag.Int("traits", 25, "number of cultural traits per feature")
	var features = flag.Int("features", 15, "number of cultural features")
	var size = flag.Int("size", 10, "size (width/height) of the landscape")

	var probveloc = flag.Float64("pveloc", 0.15, "probability that an agent moves")
	var steplength = flag.Float64("steplength", 0.2, "maximal distance a agent can travel per step")
	var sight = flag.Float64("sight", 1, "radius in which agent can interact")

	var FollowedBlogs = flag.Int("blogs", 4, "number of blogs to follow in the virtual world")
	var POnline = flag.Float64("p-online", 0.5, "probability of being online")
	var PLooking = flag.Float64("p-looking", 0.2, "probability of looking for new blogs (more similar and ditching old ones)")

	var runs = flag.Int("runs", 300, "number of simulation runs")
	var numAgents = flag.Int("agents", 30, "number of agents to simulate")

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

	model := &EchoChamberModel{Traits: *traits, Features: *features, PVeloc: *probveloc, Steplength: *steplength,
		FollowedBlogs: *FollowedBlogs, POnline: *POnline, PLookingForBlogs: *PLooking}
	physicalWorld := &goabm.FixedLandscapeWithMovement{Size: *size, NAgents: *numAgents, Sight: *sight}
	virtualWorld := &goabm.NetworkLandscape{}

	combinedLandscape := &MultilevelLandscape{Base: physicalWorld, Overlay: virtualWorld}

	sim := &goabm.Simulation{Landscape: combinedLandscape, Model: model, Log: goabm.Logger{StdOut: false}}
	fmt.Println("ABM simulation")
	sim.Init()

	var diffScore = 0

	for i := 0; i < *runs; i++ {

		diffScore += Abs(model.PhysicalCultures - model.VirtualCultures)
		if model.PhysicalCultures == 1 || model.VirtualCultures == 1 {
			sim.Stop()
			fmt.Printf("Stimulation prematurely done\n")
			break
		}
		sim.Step()

	}
	sim.Stop()

	d := model.Landscape.Dump()

	var avgOffline, avgOnline float64
	var sOnline, sOffline uint

	for _, a := range d.Nodes {
		v := a.(*EchoChamberAgent)
		sOnline += v.OnlineChangeCounter
		sOffline += v.OfflineChangeCounter
		//fmt.Printf("Node: %d : %d - %d\n",i,v.OnlineChangeCounter,v.OfflineChangeCounter)
	}
	avgOffline = float64(sOffline / uint(len(d.Nodes)))
	avgOnline = float64(sOnline / uint(len(d.Nodes)))
	fmt.Printf("\nAverage Online: %f Offline: %f\n", avgOnline, avgOffline)

	fmt.Printf("Total diff: %d\n", diffScore)
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
		return
	}
	//fmt.Printf("%v\n",sim.Landscape.GetAgents())
}

func main() {
	features := 15
	size := 10
	numAgents := 30
	runs := 200
	FollowedBlogs := 4
	probveloc := 0.15
	steplength := 0.2
	sight := 1.0
	POnline := 0.5
	PLooking := 0.2
	goabm.Init()
	//fmt.Printf("#Parameter search...\n");
	Min1 := 10
	Max1 := 20

	Min2 := 10
	Max2 := 20

	fmt.Printf("traits,f, cdiff,onc,offc,avgon,avgoff\n")
	for i := Min1; i < Max1; i++ {
		for j := Min2; j < Max2; j++ {

			r := simRun(i, features, size, numAgents, runs, FollowedBlogs, probveloc, steplength, sight, POnline, PLooking)

			fmt.Printf("%d, %d, %d, %d, %d, %f, %f\n", i, j, r.CultureDiff, r.OnlineCultures, r.OfflineCultures, r.AvgOnline, r.AvgOffline)
		}
	}

}

type SimRes struct {
	AvgOnline       float64
	AvgOffline      float64
	CultureDiff     int
	OnlineCultures  int
	OfflineCultures int
}

func simRun(traits, features, size, numAgents, runs, FollowedBlogs int, probveloc, steplength, sight, POnline, PLooking float64) SimRes {

	model := &EchoChamberModel{Traits: traits, Features: features, PVeloc: probveloc, Steplength: steplength,
		FollowedBlogs: FollowedBlogs, POnline: POnline, PLookingForBlogs: PLooking}
	physicalWorld := &goabm.FixedLandscapeWithMovement{Size: size, NAgents: numAgents, Sight: sight}
	virtualWorld := &goabm.NetworkLandscape{}

	combinedLandscape := &MultilevelLandscape{Base: physicalWorld, Overlay: virtualWorld}

	sim := &goabm.Simulation{Landscape: combinedLandscape, Model: model, Log: goabm.Logger{StdOut: false}}
	//fmt.Println("ABM simulation")
	sim.Init()

	var diffScore = 0

	for i := 0; i < runs; i++ {

		diffScore += Abs(model.PhysicalCultures - model.VirtualCultures)
		if model.PhysicalCultures == 1 || model.VirtualCultures == 1 {
			sim.Stop()
			//fmt.Printf("Stimulation prematurely done\n")
			break
		}
		sim.Step()

	}
	sim.Stop()

	d := model.Landscape.Dump()

	var avgOffline, avgOnline float64
	var sOnline, sOffline uint

	for _, a := range d.Nodes {
		v := a.(*EchoChamberAgent)
		sOnline += v.OnlineChangeCounter
		sOffline += v.OfflineChangeCounter
		//fmt.Printf("Node: %d : %d - %d\n",i,v.OnlineChangeCounter,v.OfflineChangeCounter)
	}
	avgOffline = float64(sOffline / uint(len(d.Nodes)))
	avgOnline = float64(sOnline / uint(len(d.Nodes)))
	//fmt.Printf("\nAverage Online: %f Offline: %f\n",avgOnline,avgOffline);

	//fmt.Printf("Total diff: %d\n",diffScore);

	res := SimRes{AvgOffline: avgOffline, AvgOnline: avgOnline, OnlineCultures: model.VirtualCultures, OfflineCultures: model.PhysicalCultures, CultureDiff: diffScore}
	return res

}
