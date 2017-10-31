package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"
)

// JSON-parsed format of an article.
type Article struct {
	Author			string
	Title			string
	Description		string
	Url				string
	UrlToImage		string
	PublishedAt		string
	// Custom parameters:
	NewsSourceId	string
	Host			string
	Category		string
	Language		string
	Country			string
}

var (
	// News source icons no longer part of API, so have to set manually.
	// TODO: move this data to a database.
	newsSourceIcons map[string]string = map[string]string{
		"abc-news-au": "https://icons.better-idea.org/icon?url=http://www.abc.net.au/news&size=70..120..200",
		"al-jazeera-english": "https://icons.better-idea.org/icon?url=http://www.aljazeera.com&size=70..120..200",
		"ars-technica": "https://icons.better-idea.org/icon?url=http://arstechnica.com&size=70..120..200",
		"associated-press": "https://icons.better-idea.org/icon?url=https://apnews.com/&size=70..120..200",
		"bbc-news": "https://icons.better-idea.org/icon?url=http://www.bbc.co.uk/news&size=70..120..200",
		"bbc-sport": "https://icons.better-idea.org/icon?url=http://www.bbc.co.uk/sport&size=70..120..200",
		"bloomberg": "https://icons.better-idea.org/icon?url=http://www.bloomberg.com&size=70..120..200",
		"breitbart-news": "https://icons.better-idea.org/icon?url=http://www.breitbart.com&size=70..120..200",
		"business-insider": "https://icons.better-idea.org/icon?url=http://www.businessinsider.com&size=70..120..200",
		"business-insider-uk": "https://icons.better-idea.org/icon?url=http://uk.businessinsider.com&size=70..120..200",
		"buzzfeed": "https://icons.better-idea.org/icon?url=https://www.buzzfeed.com&size=70..120..200",
		"cnbc": "https://icons.better-idea.org/icon?url=http://www.cnbc.com&size=70..120..200",
		"cnn": "https://icons.better-idea.org/icon?url=http://us.cnn.com&size=70..120..200",
		"daily-mail": "https://icons.better-idea.org/icon?url=http://www.dailymail.co.uk/home/index.html&size=70..120..200",
		"der-tagesspiegel": "https://icons.better-idea.org/icon?url=http://www.tagesspiegel.de&size=70..120..200",
		"die-zeit": "https://icons.better-idea.org/icon?url=http://www.zeit.de/index&size=70..120..200",
		"engadget": "https://icons.better-idea.org/icon?url=https://www.engadget.com&size=70..120..200",
		"entertainment-weekly": "https://icons.better-idea.org/icon?url=http://www.ew.com&size=70..120..200",
		"espn": "https://icons.better-idea.org/icon?url=http://espn.go.com&size=70..120..200",
		"espn-cric-info": "https://icons.better-idea.org/icon?url=http://www.espncricinfo.com/&size=70..120..200",
		"financial-times": "https://icons.better-idea.org/icon?url=http://www.ft.com/home/uk&size=70..120..200",
		"focus": "https://icons.better-idea.org/icon?url=http://www.focus.de&size=70..120..200",
		"football-italia": "https://icons.better-idea.org/icon?url=http://www.football-italia.net&size=70..120..200",
		"fortune": "https://icons.better-idea.org/icon?url=http://fortune.com&size=70..120..200",
		"four-four-two": "https://icons.better-idea.org/icon?url=http://www.fourfourtwo.com/news&size=70..120..200",
		"fox-sports": "https://icons.better-idea.org/icon?url=http://www.foxsports.com&size=70..120..200",
		"google-news": "https://icons.better-idea.org/icon?url=https://news.google.com&size=70..120..200",
		"gruenderszene": "https://icons.better-idea.org/icon?url=http://www.gruenderszene.de&size=70..120..200",
		"hacker-news": "https://icons.better-idea.org/icon?url=https://news.ycombinator.com&size=70..120..200",
		"handelsblatt": "https://icons.better-idea.org/icon?url=http://www.handelsblatt.com&size=70..120..200",
		"ign": "https://icons.better-idea.org/icon?url=http://www.ign.com&size=70..120..200",
		"independent": "https://icons.better-idea.org/icon?url=http://www.independent.co.uk&size=70..120..200",
		"mashable": "https://icons.better-idea.org/icon?url=http://mashable.com&size=70..120..200",
		"metro": "https://icons.better-idea.org/icon?url=http://metro.co.uk&size=70..120..200",
		"mirror": "https://icons.better-idea.org/icon?url=http://www.mirror.co.uk/&size=70..120..200",
		"mtv-news": "https://icons.better-idea.org/icon?url=http://www.mtv.com/news&size=70..120..200",
		"mtv-news-uk": "https://icons.better-idea.org/icon?url=http://www.mtv.co.uk/news&size=70..120..200",
		"national-geographic": "https://icons.better-idea.org/icon?url=http://news.nationalgeographic.com&size=70..120..200",
		"new-scientist": "https://icons.better-idea.org/icon?url=https://www.newscientist.com/section/news&size=70..120..200",
		"newsweek": "https://icons.better-idea.org/icon?url=http://www.newsweek.com&size=70..120..200",
		"new-york-magazine": "https://icons.better-idea.org/icon?url=http://nymag.com&size=70..120..200",
		"nfl-news": "https://icons.better-idea.org/icon?url=http://www.nfl.com/news&size=70..120..200",
		"polygon": "https://icons.better-idea.org/icon?url=http://www.polygon.com&size=70..120..200",
		"recode": "https://icons.better-idea.org/icon?url=http://www.recode.net&size=70..120..200",
		"reddit-r-all": "https://icons.better-idea.org/icon?url=https://www.reddit.com/r/all&size=70..120..200",
		"reuters": "https://icons.better-idea.org/icon?url=http://www.reuters.com&size=70..120..200",
		"spiegel-online": "https://icons.better-idea.org/icon?url=http://www.spiegel.de&size=70..120..200",
		"t3n": "https://icons.better-idea.org/icon?url=http://t3n.de&size=70..120..200",
		"talksport": "https://icons.better-idea.org/icon?url=http://talksport.com&size=70..120..200",
		"techcrunch": "https://icons.better-idea.org/icon?url=https://techcrunch.com&size=70..120..200",
		"techradar": "https://icons.better-idea.org/icon?url=http://www.techradar.com&size=70..120..200",
		"the-economist": "https://icons.better-idea.org/icon?url=http://www.economist.com&size=70..120..200",
		"the-guardian-au": "https://icons.better-idea.org/icon?url=https://www.theguardian.com/au&size=70..120..200",
		"the-guardian-uk": "https://icons.better-idea.org/icon?url=https://www.theguardian.com/uk&size=70..120..200",
		"the-hindu": "https://icons.better-idea.org/icon?url=http://www.thehindu.com&size=70..120..200",
		"the-huffington-post": "https://icons.better-idea.org/icon?url=http://www.huffingtonpost.com&size=70..120..200",
		"the-lad-bible": "https://icons.better-idea.org/icon?url=http://www.theladbible.com&size=70..120..200",
		"the-new-york-times": "https://icons.better-idea.org/icon?url=http://www.nytimes.com&size=70..120..200",
		"the-next-web": "https://icons.better-idea.org/icon?url=http://thenextweb.com&size=70..120..200",
		"the-sport-bible": "https://icons.better-idea.org/icon?url=http://www.thesportbible.com&size=70..120..200",
		"the-telegraph": "https://icons.better-idea.org/icon?url=http://www.telegraph.co.uk&size=70..120..200",
		"the-times-of-india": "https://icons.better-idea.org/icon?url=http://timesofindia.indiatimes.com&size=70..120..200",
		"the-verge": "https://icons.better-idea.org/icon?url=http://www.theverge.com&size=70..120..200",
		"the-wall-street-journal": "https://icons.better-idea.org/icon?url=http://www.wsj.com&size=70..120..200",
		"the-washington-post": "https://icons.better-idea.org/icon?url=https://www.washingtonpost.com&size=70..120..200",
		"time": "https://icons.better-idea.org/icon?url=http://time.com&size=70..120..200",
		"usa-today": "https://icons.better-idea.org/icon?url=http://www.usatoday.com/news&size=70..120..200",
		"wired-de": "https://icons.better-idea.org/icon?url=https://www.wired.de&size=70..120..200",
		"wirtschafts-woche": "https://icons.better-idea.org/icon?url=http://www.wiwo.de&size=70..120..200",
	}
)

