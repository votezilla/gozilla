package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"io/ioutil"
	"net/url"
	"math/rand"
)

// A news source to request the news from.
type NewsSource struct {
	Name		string
	ImageUrl	string
	Id			string
}

// JSON-parsed format of an article.
type Article struct {
	Author		string
	Title		string
	Description	string
	Url			string
	UrlToImage	string
	PublishedAt	string
}

// JSON-parsed format of a news request.
type News struct {
	Status		string
	Source		string
	SortBy		string
	Articles	[]Article
}

// For rendering the news article information.
type ArticleArg struct {
	Article
	Index		int
	Host		string
}

var (
//	Nums = []int{6, 7, 8}
//	
//	Source = NewsSource{
//		"ABC News (AU)",
//		"https://icons.better-idea.org/icon?url=http://www.abc.net.au/news&size=70..120..200",
//		"abc-news-au"}
	
	NewsSources = []NewsSource{
		{"ABC News (AU)",
		"https://icons.better-idea.org/icon?url=http://www.abc.net.au/news&size=70..120..200",
		"abc-news-au"},
		{"Al Jazeera English",
		"https://icons.better-idea.org/icon?url=http://www.aljazeera.com&size=70..120..200",
		"al-jazeera-english"},
		{"Ars Technica",
		"https://icons.better-idea.org/icon?url=http://arstechnica.com&size=70..120..200",
		"ars-technica"},
		{"Associated Press",
		"https://icons.better-idea.org/icon?url=https://apnews.com/&size=70..120..200",
		"associated-press"},
		{"BBC News",
		"https://icons.better-idea.org/icon?url=http://www.bbc.co.uk/news&size=70..120..200",
		"bbc-news"},
		{"BBC Sport",
		"https://icons.better-idea.org/icon?url=http://www.bbc.co.uk/sport&size=70..120..200",
		"bbc-sport"},
		{"Bild",
		"https://icons.better-idea.org/icon?url=http://www.bild.de&size=70..120..200",
		"bild"},
		{"Bloomberg",
		"https://icons.better-idea.org/icon?url=http://www.bloomberg.com&size=70..120..200",
		"bloomberg"},
		{"Breitbart News",
		"https://icons.better-idea.org/icon?url=http://www.breitbart.com&size=70..120..200",
		"breitbart-news"},
		{"Business Insider",
		"https://icons.better-idea.org/icon?url=http://www.businessinsider.com&size=70..120..200",
		"business-insider"},
		{"Business Insider (UK)",
		"https://icons.better-idea.org/icon?url=http://uk.businessinsider.com&size=70..120..200",
		"business-insider-uk"},
		{"Buzzfeed",
		"https://icons.better-idea.org/icon?url=https://www.buzzfeed.com&size=70..120..200",
		"buzzfeed"},
		{"CNBC",
		"https://icons.better-idea.org/icon?url=http://www.cnbc.com&size=70..120..200",
		"cnbc"},
		{"CNN",
		"https://icons.better-idea.org/icon?url=http://us.cnn.com&size=70..120..200",
		"cnn"},
		{"Daily Mail",
		"https://icons.better-idea.org/icon?url=http://www.dailymail.co.uk/home/index.html&size=70..120..200",
		"daily-mail"},
		{"Der Tagesspiegel",
		"https://icons.better-idea.org/icon?url=http://www.tagesspiegel.de&size=70..120..200",
		"der-tagesspiegel"},
		{"Die Zeit",
		"https://icons.better-idea.org/icon?url=http://www.zeit.de/index&size=70..120..200",
		"die-zeit"},
		{"Engadget",
		"https://icons.better-idea.org/icon?url=https://www.engadget.com&size=70..120..200",
		"engadget"},
		{"Entertainment Weekly",
		"https://icons.better-idea.org/icon?url=http://www.ew.com&size=70..120..200",
		"entertainment-weekly"},
		{"ESPN",
		"https://icons.better-idea.org/icon?url=http://espn.go.com&size=70..120..200",
		"espn"},
		{"ESPN Cric Info",
		"https://icons.better-idea.org/icon?url=http://www.espncricinfo.com/&size=70..120..200",
		"espn-cric-info"},
		{"Financial Times",
		"https://icons.better-idea.org/icon?url=http://www.ft.com/home/uk&size=70..120..200",
		"financial-times"},
		{"Focus",
		"https://icons.better-idea.org/icon?url=http://www.focus.de&size=70..120..200",
		"focus"},
		{"Football Italia",
		"https://icons.better-idea.org/icon?url=http://www.football-italia.net&size=70..120..200",
		"football-italia"},
		{"Fortune",
		"https://icons.better-idea.org/icon?url=http://fortune.com&size=70..120..200",
		"fortune"},
		{"FourFourTwo",
		"https://icons.better-idea.org/icon?url=http://www.fourfourtwo.com/news&size=70..120..200",
		"four-four-two"},
		{"Fox Sports",
		"https://icons.better-idea.org/icon?url=http://www.foxsports.com&size=70..120..200",
		"fox-sports"},
		{"Google News",
		"https://icons.better-idea.org/icon?url=https://news.google.com&size=70..120..200",
		"google-news"},
		{"Gruenderszene",
		"https://icons.better-idea.org/icon?url=http://www.gruenderszene.de&size=70..120..200",
		"gruenderszene"},
		{"Hacker News",
		"https://icons.better-idea.org/icon?url=https://news.ycombinator.com&size=70..120..200",
		"hacker-news"},
		{"Handelsblatt",
		"https://icons.better-idea.org/icon?url=http://www.handelsblatt.com&size=70..120..200",
		"handelsblatt"},
		{"IGN",
		"https://icons.better-idea.org/icon?url=http://www.ign.com&size=70..120..200",
		"ign"},
		{"Independent",
		"https://icons.better-idea.org/icon?url=http://www.independent.co.uk&size=70..120..200",
		"independent"},
		{"Mashable",
		"https://icons.better-idea.org/icon?url=http://mashable.com&size=70..120..200",
		"mashable"},
		{"Metro",
		"https://icons.better-idea.org/icon?url=http://metro.co.uk&size=70..120..200",
		"metro"},
		{"Mirror",
		"https://icons.better-idea.org/icon?url=http://www.mirror.co.uk/&size=70..120..200",
		"mirror"},
		{"MTV News",
		"https://icons.better-idea.org/icon?url=http://www.mtv.com/news&size=70..120..200",
		"mtv-news"},
		{"MTV News (UK)",
		"https://icons.better-idea.org/icon?url=http://www.mtv.co.uk/news&size=70..120..200",
		"mtv-news-uk"},
		{"National Geographic",
		"https://icons.better-idea.org/icon?url=http://news.nationalgeographic.com&size=70..120..200",
		"national-geographic"},
		{"New Scientist",
		"https://icons.better-idea.org/icon?url=https://www.newscientist.com/section/news&size=70..120..200",
		"new-scientist"},
		{"Newsweek",
		"https://icons.better-idea.org/icon?url=http://www.newsweek.com&size=70..120..200",
		"newsweek"},
		{"New York Magazine",
		"https://icons.better-idea.org/icon?url=http://nymag.com&size=70..120..200",
		"new-york-magazine"},
		{"NFL News",
		"https://icons.better-idea.org/icon?url=http://www.nfl.com/news&size=70..120..200",
		"nfl-news"},
		{"Polygon",
		"https://icons.better-idea.org/icon?url=http://www.polygon.com&size=70..120..200",
		"polygon"},
		{"Recode",
		"https://icons.better-idea.org/icon?url=http://www.recode.net&size=70..120..200",
		"recode"},
		{"Reddit /r/all",
		"https://icons.better-idea.org/icon?url=https://www.reddit.com/r/all&size=70..120..200",
		"reddit-r-all"},
		{"Reuters",
		"https://icons.better-idea.org/icon?url=http://www.reuters.com&size=70..120..200",
		"reuters"},
		{"Spiegel Online",
		"https://icons.better-idea.org/icon?url=http://www.spiegel.de&size=70..120..200",
		"spiegel-online"},
		{"T3n",
		"https://icons.better-idea.org/icon?url=http://t3n.de&size=70..120..200",
		"t3n"},
		{"TalkSport",
		"https://icons.better-idea.org/icon?url=http://talksport.com&size=70..120..200",
		"talksport"},
		{"TechCrunch",
		"https://icons.better-idea.org/icon?url=https://techcrunch.com&size=70..120..200",
		"techcrunch"},
		{"TechRadar",
		"https://icons.better-idea.org/icon?url=http://www.techradar.com&size=70..120..200",
		"techradar"},
		{"The Economist",
		"https://icons.better-idea.org/icon?url=http://www.economist.com&size=70..120..200",
		"the-economist"},
		{"The Guardian (AU)",
		"https://icons.better-idea.org/icon?url=https://www.theguardian.com/au&size=70..120..200",
		"the-guardian-au"},
		{"The Guardian (UK)",
		"https://icons.better-idea.org/icon?url=https://www.theguardian.com/uk&size=70..120..200",
		"the-guardian-uk"},
		{"The Hindu",
		"https://icons.better-idea.org/icon?url=http://www.thehindu.com&size=70..120..200",
		"the-hindu"},
		{"The Huffington Post",
		"https://icons.better-idea.org/icon?url=http://www.huffingtonpost.com&size=70..120..200",
		"the-huffington-post"},
		{"The Lad Bible",
		"https://icons.better-idea.org/icon?url=http://www.theladbible.com&size=70..120..200",
		"the-lad-bible"},
		{"The New York Times",
		"https://icons.better-idea.org/icon?url=http://www.nytimes.com&size=70..120..200",
		"the-new-york-times"},
		{"The Next Web",
		"https://icons.better-idea.org/icon?url=http://thenextweb.com&size=70..120..200",
		"the-next-web"},
		{"The Sport Bible",
		"https://icons.better-idea.org/icon?url=http://www.thesportbible.com&size=70..120..200",
		"the-sport-bible"},
		{"The Telegraph",
		"https://icons.better-idea.org/icon?url=http://www.telegraph.co.uk&size=70..120..200",
		"the-telegraph"},
		{"The Times of India",
		"https://icons.better-idea.org/icon?url=http://timesofindia.indiatimes.com&size=70..120..200",
		"the-times-of-india"},
		{"The Verge",
		"https://icons.better-idea.org/icon?url=http://www.theverge.com&size=70..120..200",
		"the-verge"},
		{"The Wall Street Journal",
		"https://icons.better-idea.org/icon?url=http://www.wsj.com&size=70..120..200",
		"the-wall-street-journal"},
		{"The Washington Post",
		"https://icons.better-idea.org/icon?url=https://www.washingtonpost.com&size=70..120..200",
		"the-washington-post"},
		{"Time",
		"https://icons.better-idea.org/icon?url=http://time.com&size=70..120..200",
		"time"},
		{"USA Today",
		"https://icons.better-idea.org/icon?url=http://www.usatoday.com/news&size=70..120..200",
		"usa-today"},
		{"Wired.de",
		"https://icons.better-idea.org/icon?url=https://www.wired.de&size=70..120..200",
		"wired-de"},
		{"Wirtschafts Woche",
		"https://icons.better-idea.org/icon?url=http://www.wiwo.de&size=70..120..200",
		"wirtschafts-woche"},
	}
)
//////////////////////////////////////////////////////////////////////////////
//
// display news
// TODO: santize (html- and url-escape the arguments)
//       use a caching, resizing image proxy for the images.
//
//////////////////////////////////////////////////////////////////////////////
func fetchNews(newsSource string, c chan []Article) {
	// Note: I should be passing in category, language, and country parameters.
	newsRequestUrl := "https://newsapi.org/v1/articles"
	//newsRequestUrl += "?sortBy=latest"
	newsRequestUrl += "?apiKey=" + flags.newsAPIKey //1ff33b5f808b474384aa5fde75844e6b
	newsRequestUrl += "&source=" + newsSource //the-next-web&
	
	printVal("newsRequestUrl", newsRequestUrl)
	
	resp, err := http.Get(newsRequestUrl)
	check(err)
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	check(err)
	
	// Parse the News API json.
	var news News
	err = json.Unmarshal(body, &news)
	check(err)
	
	// News request returned an error.
	if news.Status != "ok" {
		fmt.Printf("Error fetching news from '%s': '%s'\n", newsSource, body)
		c <- []Article{}
		return
	}
	
	c <- news.Articles
	//close(c)
}
func newsHandler(w http.ResponseWriter, r *http.Request) {
	c := make(chan []Article)
	
	go fetchNews("bbc-news", c)
	go fetchNews("the-next-web", c)
	
	var articles []Article
	articles = <-c
	articles = append(articles, <-c...)
	
	printVal("artices", articles)
	
	//dest := make([]int, len(src))
	perm := rand.Perm(len(articles))
	//for i, v := range perm {
	//    dest[v] = src[i]
	//}
	
	articleArgs := make([]ArticleArg, len(articles))
	for i, _ := range articles {
		article := articles[perm[i]] // shuffle the article order (to mix between sources)
		
		// Copy the article information.
		articleArgs[i].Article		= article

		// Set the index
		articleArgs[i].Index = i + 1

		// Parse the hostname.
		u, err := url.Parse(article.Url)
		check(err)
		articleArgs[i].Host	= u.Host
	}	
	
	// Render the news articles.
	newsArgs := struct {
		PageArgs
		Articles	[]ArticleArg
	}{
		PageArgs: PageArgs{Title: "votezilla - News"},
		Articles: articleArgs,
	}
	executeTemplate(w, "news", newsArgs)
	
	//fmt.Fprintf(w, string(body))
	//fmt.Fprintf(w, "\n\nNews: %#v", news)
}
/*
A goroutine is a lightweight thread managed by the Go runtime.

go f(x, y, z)
starts a new goroutine running

f(x, y, z)

---

func sum(s []int, c chan int) {
	sum := 0
	for _, v := range s {
		sum += v
	}
	c <- sum // send sum to c
	
}

func main() {
	s := []int{7, 2, 8, -9, 4, 0}

	c := make(chan int)
	go sum(s[:len(s)/2], c)
	go sum(s[len(s)/2:], c)
	x, y := <-c, <-c // receive from c

	fmt.Println(x, y, x+y)
}

---

func fibonacci(n int, c chan int) {
	x, y := 0, 1
	for i := 0; i < n; i++ {
		c <- x
		x, y = y, x+y
	}
	close(c)
}

func main() {
	c := make(chan int, 10)
	go fibonacci(cap(c), c)
	for i := range c {
		fmt.Println(i)
	}
}
---


func fibonacci(c, quit chan int) {
	x, y := 0, 1
	for {
		select {
		case c <- x:
			x, y = y, x+y
		case <-quit:
			fmt.Println("quit")
			return
		}
	}
}

func main() {
	c := make(chan int)
	quit := make(chan int)
	go func() {
		for i := 0; i < 10; i++ {
			fmt.Println(<-c)
		}
		quit <- 0
	}()
	fibonacci(c, quit)
}
---

func main() {
	tick := time.Tick(100 * time.Millisecond)
	boom := time.After(500 * time.Millisecond)
	for {
		select {
		case <-tick:
			fmt.Println("tick.")
		case <-boom:
			fmt.Println("BOOM!")
			return
		default:
			fmt.Println("    .")
			time.Sleep(50 * time.Millisecond)
		}
	}
}
*/
///////////////////////////////////////////////////////////////////////////////
//
// display news sources
//
///////////////////////////////////////////////////////////////////////////////
func newsSourcesHandler(w http.ResponseWriter, r *http.Request) {
	newsSourcesArgs := struct {
		PageArgs
		NewsSources []NewsSource
	}{
		PageArgs: PageArgs{Title: "News Sources"},
		NewsSources: NewsSources,
	}
	fmt.Println("newsSourcesArgs: %#v\n", newsSourcesArgs)
	executeTemplate(w, "newsSources", newsSourcesArgs)	
}