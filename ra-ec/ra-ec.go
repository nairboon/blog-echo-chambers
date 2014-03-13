package main

import "math"
import "goabm"
import "math/rand"
import "fmt"

type Opinion float64

type Blog struct {
	Id     int
	Writer *EchoChamberAgent

	// extended boundries over all opinions ever held by the writer
	TopicMin Opinion
	TopicMax Opinion

	Subscribers map[goabm.AgentID]*EchoChamberAgent //agentid
}

type EchoChamberAgent struct {
	OpinionHistory []Opinion

	Uncertainty float64

	POnline   float64
	NComments int

	Writer bool
	Blog   *Blog

	// goabm related
	//goabm.Agenter `json:"Agent"`
	*goabm.GenericAgent
	Model *EchoChamberModel `json:"-"`
}

func (a *EchoChamberAgent) UpdateBlogBoundaries() {

	if a.Opinion() > a.Blog.TopicMax {
		a.Blog.TopicMax = a.Opinion()
	}

	if a.Opinion() < a.Blog.TopicMin {
		a.Blog.TopicMin = a.Opinion()
	}
}

func (a *EchoChamberAgent) Opinion() Opinion {
	return a.OpinionHistory[len(a.OpinionHistory)-1]
}

func (a *EchoChamberAgent) AddOpinion(no float64) {
	a.OpinionHistory = append(a.OpinionHistory, Opinion(no))
}

func (a *EchoChamberAgent) Act() {

	if a.Model.RollDice(a.POnline) {
		// agent is only considering the blogosphere

		a.Model.UpdateBlogSubscriptions(a)

		ok, blog := a.Model.RandomBlog(a)
		if !ok { // someĥow there is no blog...
			a.AddOpinion(float64(a.Opinion()))
			return
		}

		// interact with the writer
		writer := blog.Writer
		a.InteractWithAgent(writer)

		// read at most 10 "comments"
		nc := rand.Intn(int(math.Min(float64(len(blog.Subscribers)), float64(a.NComments))))
		c := 0
		for i, _ := range blog.Subscribers { // range is supposed to be random
			a.InteractWithAgent(blog.Subscribers[i])
			if c > nc {
				break
			}
			c++
		}

	} else {
		// we interact with random agnets

		other := a.Model.Landscape.RandomAgent().(*EchoChamberAgent)

		a.InteractWithAgent(other)
	}

}

func (a *EchoChamberAgent) AgreesWith(other *EchoChamberAgent) bool {

	Xi := float64(a.Opinion())
	Xj := float64(other.Opinion())

	Ui := a.Uncertainty
	Uj := other.Uncertainty

	Hij := math.Min(Xi+Ui, Xj+Uj) - math.Max(Xi-Ui, Xj-Uj)

	if Hij > Ui {
		return true
	} else {
		return false
	}

}

func (a *EchoChamberAgent) InteractWithAgent(other *EchoChamberAgent) {

	// dont interact with ourself
	if a == other {

		a.AddOpinion(float64(a.Opinion()))
		return
	}
	/* relative agreement model
	modified after: Michael Meadows and Dave Cliff (2012)
		"Reexamining the Relative Agreement Model of Opinion Dynamics"
	*/

	Xi := float64(a.Opinion())
	Xj := float64(other.Opinion())

	Ui := a.Uncertainty
	Uj := other.Uncertainty

	Hji := math.Min(Xj+Uj, Xi+Ui) - math.Max(Xj-Uj, Xi-Ui)
	Hij := math.Min(Xi+Ui, Xj+Uj) - math.Max(Xi-Ui, Xj-Uj)

	RAji := (Hji / Uj) - 1.0
	RAij := (Hij / Ui) - 1.0

	// Update
	if Hji > Uj {
		a.AddOpinion(Xi + (a.Model.MU * RAji * (Xj - Xi)))
		a.Uncertainty = Ui + (a.Model.MU * RAji * (Uj - Ui))

		// topic min/max
		if a.Writer {
			a.UpdateBlogBoundaries()
		}
	} else {
		a.AddOpinion(Xi) // For history.
	}

	if Hij > Ui {
		other.AddOpinion(Xj + (a.Model.MU * RAij * (Xi - Xj)))
		other.Uncertainty = Uj + (a.Model.MU * RAij * (Ui - Uj))

		if a.Writer {
			a.UpdateBlogBoundaries()
		}
	} else {
		other.AddOpinion(Xj) // For history.
	}
}

type EchoChamberModel struct {
	MU          float64
	Uncertainty float64
	POnline     float64
	NBlogs      int
	NComments   int

	ECRatio float64

	//datastructures
	Landscape goabm.Landscaper
	Blogs     []Blog
	goabm.Model

	_blog_counter int
}

func (e *EchoChamberModel) Init(l interface{}) {
	e.Landscape = l.(goabm.Landscaper)
	e.Blogs = make([]Blog, e.NBlogs)
}