//////////////////////////////////////////////////////////////////////////////
//
// fetches news sources
//
//////////////////////////////////////////////////////////////////////////////
func fetchNewsSources() bool {
	pr(ns_, "fetchNewsSources")
	
	// Note: I should be passing in category, language, and country parameters.
	newsRequestUrl := "https://newsapi.org/v1/sources"
	newsRequestUrl += "?apiKey=" + flags.newsAPIKey
	newsRequestUrl += "&language=en" // TODO: handle news source language selection.
	newsRequestUrl += "&country=us"  // TODO: handle news source country selection.
	
	prVal(ns_, "newsRequestUrl", newsRequestUrl)
	
	resp, err := httpGet(newsRequestUrl, 60.0)
	if err != nil {
		prVal(ns_, "fetchNewsSources request err", err)
		return false
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		prVal(ns_, "fetchNewsSources read err", err)
		return false
	}
	
	// Parse the News Sources json.
	var newsSourcesResp struct {
		Status	string
		Sources	[]NewsSource
	}
	err = json.Unmarshal(body, &newsSourcesResp)
	if err != nil {
		prVal(ns_, "fetchNewsSources unmarshall err", err)
		return false
	}
	
	// News request returned an error.
	if newsSourcesResp.Status != "ok" {
		prf(ns_, "Error fetching news sources: '%s'\n", body)
		return false
	}
	
	// Copy news source data to newsSources, and assign icon.
	newsSources = NewsSources{}
	for _, newsSource := range newsSourcesResp.Sources {
		newsSource.Icon = newsSourceIcons[newsSource.Id]
		
		newsSources[newsSource.Id] = newsSource
	}
	
	//prVal(ns_, "newsSources", newsSources)
	return true
}

