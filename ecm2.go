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

type FloatRange [2]float64
type IntRange [2]int

type Comment struct {
	Message   Feature
	Responses []Feature
}

type Blog struct {
	Posts     []Comment
	Followers []goabm.AgentID
	ID        int
}

func (b *Blog) Publish(f Feature) {
	b.Posts = append(b.Posts, Comment{Message: f})
}

type BlogSubscription struct {
	FollowedBlogs []*Blog
	ReadPosts     map[int]map[int]bool //blogid -> postid
}

func (bs *BlogSubscription) Subscribe(b *Blog) {
	bs.FollowedBlogs = append(bs.FollowedBlogs, b)
	bs.ReadPosts[b.ID] = make(map[int]bool, len(b.Posts))
}

func (bs *BlogSubscription) UnreadBlogPost() *Comment {
	//possibleReads := make([]*Comment, 1)

	//fmt.Printf("i follow: %d %v %v", len(bs.FollowedBlogs),
	//	bs.FollowedBlogs, bs.ReadPosts)
	var PostToRead *Comment
	// foreach blog
	for _, blog := range bs.FollowedBlogs {
		// have we read it all?
		if val, ok := bs.ReadPosts[blog.ID]; ok && len(blog.Posts) == len(val) {
			//skip
			continue
		}

		for j, post := range blog.Posts {

			// is read?
			//val :=
			if bs.ReadPosts[blog.ID][j] {
				continue
			}

			// mark as read
			//fmt.Printf("read P %d %v", j, post)
			bs.ReadPosts[blog.ID][j] = true
			PostToRead = &post

		}

	}

	//pickPost := rand.Intn(len(possibleReads))
	//post :=
	return PostToRead
}

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
	PStartBlogging          float64    `goabm:"hide"`
	PWriteBlogPost          float64    `goabm:"hide"`
	RSubscribedBlogs        IntRange   `goabm:"hide"`
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
	dice := rand.Float64()
	//interact with sim% chance
	if dice <= sim {
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

func (a *EchoChamberAgent) InteractWithComment(c Feature) {

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
	if len(post.Responses) == 0 {
		return
	}
	numResponses := rand.Intn(len(post.Responses))
	for i := 0; i < numResponses; i++ {
		// and interact with them
		a.InteractWithComment(post.Responses[i])
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
	// check if we have agent around
	other := a.GetRandomNeighbor()
	if other != nil {
		a.PhysicalInteraction(other.(*EchoChamberAgent))
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

	TotalBlogPosts int
	TotalBlogs     int

	Landscape goabm.Landscaper

	//parameters
	NTraits   int `goabm:"hide"` // don't show these in the stats'
	NFeatures int `goabm:"hide"`

	POnline float64 `goabm:"hide"`

	// blogging parameters
	PStartBlogging          float64    `goabm:"hide"`
	PWriteBlogPost          float64    `goabm:"hide"`
	RSubscribedBlogs        IntRange   `goabm:"hide"`
	RSimilarityConfortLevel FloatRange `goabm:"hide"`

	Steplength float64 `goabm:"hide"`
	PVeloc     float64 `goabm:"hide"`

	Blogger map[goabm.AgentID]*Blog `goabm:"hide"`
}

// helper function to determine the similarity between to features
func (e *EchoChamberModel) Similarity(first, other Feature) float64 {
	c := float64(0.0)
	// count equal traits, final score = shared traits/total traits
	for i := range first {
		if first[i] == other[i] {
			c = c + 1
		}
	}
	//fmt.Printf("sim: %f/%d\n",c,len(a.features))
	return c / float64(len(first))
}

func (e *EchoChamberModel) CreateBlog(a *EchoChamberAgent) *Blog {
	//fmt.Println("created blog")
	e.Blogger[a.ID()] = &Blog{ID: len(e.Blogger)}
	return e.Blogger[a.ID()]
}

func (e *EchoChamberModel) GoogleBlog(f Feature) *Blog {
	// we google a blog for our cultural identity (features)
	// rank all blogs for similarity (last post), return the best

	ranking := make(map[goabm.AgentID]float64, len(e.Blogger))

	for i, blog := range e.Blogger {

		lastPost := blog.Posts[len(blog.Posts)-1]
		sim := e.Similarity(f, lastPost.Message)
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

	agent.MySubscriptions.ReadPosts = make(map[int]map[int]bool)
	agent.MySubscriptions.FollowedBlogs = make([]*Blog, 0)
	agent.Model = a
	return agent
}

func (a *EchoChamberModel) LandscapeAction() {
	a.Cultures = a.CountCultures()

	a.TotalBlogs = len(a.Blogger)
	posts := 0
	for _, blog := range a.Blogger {
		posts += len(blog.Posts)
	}

	if posts > a.TotalBlogPosts {
		//panic("ajlkajdas")
		a.TotalBlogPosts = posts

	}

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
	AvgOnline       float64
	AvgOffline      float64
	CultureDiff     int
	OnlineCultures  int
	OfflineCultures int
}

func simRun(traits, features, size, numAgents, runs int,
	probveloc, steplength, sight, POnline, PLooking, PStartBlogging, PWriteBlogPost float64,
	RSubscribedBlogs IntRange) SimRes {

	model := &EchoChamberModel{
		NTraits:          traits,
		NFeatures:        features,
		PVeloc:           probveloc,
		Steplength:       steplength,
		POnline:          POnline,
		PStartBlogging:   PStartBlogging,
		PWriteBlogPost:   PWriteBlogPost,
		RSubscribedBlogs: RSubscribedBlogs}

	sim := &goabm.Simulation{Landscape: &goabm.FixedLandscapeWithMovement{Size: size, NAgents: numAgents, Sight: sight},
		Model: model, Log: goabm.Logger{StdOut: true}}
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

	features := 5
	size := 10
	numAgents := 5
	runs := 50

	probveloc := 0.15

	steplength := 1.5
	sight := 1.0

	POnline := 0.5
	PLooking := 0.2
	traits := 5

	PStartBlogging := 0.1
	PWriteBlogPost := 0.2
	RSubscribedBlogs := IntRange{1, 5}

	//fmt.Printf("#Parameter search...\n");

	r := simRun(traits, features, size, numAgents, runs,
		probveloc, steplength, sight, POnline, PLooking,
		PStartBlogging, PWriteBlogPost, RSubscribedBlogs)

	fmt.Printf("%d, %d, %d, %d, %d, %f, %f\n", traits, features, r.CultureDiff, r.OnlineCultures, r.OfflineCultures, r.AvgOnline, r.AvgOffline)
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
