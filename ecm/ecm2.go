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
import "math"

import . "flache/ecm/model"
//import "time"
import "runtime"
import "github.com/GaryBoone/GoStats/stats"


// Implementation of the Agent, cultural Traits are stored in features
type EchoChamberAgent struct {
	OnlineInteraction  int
	OfflineInteraction int

	NInteractionF              float64
	NDaysSinceLastInteractionF float64
	NNeedToInteractF           float64
	SimilarityF                float64
	NMaxRelations              uint

	// blogging parameters
	PStartBlogging   float64 `goabm:"hide"`
	PWriteBlogPost   float64 `goabm:"hide"`
	PRespondBlogPost float64 `goabm:"hide"`
	POnline          float64 `goabm:"hide"`

	RSubscribedBlogs IntRange `goabm:"hide"`

	// if blog similarity < minConfort || > maxConfort, unsubscribe
	RSimilarityConfortLevel FloatRange `goabm:"hide"`

	// movement related
	Steplength float64 `goabm:"hide"`
	PVeloc     float64 `goabm:"hide"`

	// data structures
	MyBlog          *Blog
	MySubscriptions BlogSubscription
	Features        Feature

	// goabm related
	*goabm.FLWMAgent `json:"Agent"`
	Model            *EchoChamberModel `json:"-"`
}

// returns the culture as a string
func (a *EchoChamberAgent) Culture() string {
	return fmt.Sprintf("%v", a.Features)
}

func (a *EchoChamberAgent) MutateFeatures() {
        i := rand.Intn(len(a.Features))
        j := rand.Intn(a.Model.NTraits)
        
        a.Features[i] = j
}

func (a *EchoChamberAgent) ChangeFeatures(other Feature) {
	for i := range a.Features {
		if a.Features[i] != other[i] {
			//fmt.Printf("%d influenced %d\n", other.seqnr, a.seqnr)
			a.Features[i] = other[i]
			/*if OtherIsOnline {
			a.OnlineChangeCounter++;
			} else {
			a.OfflineChangeCounter++;
			}*/
			return
		}
	}
}

// returns true if  a change happend
func (a *EchoChamberAgent) FeatureInteraction(other Feature) bool {

	sim := a.Similarity(other)

	//fmt.Printf("sim: %f", sim)
	if sim >= 0.99 {
		// agents are already equal
		return false
	}

	//interact with sim% chance
	if goabm.RollDice(sim) {
	        if a.Model.IsRuleActive("transmission_error") {
	        // the feature will be changed randomly, with the inverse similarity probability
	         np:= 1.0 - sim
	         if goabm.RollDice(np) {
	       // random feature change
	                a.MutateFeatures()
	                return true
	         }
	        }
		a.ChangeFeatures(other)
		return true
	}
	return false
}

func (a *EchoChamberAgent) AgentInteraction(other *EchoChamberAgent) bool {
	return a.FeatureInteraction(other.Features)
}

func (a *EchoChamberAgent) PhysicalInteraction(other *EchoChamberAgent) {

	//fmt.Printf("before: %s", a.Culture())
	// plain simple interaction
	change := a.AgentInteraction(other)
	if change {
		a.OfflineInteraction++
	}
	//fmt.Printf("after: %s", a.Culture())

}

func (a *EchoChamberAgent) FindABlog() {
	blog := a.Model.GoogleBlog(a.Features)
	if blog == nil {
		//panic("nil blog")
		//fmt.Println("no blog found...")
		return
	}
	a.MySubscriptions.Subscribe(blog)
}

func (a *EchoChamberAgent) WriteBlog() {
	// blogging consists of publishing the current agents cultural identiy (features)
	if a.MyBlog == nil {
		panic("can't blog without a blog")
	}

	a.MyBlog.Publish(a.Features)
}