//////////////////////////////////////////////////////////////////////////////
//
// fetches news articles from a single source
//
//////////////////////////////////////////////////////////////////////////////
func fetchNews(newsSource string, c chan []Article) {
	pr(ns_, "fetchNews")
	
	// Site: https://newsapi.org/
	// Note: I should be passing in category, language, and country parameters.
	newsRequestUrl := "https://newsapi.org/v1/articles"
	//newsRequestUrl += "?sortBy=latest" // top, latest, or popular
	newsRequestUrl += "?apiKey=" + flags.newsAPIKey
	newsRequestUrl += "&source=" + newsSource
	
	prVal(ns_, "newsRequestUrl", newsRequestUrl)
	
	resp, err := httpGet(newsRequestUrl, 60.0)
	if err != nil {
		prf(ns_, "Error fetching news from '%s': '%s'\n", newsSource, err)
		c <- []Article{}
		return
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		prf(ns_, "Error fetching news from '%s': '%s'\n", newsSource, err)
		c <- []Article{}
		return
	}
	
	// Parse the News API json.
	var news struct {
		Status		string
		Source		string
		SortBy		string
		Articles	[]Article
	}
	err = json.Unmarshal(body, &news)
	if err != nil {
		prf(ns_, "Error fetching news from '%s': '%s' '%s'\n", newsSource, err, body)
		c <- []Article{}
		return
	}
	
	// News request returned an error.
	if news.Status != "ok" {
		prf(ns_, "Error fetching news from '%s': '%s'\n", newsSource, body)
		c <- []Article{}
		return
	}

	for i := 0; i < len(news.Articles); i++ {
		// Set the news source
		news.Articles[i].NewsSourceId = newsSource
				
		// Set the category, language, and country.
		news.Articles[i].Category = newsSources[newsSource].Category
		news.Articles[i].Language = newsSources[newsSource].Language
		news.Articles[i].Country  = newsSources[newsSource].Country
	}
	
	c <- news.Articles
	return
}