func (a *EchoChamberModel) LandscapeAction() {
	// measure overall agreement/disagreement

	tagree := 0
	tdisagree := 0

	ec := 0

	tspread := 0.0
	for _, blog := range a.Blogs {

		tspread += float64(blog.TopicMax - blog.TopicMin)
		agree := 0
		disagree := 0
		author := blog.Writer
		// every subscriber
		for _, agent := range blog.Subscribers {
			if agent.AgreesWith(author) {
				agree++
			} else {
				disagree++
			}
		}
		ratio := float64(agree) / float64(agree+disagree) // agreement in %
		if ratio > 0.64 {
			ec++
		}
		tagree += agree
		tdisagree += disagree
		//fmt.Printf("agreement: %f %d %d\n", ratio, agree, disagree)
	}

	ratio := float64(tagree) / float64(tagree+tdisagree) // agreement in %
	a.ECRatio = ratio

	//avgtspread := float64(tspread) / float64(len(a.Blogs))
	//fmt.Printf("avts: %f\n", avgtspread)
	//fmt.Printf("Total agreement: %f %d %d\n", ratio, tagree, tdisagree)
}

func (a *EchoChamberModel) RandomBlog(agent *EchoChamberAgent) (bool, *Blog) {
	for i, blog := range a.Blogs {

		_, ok := blog.Subscribers[agent.ID()]
		if ok {
			return true, &a.Blogs[i]
		}
	}
	return false, nil
}

func (a *EchoChamberModel) UpdateBlogSubscriptions(agent *EchoChamberAgent) {

	//fmt.Printf("update subs %d\n", len(a.Blogs))
	// check for each blog, if we agent.optinion > blog.min & < blog.max, subscribe or unsubscribe
	for i, blog := range a.Blogs {

		_, ok := blog.Subscribers[agent.ID()]
		// we should subscribe to this one
		if agent.Opinion() >= (blog.TopicMin-0.0) && agent.Opinion() <= blog.TopicMax {
			if !ok { // we're not yet subscribed
				a.Blogs[i].Subscribers[agent.ID()] = agent
			}

		} else { // unsubscribe
			if ok { // we are subscribed
				delete(a.Blogs[i].Subscribers, agent.ID())
			}
		}
	}
}

func Random(min, max float64) float64 {
	return rand.Float64()*(max-min) + min
}

func (a *EchoChamberModel) CreateAgent(agenter interface{}) goabm.Agenter {

	agent := &EchoChamberAgent{GenericAgent: agenter.(*goabm.GenericAgent)}

	agent.OpinionHistory = make([]Opinion, 0, 100)
	agent.AddOpinion(Random(-1.0, 1.0))

	agent.Uncertainty = a.Uncertainty
	agent.POnline = a.POnline
	agent.Model = a
	agent.NComments = a.NComments

	if a._blog_counter < a.NBlogs {
		agent.Blog = &a.Blogs[a._blog_counter]
		agent.Writer = true
		agent.Blog.Writer = agent
		agent.Blog.Id = a._blog_counter
		agent.Blog.TopicMax = agent.Opinion()
		agent.Blog.TopicMin = agent.Opinion()

		agent.Blog.Subscribers = make(map[goabm.AgentID]*EchoChamberAgent)
		a._blog_counter++
	}
	return agent
}

func simRun(MU, Uncertainty, POnline float64, N, runs, blogs, NComments int) float64 {

	model := &EchoChamberModel{
		MU:          MU,
		Uncertainty: Uncertainty,
		NBlogs:      blogs,
		POnline:     POnline,
		NComments:   NComments}

	sim := &goabm.Simulation{Landscape: &goabm.NetworkLandscape{
		Size: N},
		Model: model, Log: goabm.Logger{StdOut: false}}
	sim.Init()

	for i := 0; i < runs; i++ {

		sim.Step()

	}
	sim.Stop()

	//agents := model.Landscape.GetAgents()
	for i := 0; i < runs; i++ {

		for _, b := range *model.Landscape.GetAgents() {
			agent := b.(*EchoChamberAgent)
			if len(agent.OpinionHistory) < runs {
				fmt.Printf("%d instead of %d\n", len(agent.OpinionHistory), runs)
				panic("agent has less")
			}
			if i == 0 { // header
				if agent.Writer {
					//fmt.Printf("b")
				}
				//fmt.Printf("agent%d,\t", j)
			} else {

				if agent.Writer {
					//fmt.Printf("t(%f,%f)", agent.Blog.TopicMin, agent.Blog.TopicMax)
				}
				//fmt.Printf("%f,\t", agent.OpinionHistory[i])

			}
		}
		//fmt.Printf("\n")
	}

	//fmt.Printf("EC: %f\n", model.ECRatio)

	return model.ECRatio
}

func main() {
	goabm.Init()

	ponline := 0.5

	agents := 200
	runs := 400
	blogs := 10

	samplestep := 0.15

	fmt.Printf("mu, ponline, deltares\n")
	for mu := 1.6; mu < 3.0; mu += samplestep {
		for po := 1; po < 20; po += 1 {

			r := simRun(mu, 0.3, ponline, agents, runs, blogs, po)
			/*
				d := math.Abs(r-0.64) * 10

				e := d * d
				if r < 0.64 {
					e *= -1.0
				}*/
			fmt.Printf("%f, %d, %f\n", mu, po, r)
		}
	}
}