func (a *EchoChamberAgent) ReadBlogs() {
	// are we subscribed to any blgos?
	numBlogs := len(a.MySubscriptions.FollowedBlogs)
	if numBlogs == 0 {
		//no! lets find some
		a.FindABlog()
		return
	}

	//check if we should find some more blogs, to between minBlog < numBlog < maxBlog
	if numBlogs < a.RSubscribedBlogs[0] {
		a.FindABlog()
	} else if numBlogs < a.RSubscribedBlogs[1] {
		// we have still space more more blogs, add one with p=0.1
		if goabm.RollDice(0.1) {
			a.FindABlog()

		}
	}

	// check if we like our blogs
	if goabm.RollDice(0.4) {
		a.MySubscriptions.Remove(a.RSimilarityConfortLevel, a.Features)
	}

	if len(a.MySubscriptions.FollowedBlogs) == 0 {
		// still no blogs available?
		return
	}
	// so we're subscribed to a bunch of blogs, let's pick a new post and read it
	post := a.MySubscriptions.UnreadBlogPost()

	if post == nil {
		//we have read all posts!! move on...
		return
	}

	//the post consists of a Feature and some responses
	change := a.FeatureInteraction(post.Message)

	if change {
		a.OnlineInteraction++
	}
	// now we read some responses, if there are any
	if len(post.Responses) > 0 {

		numResponses := rand.Intn(len(post.Responses))
		for i := 0; i < numResponses; i++ {
			// and interact with them
			comment := post.Responses[i]
			//read it
			change := a.FeatureInteraction(comment)

			if change {
				a.OnlineInteraction++
			}

		}
	}

	//fmt.Printf("r %f\n")

	if goabm.RollDice(a.PRespondBlogPost) {
		// write comment
		post.Respond(a.Features)

	}
}

func (a *EchoChamberAgent) VirtualInteraction() {

	//are we blogging?
	if a.MyBlog == nil {
		// no we're not

		//let's consider starting a blog
		if goabm.RollDice(a.PStartBlogging) {
			// setup blog
			a.MyBlog = a.Model.CreateBlog(a)

			// first post!
			a.WriteBlog()

			// enough for today
			return
		}

	} else {
		// we have a blog
		if goabm.RollDice(a.PWriteBlogPost) {
			// we blog
			a.WriteBlog()
			return
		}
	}

	// otherwiese read some blogs
	a.ReadBlogs()

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
	/*// check if we have agent around
	other := a.GetRandomNeighbor()
	if other != nil {
		a.PhysicalInteraction(other.(*EchoChamberAgent))
	} else {
		// go online otherwise
		a.VirtualInteraction()
	}*/

	if goabm.RollDice(a.POnline) {
		a.VirtualInteraction()
	} else {
		other := a.GetRandomNeighbor()
		if other != nil {
			a.PhysicalInteraction(other.(*EchoChamberAgent))
		}
	}

}

// helper function to determine the similarity between to agents
func (a *EchoChamberAgent) Similarity(other Feature) float64 {
	c := float64(0.0)
	// count equal traits, final score = shared traits/total traits
	for i := range a.Features {
		if a.Features[i] == other[i] {
			c = c + 1
		}
	}
	//fmt.Printf("sim: %f/%d\n",c,len(a.features))
	return c / float64(len(a.Features))
}

type EchoChamberModel struct {

	// stats
	Cultures           int
	OnlineInteraction  int
	OfflineInteraction int
	TotalComments      int
	TotalBlogPosts     int
	TotalBlogs         int
	TotalEchoChambers  int
	EchoChamberRatio   float64

	//parameters
	NTraits   int `goabm:"hide"` // don't show these in the stats'
	NFeatures int `goabm:"hide"`

	// blogging parameters
	PStartBlogging          float64    `goabm:"hide"`
	PWriteBlogPost          float64    `goabm:"hide"`
	RSubscribedBlogs        IntRange   `goabm:"hide"`
	RSimilarityConfortLevel FloatRange `goabm:"hide"`
	PRespondBlogPost        float64    `goabm:"hide"`

	POnline float64 `goabm:"hide"`

	//movement parameters
	Steplength float64 `goabm:"hide"`
	PVeloc     float64 `goabm:"hide"`

	//datastructures
	Blogger   map[goabm.AgentID]*Blog `goabm:"hide"`
	Landscape goabm.Landscaper

	goabm.Model
}



func (e *EchoChamberModel) CreateBlog(a *EchoChamberAgent) *Blog {
	//e.Blogger = append(e.Blogger, )
	e.Blogger[a.ID()] = &Blog{ID: len(e.Blogger)}
	//fmt.Printf("%d created blog %d\n", a.ID(), len(e.Blogger))

	return e.Blogger[a.ID()]
}

