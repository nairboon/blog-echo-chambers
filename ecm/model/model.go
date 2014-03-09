package model

import "goabm"

import "fmt"
import "math/rand"



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


type PF func() float64

type Feature []int

type Comment struct {
	Message   Feature
	Responses []Feature
}

func (c *Comment) Respond(f Feature) {
	c.Responses = append(c.Responses, f)
}

type FloatRange [2]float64
type IntRange [2]int

type Blog struct {
	Posts     []Comment
	Followers []goabm.AgentID
	ID        int
}

func (b *Blog) Publish(f Feature) {
	b.Posts = append(b.Posts, Comment{Message: f})
}

type BlogSubscription struct {
	FollowedBlogs map[int]*Blog
	ReadPosts     map[int]map[int]bool //blogid -> postid
}

func (bs *BlogSubscription) Subscribe(b *Blog) {
	bs.FollowedBlogs[len(bs.FollowedBlogs)] = b
	bs.ReadPosts[b.ID] = make(map[int]bool, len(b.Posts))
}

func (bs *BlogSubscription) Remove(cl FloatRange, subscriber Feature) {

	nb := make(map[int]*Blog)
	for _, blog := range bs.FollowedBlogs {
		// get last post
		p := blog.Posts[len(blog.Posts)-1]

		sim := Similarity(subscriber, p.Message)
		if sim > cl[0] {
			// keep it

			//nb = append(nb, blog)
			nb[len(nb)] = blog
		}
	}
	bs.FollowedBlogs = nb
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

		for j, _ := range blog.Posts {

			// is read?
			//val :=
			if bs.ReadPosts[blog.ID][j] {
				continue
			}

			// mark as read
			//fmt.Printf("read P %d %v", j, post)
			bs.ReadPosts[blog.ID][j] = true
			PostToRead = &blog.Posts[j]

		}

	}

	//pickPost := rand.Intn(len(possibleReads))
	//post :=
	return PostToRead
}

// helper function to determine the similarity between to features
func Similarity(first, other Feature) float64 {
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


	PUnderstanding   float64 `goabm:"hide"`
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
			
			if goabm.RollDice(a.PUnderstanding) {
			// we understood the agent
				a.Features[i] = other[i]
			} else {
			        // we didn't, but we still got influeced
			     	j := rand.Intn(a.Model.NTraits)
			     	a.Features[i] = j
			}

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
			np := 1.0 - sim
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

        // pdf
        PFAI PF
        PFOnline PF
        PFU PF

	// blogging parameters
	PStartBlogging          float64    `goabm:"hide"`

	RSubscribedBlogs        IntRange   `goabm:"hide"`
	RSimilarityConfortLevel FloatRange `goabm:"hide"`



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

	agent.RSubscribedBlogs = a.RSubscribedBlogs

	agent.RSimilarityConfortLevel = a.RSimilarityConfortLevel
	
	// pdfs
	agent.POnline = a.PFOnline()
	agent.PRespondBlogPost = a.PFAI()
	agent.PWriteBlogPost = agent.PRespondBlogPost
	
	agent.PUnderstanding = a.PFU()
	
	
	agent.MySubscriptions.ReadPosts = make(map[int]map[int]bool)
	agent.MySubscriptions.FollowedBlogs = make(map[int]*Blog)
	agent.Model = a
	//fmt.Printf("agent: %v\n",agent)
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
