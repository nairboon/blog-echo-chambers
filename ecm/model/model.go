package model

import "goabm"

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