//////////////////////////////////////////////////////////////////////////////
//
// news server - On startup, and every 5 minutes, fetches the latest news, then
//				 adds it to votezilla.NewsPost table.
//
//////////////////////////////////////////////////////////////////////////////
func NewsServer() {
	var newArticles []Article

	pr(ns_, "========================================")
	pr(ns_, "======== STARTING NEWS SERVER ==========")
	pr(ns_, "========================================\n")

	for {
		pr(ns_, "========================================")
		pr(ns_, "============ FETCHING NEWS =============")
		pr(ns_, "========================================\n")

		if flags.offlineNews != "" {
			pr(ns_, "Fetching offline news articles and sources")
			newArticles = []Article{Article{Author:"MICHAEL BALSAMO and BRIAN MELLEY", Title:"Thousands mourn slain officer as Las Vegas probe goes on", Description:"LAS VEGAS (AP) Las Vegas gunman Stephen Paddock booked rooms over other music festivals in the months before opening fire on a country music festival, authorities said, while thousands came out to mourn a police officer who was one of the 58 people he killed. Paddock booked rooms overlooking the Lollapalooza festival in Chicago in August and the Life Is Beautiful show near the Vegas Strip in late September, according to authorities reconstructing his movements before he undertook the deadliest mass shooting in modern U.S. history.", Url:"https://apnews.com/122b18f2ec0c448c80ced46fd1c58ba6", UrlToImage:"https://storage.googleapis.com/afs-prod/media/media:be66c3e33f2943dbbc55c9740052f19e/3000.jpeg", PublishedAt:"2017-10-06T07:05:57Z", NewsSourceId:"associated-press", Host:"apnews.com", Category:"general", Language:"en", Country:"us"}, Article{Author:"ANITA SNOW", Title:"1st firefighters at Vegas massacre came across it by chance", Description:"LAS VEGAS (AP) Fire engineer Brian Emery was driving his station's engine back from a call for a minor car crash when hundreds of hysterical people began swarming the vehicle near an outdoor country music festival in Las Vegas. \"Then, suddenly, we heard automatic gunfire,\" Emery recalled Thursday after his crew became the first to respond to the deadliest shooting in modern American history. It was pure coincidence that the Clark County Fire Department crew members on Engine 11 Emery, team leader Capt. Ken O'Shaughnessy and two firefighters, including a rookie were the first on-duty emergency personnel to arrive Sunday night.", Url:"https://apnews.com/e6961e6e47fd44afa285ae1bce5ddaed", UrlToImage:"https://storage.googleapis.com/afs-prod/media/media:25824cb91f884425b665a39b528abb72/1470.jpeg", PublishedAt:"2017-10-06T07:09:40Z", NewsSourceId:"associated-press", Host:"apnews.com", Category:"general", Language:"en", Country:"us"}, Article{Author:"ERICA WERNER", Title:"Effort to restrict 'bump stock' draws unlikely supporters", Description:"WASHINGTON (AP) The National Rifle Association have joined the Trump administration and top congressional Republicans in a swift and surprising embrace of a restriction on Americans' guns, though a narrow one: to regulate the \"bump stock\" devices the Las Vegas shooter apparently used to horrifically lethal effect. The devices, originally intended to help people with disabilities, fit over the stock and grip of a semi-automatic rifle and allow the weapon to fire continuously, some 400 to 800 rounds in a single minute. Bump stocks were found among the gunman's weapons and explain why victims in Las Vegas heard what sounded like automatic-weapons fire as the shooter rained bullets from a casino high-rise, slaughtering 58 people in a concert below and wounding hundreds more.", Url:"https://apnews.com/94d04219df8a4e37b223a3473c165dab", UrlToImage:"https://storage.googleapis.com/afs-prod/media/media:e547f82bd81f4b0ba01a60a9e6d708ed/3000.jpeg", PublishedAt:"2017-10-06T07:23:58Z", NewsSourceId:"associated-press", Host:"apnews.com", Category:"general", Language:"en", Country:"us"}, Article{Author:"JILL COLVIN", Title:"Trump, during photo shoot, talks of 'calm before the storm'", Description:"WASHINGTON (AP) President Donald Trump delivered a foreboding message Thursday night, telling reporters as he posed for photos with his senior military leaders that this might be \"the calm before the storm.\" White House reporters were summoned suddenly Thursday evening and told the president had decided he wanted the press to document a dinner he was holding with the military leaders and their wives. Reporters were led hastily to the grand State Dining Room, where they walked into a scene of the president, his highest-ranking military aides and their wives posing for a group photo. The cameras clicked and they smiled. A joke was made about someone's face being tired. Live classical music played.", Url:"https://apnews.com/b65b8810738b457a81adec4be7006a65", UrlToImage:"https://storage.googleapis.com/afs-prod/media/media:ab3fcfa0bffe422286167b35db76b5b0/3000.jpeg", PublishedAt:"2017-10-06T00:55:13Z", NewsSourceId:"associated-press", Host:"apnews.com", Category:"general", Language:"en", Country:"us"}, Article{Author:"JON GAMBRELL", Title:"APNewsBreak: US military halts exercises over Qatar crisis", Description:"DUBAI, United Arab Emirates (AP) The U.S. military has halted some exercises with its Gulf Arab allies over the ongoing diplomatic crisis targeting Qatar, trying to use its influence to end the monthslong dispute, authorities told The Associated Press on Friday. While offering few details, the acknowledgement by the U.S. military's Central Command shows the concern it has over the conflict gripping the Gulf, home to the U.S. Navy's 5th Fleet and crucial bases for its campaign against the Islamic State group in Iraq and Syria, as well as the war in Afghanistan.", Url:"https://apnews.com/33d75eaedbdd4b178e9025ea8a0edc83", UrlToImage:"https://storage.googleapis.com/afs-prod/media/media:e1de87d8faec4ac7b6c4e966f4106fc2/3000.jpeg", PublishedAt:"2017-10-06T07:13:28Z", NewsSourceId:"associated-press", Host:"apnews.com", Category:"general", Language:"en", Country:"us"}, Article{Author:"LINDSEY BAHR and JAKE COYLE", Title:"After accusations Hollywood asks: Is Harvey Weinstein done?", Description:"LOS ANGELES (AP) Accepting the Golden Globe best actress award in 2012 for \"The Iron Lady,\" Meryl Streep took a moment to thank the almighty \"God, Harvey Weinstein.\" For decades, Weinstein has held a lofty position in Hollywood as one of the industry's most powerful figures an old-school, larger-than-life movie mogul who was never shy about throwing his weight around. \"The Punisher. Old Testament, I guess,\" Streep added that night to laughter and applause.", Url:"https://apnews.com/134f752397d04347b9edaa4c7e4bef3b", UrlToImage:"https://storage.googleapis.com/afs-prod/media/media:317143f5fd83441d92a374f1beeede6b/3000.jpeg", PublishedAt:"2017-10-06T07:12:15Z", NewsSourceId:"associated-press", Host:"apnews.com", Category:"general", Language:"en", Country:"us"}, Article{Author:"LUIS MANUEL GALEANO", Title:"Nate takes aim at Mexico, US after dousing Central America", Description:"MANAGUA, Nicaragua (AP) Tropical Storm Nate roared toward Mexico's Yucatan Peninsula after drenching Central America in rain that was blamed for at least 22 deaths, and forecasters said it could reach the U.S. Gulf Coast as a hurricane over the weekend. Louisiana officials declared a state of emergency and ordered some people to evacuate coastal areas and barrier islands ahead of its expected landfall early Sunday, and evacuations began at some offshore oil platforms in the Gulf.", Url:"https://apnews.com/825dfdada0c043da828d3efb422a1638", UrlToImage:"https://storage.googleapis.com/afs-prod/media/media:1eb921d5959e4d8caf38108eb3137236/3000.jpeg", PublishedAt:"2017-10-06T04:20:56Z", NewsSourceId:"associated-press", Host:"apnews.com", Category:"general", Language:"en", Country:"us"}}
			newsSources = NewsSources{"buzzfeed":NewsSource{Id:"buzzfeed", Name:"Buzzfeed", Description:"BuzzFeed is a cross-platform, global network for news and entertainment that generates seven billion views each month.", Url:"https://www.buzzfeed.com", Category:"entertainment", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=https://www.buzzfeed.com&size=70..120..200"}, "national-geographic":NewsSource{Id:"national-geographic", Name:"National Geographic", Description:"Reporting our world daily: original nature and science news from National Geographic.", Url:"http://news.nationalgeographic.com", Category:"science-and-nature", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=http://news.nationalgeographic.com&size=70..120..200"}, "newsweek":NewsSource{Id:"newsweek", Name:"Newsweek", Description:"Newsweek provides in-depth analysis, news and opinion about international issues, technology, business, culture and politics.", Url:"http://www.newsweek.com", Category:"general", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://www.newsweek.com&size=70..120..200"}, "the-next-web":NewsSource{Id:"the-next-web", Name:"The Next Web", Description:"The Next Web is one of the world's largest online publications that delivers an international perspective on the latest news about Internet technology, business and culture.", Url:"http://thenextweb.com", Category:"technology", Language:"en", Country:"us", SortBysAvailable:[]string{"latest"}, Icon:"https://icons.better-idea.org/icon?url=http://thenextweb.com&size=70..120..200"}, "hacker-news":NewsSource{Id:"hacker-news", Name:"Hacker News", Description:"Hacker News is a social news website focusing on computer science and entrepreneurship. It is run by Paul Graham's investment fund and startup incubator, Y Combinator. In general, content that can be submitted is defined as \"anything that gratifies one's intellectual curiosity\".", Url:"https://news.ycombinator.com", Category:"technology", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=https://news.ycombinator.com&size=70..120..200"}, "new-scientist":NewsSource{Id:"new-scientist", Name:"New Scientist", Description:"Breaking science and technology news from around the world. Exclusive stories and expert analysis on space, technology, health, physics, life and Earth.", Url:"https://www.newscientist.com/section/news", Category:"science-and-nature", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=https://www.newscientist.com/section/news&size=70..120..200"}, "bloomberg":NewsSource{Id:"bloomberg", Name:"Bloomberg", Description:"Bloomberg delivers business and markets news, data, analysis, and video to the world, featuring stories from Businessweek and Bloomberg News.", Url:"http://www.bloomberg.com", Category:"business", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=http://www.bloomberg.com&size=70..120..200"}, "breitbart-news":NewsSource{Id:"breitbart-news", Name:"Breitbart News", Description:"Syndicated news and opinion website providing continuously updated headlines to top news and analysis sources.", Url:"http://www.breitbart.com", Category:"politics", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://www.breitbart.com&size=70..120..200"}, "engadget":NewsSource{Id:"engadget", Name:"Engadget", Description:"Engadget is a web magazine with obsessive daily coverage of everything new in gadgets and consumer electronics.", Url:"https://www.engadget.com", Category:"technology", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=https://www.engadget.com&size=70..120..200"}, "google-news":NewsSource{Id:"google-news", Name:"Google News", Description:"Comprehensive, up-to-date news coverage, aggregated from sources all over the world by Google News.", Url:"https://news.google.com", Category:"general", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=https://news.google.com&size=70..120..200"}, "nfl-news":NewsSource{Id:"nfl-news", Name:"NFL News", Description:"The official source for NFL news, schedules, stats, scores and more.", Url:"http://www.nfl.com/news", Category:"sport", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://www.nfl.com/news&size=70..120..200"}, "cnn":NewsSource{Id:"cnn", Name:"CNN", Description:"View the latest news and breaking news today for U.S., world, weather, entertainment, politics and health at CNN", Url:"http://us.cnn.com", Category:"general", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=http://us.cnn.com&size=70..120..200"}, "espn-cric-info":NewsSource{Id:"espn-cric-info", Name:"ESPN Cric Info", Description:"ESPN Cricinfo provides the most comprehensive cricket coverage available including live ball-by-ball commentary, news, unparalleled statistics, quality editorial comment and analysis.", Url:"http://www.espncricinfo.com/", Category:"sport", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://www.espncricinfo.com/&size=70..120..200"}, "fortune":NewsSource{Id:"fortune", Name:"Fortune", Description:"Fortune 500 Daily and Breaking Business News", Url:"http://fortune.com", Category:"business", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://fortune.com&size=70..120..200"}, "reuters":NewsSource{Id:"reuters", Name:"Reuters", Description:"Reuters.com brings you the latest news from around the world, covering breaking news in business, politics, entertainment, technology,video and pictures.", Url:"http://www.reuters.com", Category:"general", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://www.reuters.com&size=70..120..200"}, "cnbc":NewsSource{Id:"cnbc", Name:"CNBC", Description:"Get latest business news on stock markets, financial & earnings on CNBC. View world markets streaming charts & video; check stock tickers and quotes.", Url:"http://www.cnbc.com", Category:"business", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=http://www.cnbc.com&size=70..120..200"}, "espn":NewsSource{Id:"espn", Name:"ESPN", Description:"ESPN has up-to-the-minute sports news coverage, scores, highlights and commentary for NFL, MLB, NBA, College Football, NCAA Basketball and more.", Url:"http://espn.go.com", Category:"sport", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=http://espn.go.com&size=70..120..200"}, "techradar":NewsSource{Id:"techradar", Name:"TechRadar", Description:"The latest technology news and reviews, covering computing, home entertainment systems, gadgets and more.", Url:"http://www.techradar.com", Category:"technology", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://www.techradar.com&size=70..120..200"}, "the-verge":NewsSource{Id:"the-verge", Name:"The Verge", Description:"The Verge covers the intersection of technology, science, art, and culture.", Url:"http://www.theverge.com", Category:"technology", Language:"en", Country:"us",SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://www.theverge.com&size=70..120..200"}, "time":NewsSource{Id:"time", Name:"Time", Description:"Breaking news and analysis from TIME.com. Politics, world news, photos, video, tech reviews, health, science and entertainment news.", Url:"http://time.com", Category:"general", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://time.com&size=70..120..200"}, "reddit-r-all":NewsSource{Id:"reddit-r-all", Name:"Reddit /r/all", Description:"Reddit is an entertainment, social news networking service, and news website. Reddit's registered communitymembers can submit content, such as text posts or direct links.", Url:"https://www.reddit.com/r/all", Category:"general", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=https://www.reddit.com/r/all&size=70..120..200"}, "the-wall-street-journal":NewsSource{Id:"the-wall-street-journal", Name:"The Wall Street Journal", Description:"WSJ online coverage of breaking news and current headlines from the US and around the world. Top stories, photos, videos, detailed analysis and in-depth reporting.", Url:"http://www.wsj.com", Category:"business", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=http://www.wsj.com&size=70..120..200"}, "the-washington-post":NewsSource{Id:"the-washington-post", Name:"The Washington Post", Description:"Breaking news and analysis on politics, business, world national news, entertainment more. In-depth DC, Virginia, Maryland news coverage including traffic, weather, crime, education, restaurant reviews and more.", Url:"https://www.washingtonpost.com", Category:"general", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=https://www.washingtonpost.com&size=70..120..200"}, "associated-press":NewsSource{Id:"associated-press", Name:"Associated Press", Description:"The AP delivers in-depth coverage on the international, politics, lifestyle, business, and entertainment news.", Url:"https://apnews.com/", Category:"general", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=https://apnews.com/&size=70..120..200"}, "entertainment-weekly":NewsSource{Id:"entertainment-weekly", Name:"Entertainment Weekly", Description:"Online version of the print magazine includes entertainment news, interviews, reviews of music, film,TV and books, and a special area for magazine subscribers.", Url:"http://www.ew.com", Category:"entertainment", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=http://www.ew.com&size=70..120..200"}, "mtv-news":NewsSource{Id:"mtv-news", Name:"MTV News", Description:"The ultimate news source formusic, celebrity, entertainment, movies, and current events on the web. It's pop culture on steroids.", Url:"http://www.mtv.com/news", Category:"music", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://www.mtv.com/news&size=70..120..200"}, "new-york-magazine":NewsSource{Id:"new-york-magazine", Name:"New York Magazine", Description:"NYMAG and New York magazine cover the new, the undiscovered, thenext in politics, culture, food, fashion, and behavior nationally, through a New York lens.", Url:"http://nymag.com", Category:"general", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://nymag.com&size=70..120..200"}, "polygon":NewsSource{Id:"polygon", Name:"Polygon", Description:"Polygon is a gaming website in partnership with Vox Media. Our culture focused site covers games, their creators, the fans, trending stories and entertainment news.", Url:"http://www.polygon.com", Category:"gaming", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=http://www.polygon.com&size=70..120..200"}, "recode":NewsSource{Id:"recode", Name:"Recode", Description:"Get the latest independent tech news, reviews and analysis from Recode with the most informed and respected journalists in technology and media.", Url:"http://www.recode.net", Category:"technology", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=http://www.recode.net&size=70..120..200"}, "techcrunch":NewsSource{Id:"techcrunch", Name:"TechCrunch", Description:"TechCrunch is a leading technology media property, dedicated to obsessively profiling startups, reviewing new Internet products, and breaking tech news.", Url:"https://techcrunch.com", Category:"technology", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=https://techcrunch.com&size=70..120..200"}, "the-new-york-times":NewsSource{Id:"the-new-york-times", Name:"The New York Times", Description:"The New York Times: Find breaking news, multimedia, reviews & opinion on Washington, business, sports,movies, travel, books, jobs, education, real estate, cars & more at nytimes.com.", Url:"http://www.nytimes.com", Category:"general", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=http://www.nytimes.com&size=70..120..200"}, "al-jazeera-english":NewsSource{Id:"al-jazeera-english", Name:"Al Jazeera English", Description:"News, analysis from the Middle East and worldwide, multimedia and interactives, opinions, documentaries, podcasts, long reads and broadcast schedule.", Url:"http://www.aljazeera.com", Category:"general", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://www.aljazeera.com&size=70..120..200"}, "ars-technica":NewsSource{Id:"ars-technica", Name:"Ars Technica", Description:"The PC enthusiast's resource. Power users and the tools they love, without computing religion.", Url:"http://arstechnica.com", Category:"technology", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://arstechnica.com&size=70..120..200"}, "business-insider":NewsSource{Id:"business-insider", Name:"Business Insider", Description:"Business Insider is a fast-growing business site with deep financial, media, tech, and other industry verticals. Launched in 2007, the site is now the largest business news site on the web.", Url:"http://www.businessinsider.com", Category:"business", Language:"en", Country:"us", SortBysAvailable:[]string{"top","latest"}, Icon:"https://icons.better-idea.org/icon?url=http://www.businessinsider.com&size=70..120..200"}, "ign":NewsSource{Id:"ign", Name:"IGN", Description:"IGN is your site for Xbox One, PS4, PC, Wii-U, Xbox 360, PS3, Wii, 3DS, PSVita and iPhone games with expert reviews, news, previews, trailers, cheat codes, wiki guides and walkthroughs.", Url:"http://www.ign.com", Category:"gaming", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://www.ign.com&size=70..120..200"}, "mashable":NewsSource{Id:"mashable", Name:"Mashable", Description:"Mashable is a global, multi-platform media and entertainment company.", Url:"http://mashable.com", Category:"entertainment", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://mashable.com&size=70..120..200"}, "fox-sports":NewsSource{Id:"fox-sports", Name:"Fox Sports", Description:"Find live scores, player and team news, videos, rumors, stats, standings, schedules and fantasy games on FOX Sports.", Url:"http://www.foxsports.com", Category:"sport", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://www.foxsports.com&size=70..120..200"}, "the-huffington-post":NewsSource{Id:"the-huffington-post", Name:"The Huffington Post", Description:"The Huffington Post is a politically liberal American online news aggregator and blog that has both localized and international editions founded by Arianna Huffington, Kenneth Lerer, Andrew Breitbart, and Jonah Peretti, featuring columnists.", Url:"http://www.huffingtonpost.com", Category:"general", Language:"en", Country:"us", SortBysAvailable:[]string{"top"}, Icon:"https://icons.better-idea.org/icon?url=http://www.huffingtonpost.com&size=70..120..200"}, "usa-today":NewsSource{Id:"usa-today", Name:"USA Today", Description:"Get the latest national, international, and political news at USATODAY.com.", Url:"http://www.usatoday.com/news", Category:"general", Language:"en", Country:"us", SortBysAvailable:[]string{"top", "latest"}, Icon:"https://icons.better-idea.org/icon?url=http://www.usatoday.com/news&size=70..120..200"}}
		} else {
			pr(ns_, "Fetching news sources")
			ok := fetchNewsSources()
			if !ok {
				pr(ns_, "Error: Failed to fetch news sources.  Probably Internet connectivity issues.  Trying again in 5 minutes.")

				time.Sleep(5 * time.Minute)
				continue
			}

			c := make(chan []Article)

			timeout := time.After(60 * time.Second)

			prVal(ns_, "len(newsSources)", len(newsSources))

			// Fetch articles.
			for _, newsSource := range newsSources {
				prVal(ns_, "Fetching article from", newsSource.Id)

				go fetchNews(newsSource.Id, c)
			}

			newArticles = []Article{}
			numSourcesFetched := 0
			fetchNewsLoop: for {
				select {
					case newArticlesFetched := <-c:
						newArticles = append(newArticles, newArticlesFetched...)
						numSourcesFetched++

						prVal(ns_, "New articles fetched", numSourcesFetched)
						
						if numSourcesFetched == len(newsSources) {
							pr(ns_, "Fetched all news sources!")
							break fetchNewsLoop
						}
					case <- timeout:
						pr(ns_, "Timeout!")
						break fetchNewsLoop
				}
			}
			
			if len(newArticles) < 50 {
				pr(ns_, "Error: Failed to fetch all the news.  Probably Internet connectivity issues.  Will populate the database anyways.")
				
				time.Sleep(5 * time.Minute)
				continue
			}
		}
				
		prVal(ns_, "len(newArticles)", len(newArticles))
	
		// Insert the news articles all in one query.
		sqlStr := `INSERT INTO votezilla.NewsPost(
				     UserId, Title, LinkURL, UrlToImage,
				     Description, PublishedAt, NewsSourceId, Category, Language, Country)
				   VALUES`
		vals := []interface{}{}
		
		vals = append(vals, -1) // $1 = UserId = -1
		
		argId := 2 // arguments start at $2
		for _, a := range newArticles {	
			sqlStr += fmt.Sprintf("($1::bigint,$%d,$%d,$%d,$%d,$%d::timestamptz,$%d,$%d,$%d,$%d),", 
				argId, argId+1, argId+2, argId+3, argId+4, argId+5, argId+6, argId+7, argId+8)
			argId += 9
			
			// Null PublishedAt causes uniqueness problems, so use zero time as a replacement in this case.
			publishedAt := a.PublishedAt
			if len(publishedAt) == 0 {
				publishedAt = "epoch" //"1970-01-01 00:00:00" // January 1, year 1 00:00:00 UTC.
			}
			
			vals = append(vals, 
				a.Title, a.Url, a.UrlToImage,
				a.Description, 
				publishedAt, 
				a.NewsSourceId, a.Category, a.Language, a.Country)	
		}
		//trim the last ',', add a trailing ';'
		sqlStr = strings.TrimSuffix(sqlStr, ",")
		
		// Do not insert duplicate news articles.
		sqlStr += " ON CONFLICT DO NOTHING"
		
		sqlStr += ";"
		
		prVal(ns_, "sqlStr", sqlStr)
		
		DbExec(sqlStr, vals...)
		
		//TODO: Remove duplicate news articles.
		
		DbTrackOpenConnections()

		pr(ns_, "Sleeping 5 minutes")
		time.Sleep(5 * time.Minute)
	}
}

/*
Various tests for, and eliminations of, duplicate news data:

select publishedat, title, count(*) from votezilla.newspost group by 1, 2 having count(*) > 1;

SELECT PublishedAt, Title, min(ctid)
FROM votezilla.NewsPost
GROUP BY 1, 2
HAVING count(*) > 1
WITH x AS (
	SELECT PublishedAt, Title, min(ctid)
	FROM votezilla.NewsPost
	GROUP BY 1, 2
	HAVING count(*) > 1
)
DELETE FROM votezilla.NewsPost n
USING x
WHERE n.PublishedAt = x.PublishedAt AND n.Title = x.Title
AND n.ctid <> x.min
RETURNING *;
*/