func (e *EchoChamberModel) GoogleBlog(f Feature) *Blog {
	// we google a blog for our cultural identity (features)
	// rank all blogs for similarity (last post), return the best

	ranking := make(map[goabm.AgentID]float64, len(e.Blogger))

	for i, blog := range e.Blogger {

		lastPost := blog.Posts[len(blog.Posts)-1]
		sim := Similarity(f, lastPost.Message)
		ranking[i] = sim
	}

	// find best match
	best := goabm.AgentID(0)
	for i, rank := range ranking {
		if rank > ranking[best] {
			best = i
		}
	}
	bestBlog := e.Blogger[best]
	return bestBlog
}

func (e *EchoChamberModel) Init(l interface{}) {
	e.Landscape = l.(goabm.Landscaper)

	e.Blogger = make(map[goabm.AgentID]*Blog)

	//e.Ruleset.Init()
}

func (a *EchoChamberModel) CreateAgent(agenter interface{}) goabm.Agenter {

	agent := &EchoChamberAgent{FLWMAgent: agenter.(*goabm.FLWMAgent)}

	f := make(Feature, a.NFeatures)
	for i := range f {
		f[i] = rand.Intn(a.NTraits)
	}
	agent.Features = f

	agent.PStartBlogging = a.PStartBlogging
	agent.PVeloc = a.PVeloc
	agent.PWriteBlogPost = a.PWriteBlogPost
	agent.RSubscribedBlogs = a.RSubscribedBlogs
	agent.PRespondBlogPost = a.PRespondBlogPost
	agent.POnline = a.POnline
	agent.RSimilarityConfortLevel = a.RSimilarityConfortLevel

	agent.MySubscriptions.ReadPosts = make(map[int]map[int]bool)
	agent.MySubscriptions.FollowedBlogs = make(map[int]*Blog)
	agent.Model = a
	return agent
}

func (a *EchoChamberModel) BlogStatistics() {

	a.TotalBlogs = len(a.Blogger)
	posts := 0
	comments := 0
	ec := 0
	for _, blog := range a.Blogger {
		posts += len(blog.Posts)

		approve := 0
		totalresp := 0
		for _, p := range blog.Posts {
			comments += len(p.Responses)
			// calculate the similarity of each comment to the blog

			for _, c := range p.Responses {
				sim := Similarity(p.Message, c)
				if sim > 0.50 {
					approve++
				}
				totalresp++
			}

		}

		if totalresp > 0 {
			//avg := avgt / float64(len(p.Responses))
			//fmt.Printf("avg: %f %f %d\n", avg, avgt, len(p.Responses))
			approval := float64(approve / totalresp)
			if approval > 0.64 {
				// we found an echo chamber
				//fmt.Printf("approval: %f", approval)
				ec++
			}
		}
	}

	a.TotalEchoChambers = ec

	if a.TotalBlogs > 0 {
		a.EchoChamberRatio = float64(a.TotalEchoChambers) / float64(a.TotalBlogs)
		//fmt.Printf("alkda %d   %d  %f\n", a.TotalEchoChambers, a.TotalBlogs, a.EchoChamberRatio)
	}
	if comments > a.TotalComments {
		a.TotalComments = comments
	}

	if posts > a.TotalBlogPosts {
		//panic("ajlkajdas")
		a.TotalBlogPosts = posts
	}

	// analyze follower constitution
	for _, blog := range a.Blogger {
		posts += len(blog.Posts)
	}
}

func (a *EchoChamberModel) LandscapeAction() {
	a.BlogStatistics()

	a.Cultures = a.CountCultures()

	for _, b := range *a.Landscape.GetAgents() {
		eca := b.(*EchoChamberAgent)

		a.OfflineInteraction += eca.OfflineInteraction
		a.OnlineInteraction += eca.OnlineInteraction
	}

}

func (a *EchoChamberModel) CountCultures() int {
	cultures := make(map[string]int)
	for _, b := range *a.Landscape.GetAgents() {
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
				resc,p.Rules)
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
	rules.SetRule("movement", true)
	rules.SetRule("transmission_error", false)

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
